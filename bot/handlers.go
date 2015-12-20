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
	"runtime"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/goxforce"
	"github.com/slavikm/govt"
)

const (
	numOfPositivesToConvict         = 7
	numOfPositivesToConvictForFiles = 3
	xfeScoreToConvict               = 7
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
	xfe, err := goxforce.New(
		goxforce.SetCredentials(conf.Options.XFE.Key, conf.Options.XFE.Password),
		goxforce.SetErrorLog(log.New(conf.LogWriter, "XFE:", log.Lshortfile)))
	if err != nil {
		return nil, err
	}
	vt, err := govt.New(
		govt.SetApikey(conf.Options.VT),
		govt.SetErrorLog(log.New(conf.LogWriter, "VT:", log.Lshortfile)))
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
				w.handleURL(msg.Text, msg.Online, reply)
			}
			if ipReg.MatchString(msg.Text) {
				w.handleIP(msg.Text, msg.Online, reply)
			}
			if md5Reg.MatchString(msg.Text) {
				w.handleMD5(msg.Text, reply)
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
			logrus.Infof("Stoping WorkManager process - %v, %v", err, msg)
			close(w.c)
			return
		}
		logrus.Debugf("Working on message - %+v", msg)
		w.c <- msg
	}
}

func contextFromMap(c map[string]interface{}) *domain.Context {
	return &domain.Context{
		Team:         c["team"].(string),
		User:         c["user"].(string),
		OriginalUser: c["original_user"].(string),
		Channel:      c["channel"].(string),
		Type:         c["type"].(string),
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

func (w *Worker) handleURL(text string, online bool, reply *domain.WorkReply) {
	for {
		start := strings.Index(text, "<http")
		if start < 0 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end < start {
			break
		}
		end = end + start
		realEnd := end
		filter := strings.Index(text[start:end], "|")
		if filter > 0 {
			end = start + filter
		}
		url := text[start+1 : end]
		logrus.Debugf("URL found - %s\n", url)
		text = text[realEnd:]
		reply.URLs = append(reply.URLs, domain.URLReply{})
		counter := len(reply.URLs) - 1
		reply.URLs[counter].Details = url
		reply.Type |= domain.ReplyTypeURL
		// Do the network commands in parallel
		c := make(chan int, 2)
		go func() {
			urlResp, err := w.xfe.URL(url)
			if err != nil {
				// Small hack - see if the URL was not found
				if strings.Contains(err.Error(), "404") {
					reply.URLs[counter].XFE.NotFound = true
				} else {
					reply.URLs[counter].XFE.Error = err.Error()
				}
			} else {
				reply.URLs[counter].XFE.URLDetails = urlResp.Result
			}
			resolve, err := w.xfe.Resolve(url)
			if err == nil {
				reply.URLs[counter].XFE.Resolve = *resolve
			}
			if online {
				malware, err := w.xfe.URLMalware(url)
				if err == nil {
					reply.URLs[counter].XFE.URLMalware = *malware
				}
			}
			c <- 1
		}()
		go func() {
			vtResp, err := w.vt.GetUrlReport(url)
			if err != nil {
				reply.URLs[counter].VT.Error = err.Error()
			} else {
				reply.URLs[counter].VT.URLReport = *vtResp
			}
			c <- 1
		}()
		for i := 0; i < 2; i++ {
			<-c
		}
		if reply.URLs[counter].XFE.URLDetails.Score >= xfeScoreToConvict || reply.URLs[counter].VT.URLReport.Positives >= numOfPositivesToConvict {
			// This is known bad scenario
			reply.URLs[counter].Result = domain.ResultDirty
		} else if !reply.URLs[counter].XFE.NotFound || reply.URLs[counter].VT.URLReport.ResponseCode == 1 {
			// At least one of reputation services found this to be known good
			// Keep the default
			reply.URLs[counter].Result = domain.ResultClean
		}
	}
}

func (w *Worker) handleIP(text string, online bool, reply *domain.WorkReply) {
	ips := ipReg.FindAllString(text, -1)
	for _, ip := range ips {
		reply.IPs = append(reply.IPs, domain.IPReply{})
		counter := len(reply.IPs) - 1
		reply.IPs[counter].Details = ip
		reply.Type |= domain.ReplyTypeIP
		// First, let's check if IP is globally unicast addressable and is public
		ipData := net.ParseIP(ip)
		ipv4 := ipData.To4()
		if ipv4 == nil {
			// If not IPv4 then return - by default it will be marked clean
			reply.IPs[counter].XFE.NotFound = true
			return
		}
		if !ipv4.IsGlobalUnicast() {
			// If not global unicast ignore
			reply.IPs[counter].XFE.NotFound = true
			return
		}
		// Private networks
		if ipv4[0] == 10 || ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 || ipv4[0] == 192 && ipv4[1] == 168 {
			reply.IPs[counter].XFE.NotFound = true
			reply.IPs[counter].Private = true
			return
		}
		c := make(chan int, 2)
		go func() {
			ipResp, err := w.xfe.IPR(ip)
			if err != nil {
				// Small hack - see if the URL was not found
				if strings.Contains(err.Error(), "404") {
					reply.IPs[counter].XFE.NotFound = true
				} else {
					reply.IPs[counter].XFE.Error = err.Error()
				}
			} else {
				reply.IPs[counter].XFE.IPReputation = *ipResp
				if online {
					hist, err := w.xfe.IPRHistory(ip)
					if err == nil {
						reply.IPs[counter].XFE.IPHistory = *hist
					}
				}
			}
			c <- 1
		}()
		go func() {
			vtResp, err := w.vt.GetIpReport(ip)
			if err != nil {
				reply.IPs[counter].VT.Error = err.Error()
			} else {
				reply.IPs[counter].VT.IPReport = *vtResp
			}
			c <- 1
		}()
		for i := 0; i < 2; i++ {
			<-c
		}
		var vtPositives uint16
		now := time.Now()
		for i := range reply.IPs[counter].VT.IPReport.DetectedUrls {
			t, err := time.Parse("2006-01-02 15:04:05", reply.IPs[counter].VT.IPReport.DetectedUrls[i].ScanDate)
			if err != nil {
				logrus.Debugf("Error parsing scan date - %v", err)
				continue
			}
			if reply.IPs[counter].VT.IPReport.DetectedUrls[i].Positives > vtPositives && t.Add(365*24*time.Hour).After(now) {
				vtPositives = reply.IPs[counter].VT.IPReport.DetectedUrls[i].Positives
			}
		}
		reply.IPs[counter].Result = domain.ResultUnknown
		if reply.IPs[counter].XFE.IPReputation.Score >= xfeScoreToConvict || vtPositives >= numOfPositivesToConvict && reply.IPs[counter].XFE.NotFound {
			// This is known bad scenario
			reply.IPs[counter].Result = domain.ResultDirty
		} else if !reply.IPs[counter].XFE.NotFound || reply.IPs[counter].VT.IPReport.ResponseCode == 1 {
			// At least one of reputation services found this to be known good
			// Keep the default
			reply.IPs[counter].Result = domain.ResultClean
		}
	}
}

func (w *Worker) handleMD5(text string, reply *domain.WorkReply) {
	md5s := md5Reg.FindAllString(text, -1)
	for _, md5 := range md5s {
		reply.MD5s = append(reply.MD5s, domain.MD5Reply{})
		counter := len(reply.MD5s) - 1
		reply.Type |= domain.ReplyTypeMD5
		reply.MD5s[counter].Details = md5
		c := make(chan int, 2)
		go func() {
			md5Resp, err := w.xfe.MalwareDetails(md5)
			if err != nil {
				// Small hack - see if the file was not found
				if strings.Contains(err.Error(), "404") {
					reply.MD5s[counter].XFE.NotFound = true
				} else {
					reply.MD5s[counter].XFE.Error = err.Error()
				}
			} else {
				reply.MD5s[counter].XFE.Malware = md5Resp.Malware
			}
			c <- 1
		}()
		go func() {
			vtResp, err := w.vt.GetFileReport(md5)
			if err != nil {
				reply.MD5s[counter].VT.Error = err.Error()
			} else {
				reply.MD5s[counter].VT.FileReport = *vtResp
			}
			c <- 1
		}()
		for i := 0; i < 2; i++ {
			<-c
		}
		reply.MD5s[counter].Result = domain.ResultUnknown
		if len(reply.MD5s[counter].XFE.Malware.Family) > 0 || reply.MD5s[counter].VT.FileReport.Positives >= numOfPositivesToConvictForFiles {
			// This is known bad scenario
			reply.MD5s[counter].Result = domain.ResultDirty
		} else if !reply.MD5s[counter].XFE.NotFound || reply.MD5s[counter].VT.FileReport.ResponseCode == 1 {
			// At least one of reputation services found this to be known good
			// Keep the default
			reply.MD5s[counter].Result = domain.ResultClean
		}
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
	if len(reply.MD5s) != 1 {
		logrus.Warnf("Handling file but did not get an MD5 reply - %+v", reply)
		return
	}
	if reply.File.Virus != "" || len(reply.MD5s[0].XFE.Malware.Family) > 0 || reply.MD5s[0].VT.FileReport.Positives > numOfPositivesToConvict {
		// This is known bad scenario
		reply.File.Result = domain.ResultDirty
	} else if reply.File.Virus == "" || !reply.MD5s[0].XFE.NotFound || reply.MD5s[0].VT.FileReport.ResponseCode == 1 {
		// At least one of reputation services found this to be known good
		// Keep the default
		reply.File.Result = domain.ResultClean
	}
}
