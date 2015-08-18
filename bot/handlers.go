package bot

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/goxforce"
	"github.com/slavikm/govt"
)

const (
	numOfPositivesToConvict = 3
	xfeScoreToConvict       = 3
)

// Worker reads messages from the queue and does the actual work
type Worker struct {
	q    queue.Queue
	c    chan *domain.WorkRequest
	r    repo.Repo
	xfe  *goxforce.Client
	vt   *govt.Client
	clam *clamEngine
}

// NewWorker that loads work messages from the queue
func NewWorker(r repo.Repo, q queue.Queue) (*Worker, error) {
	xfe, err := goxforce.New(goxforce.SetErrorLog(log.New(conf.LogWriter, "XFE:", log.Lshortfile)))
	if err != nil {
		return nil, err
	}
	vt, err := govt.New(govt.SetApikey(conf.Options.VT), govt.SetErrorLog(log.New(os.Stderr, "VT:", log.Lshortfile)))
	if err != nil {
		return nil, err
	}
	clam, err := newClamEngine()
	if err != nil {
		return nil, err
	}
	return &Worker{
		r:    r,
		q:    q,
		c:    make(chan *domain.WorkRequest, runtime.NumCPU()),
		xfe:  xfe,
		vt:   vt,
		clam: clam,
	}, nil
}

func (w *Worker) handle() {
	for msg := range w.c {
		if msg == nil {
			w.clam.close()
			return
		}
		if msg.ReplyQueue == "" {
			logrus.Warnf("Got message without a reply queue destination %+v\n", msg)
			continue
		}
		reply := &domain.WorkReply{Context: msg.Context, MessageID: msg.MessageID}
		switch msg.Type {
		case "message":
			if strings.Contains(msg.Text, "<http") {
				w.handleURL(msg.Text, reply)
			}
			if ip := ipReg.FindString(msg.Text); ip != "" {
				w.handleIP(ip, reply)
			}
			if hash := md5Reg.FindString(msg.Text); hash != "" {
				w.handleMD5(hash, reply)
			}
		case "file":
			w.handleFile(msg, reply)
		}
		if err := w.q.PushWorkReply(msg.ReplyQueue, reply); err != nil {
			logrus.Warnf("Error pushing message to reply queue %+v - %v\n", msg, err)
		}
	}
}

// Start the dedup process. To stop, just close the queue.
func (w *Worker) Start() {
	// Right now, just use the numebr of CPUs
	for i := 0; i < runtime.NumCPU(); i++ {
		go w.handle()
	}
	for {
		msg, err := w.q.PopWork(0)
		if err != nil || msg == nil {
			logrus.Info("Stoping WorkManager process")
			close(w.c)
			return
		}
		w.c <- msg
	}
}

func contextFromMap(c map[string]interface{}) *domain.Context {
	return &domain.Context{
		Team:         c["team"].(string),
		User:         c["user"].(string),
		OriginalUser: c["original_user"].(string),
		Channel:      c["channel"].(string),
	}
}

// GetContext from a message based on actual type
func GetContext(context interface{}) (*domain.Context, error) {
	switch c := context.(type) {
	case *domain.Context:
		// Hack to duplicate the context so if we are using channels not to override it
		cx := *c
		return &cx, nil
	case map[string]interface{}:
		return contextFromMap(c), nil
	default:
		return nil, errors.New("Unknown context")
	}
}

func (w *Worker) handleURL(text string, reply *domain.WorkReply) {
	start := strings.Index(text, "<http")
	end := strings.Index(text[start:], ">")
	if end > 0 {
		end = end + start
		filter := strings.Index(text[start:end], "|")
		if filter > 0 {
			end = start + filter
		}
		url := text[start+1 : end]
		logrus.Debugf("URL found - %s\n", url)
		reply.URL.Details = url
		reply.Type |= domain.ReplyTypeURL
		// Do the network commands in parallel
		c := make(chan int, 2)
		go func() {
			urlResp, err := w.xfe.URL(url)
			if err != nil {
				// Small hack - see if the URL was not found
				if strings.Contains(err.Error(), "404") {
					reply.URL.XFE.NotFound = true
				} else {
					reply.URL.XFE.Error = err.Error()
				}
			} else {
				reply.URL.XFE.URLDetails = urlResp.Result
			}
			resolve, err := w.xfe.Resolve(url)
			if err == nil {
				reply.URL.XFE.Resolve = *resolve
			}
			c <- 1
		}()
		go func() {
			vtResp, err := w.vt.GetUrlReport(url)
			if err != nil {
				reply.URL.VT.Error = err.Error()
			} else {
				reply.URL.VT.URLReport = *vtResp
			}
			c <- 1
		}()
		for i := 0; i < 2; i++ {
			<-c
		}
		if reply.URL.XFE.URLDetails.Score > xfeScoreToConvict || reply.URL.VT.URLReport.Positives > numOfPositivesToConvict {
			// This is known bad scenario
			reply.URL.Result = domain.ResultDirty
		} else if !reply.MD5.XFE.NotFound || reply.MD5.VT.FileReport.ResponseCode == 1 {
			// At least one of reputation services found this to be known good
			// Keep the default
			reply.URL.Result = domain.ResultClean
		}
	}
}

func (w *Worker) handleIP(ip string, reply *domain.WorkReply) {
	reply.IP.Details = ip
	reply.Type |= domain.ReplyTypeIP
	// First, let's check if IP is globally unicast addressable and is public
	ipData := net.ParseIP(ip)
	ipv4 := ipData.To4()
	if ipv4 == nil {
		// If not IPv4 then return - by default it will be marked clean
		return
	}
	if !ipv4.IsGlobalUnicast() {
		// If not global unicast ignore
		return
	}
	// Private networks
	if ipv4[0] == 10 || ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 || ipv4[0] == 192 && ipv4[1] == 168 {
		return
	}
	c := make(chan int, 2)
	go func() {
		ipResp, err := w.xfe.IPR(ip)
		if err != nil {
			// Small hack - see if the URL was not found
			if strings.Contains(err.Error(), "404") {
				reply.IP.XFE.NotFound = true
			} else {
				reply.IP.XFE.Error = err.Error()
			}
		} else {
			reply.IP.XFE.IPReputation = *ipResp
		}
		c <- 1
	}()
	go func() {
		vtResp, err := w.vt.GetIpReport(ip)
		if err != nil {
			reply.IP.VT.Error = err.Error()
		} else {
			reply.IP.VT.IPReport = *vtResp
		}
		c <- 1
	}()
	for i := 0; i < 2; i++ {
		<-c
	}
	var vtPositives uint16
	for i := range reply.IP.VT.IPReport.DetectedUrls {
		if reply.IP.VT.IPReport.DetectedUrls[i].Positives > vtPositives {
			vtPositives = reply.IP.VT.IPReport.DetectedUrls[i].Positives
		}
	}
	reply.IP.Result = domain.ResultUnknown
	if reply.IP.XFE.IPReputation.Score > xfeScoreToConvict || vtPositives > numOfPositivesToConvict {
		// This is known bad scenario
		reply.IP.Result = domain.ResultDirty
	} else if !reply.MD5.XFE.NotFound || reply.MD5.VT.FileReport.ResponseCode == 1 {
		// At least one of reputation services found this to be known good
		// Keep the default
		reply.IP.Result = domain.ResultClean
	}
}

func (w *Worker) handleMD5(md5 string, reply *domain.WorkReply) {
	reply.Type |= domain.ReplyTypeMD5
	reply.MD5.Details = md5
	c := make(chan int, 2)
	go func() {
		md5Resp, err := w.xfe.MalwareDetails(md5)
		if err != nil {
			// Small hack - see if the file was not found
			if strings.Contains(err.Error(), "404") {
				reply.MD5.XFE.NotFound = true
			} else {
				reply.MD5.XFE.Error = err.Error()
			}
		} else {
			reply.MD5.XFE.Malware = md5Resp.Malware
		}
		c <- 1
	}()
	go func() {
		vtResp, err := w.vt.GetFileReport(md5)
		if err != nil {
			reply.MD5.VT.Error = err.Error()
		} else {
			reply.MD5.VT.FileReport = *vtResp
		}
		c <- 1
	}()
	for i := 0; i < 2; i++ {
		<-c
	}
	reply.MD5.Result = domain.ResultUnknown
	if len(reply.MD5.XFE.Malware.Family) > 0 || reply.MD5.VT.FileReport.Positives > numOfPositivesToConvict {
		// This is known bad scenario
		reply.MD5.Result = domain.ResultDirty
	} else if !reply.MD5.XFE.NotFound || reply.MD5.VT.FileReport.ResponseCode == 1 {
		// At least one of reputation services found this to be known good
		// Keep the default
		reply.MD5.Result = domain.ResultClean
	}
}

func (w *Worker) handleFile(request *domain.WorkRequest, reply *domain.WorkReply) {
	reply.Type |= domain.ReplyTypeFile
	reply.File.Details = request.File
	if request.File.Size > 30*1024*1024 {
		logrus.Infof("File %s is bigger than 30M, skipping\n", request.File.Name)
		reply.File.FileTooLarge = true
		return
	}
	hash := md5.New()
	resp, err := http.Get(request.File.URL)
	if err != nil {
		logrus.Errorf("Unable to download file - %v\n", err)
		return
	}
	defer resp.Body.Close()
	buf := &bytes.Buffer{}
	io.Copy(buf, resp.Body)
	io.Copy(hash, bytes.NewReader(buf.Bytes()))
	h := fmt.Sprintf("%x", hash.Sum(nil))
	logrus.Debugf("MD5 for file %s is %s\n", request.File.Name, h)
	// Do the network commands in parallel
	c := make(chan int, 1)
	go func() {
		virus, err := w.clam.scan(request.File.Name, buf.Bytes())
		if (err == nil || err.Error() == "Virus(es) detected") && virus != "" {
			reply.File.Virus = virus
		} else if err != nil {
			reply.File.Error = err.Error()
		}
		c <- 1
	}()
	w.handleMD5(h, reply)
	<-c
	reply.File.Result = domain.ResultUnknown
	if reply.File.Virus != "" || len(reply.MD5.XFE.Malware.Family) > 0 || reply.MD5.VT.FileReport.Positives > numOfPositivesToConvict {
		// This is known bad scenario
		reply.File.Result = domain.ResultDirty
	} else if reply.File.Virus == "" || !reply.MD5.XFE.NotFound || reply.MD5.VT.FileReport.ResponseCode == 1 {
		// At least one of reputation services found this to be known good
		// Keep the default
		reply.File.Result = domain.ResultClean
	}
}
