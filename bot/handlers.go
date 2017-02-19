package bot

import (
	"bytes"
	"crypto/md5"
	"debug/pe"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/goxforce"
	"github.com/demisto/infinigo"
	stackerr "github.com/go-errors/errors"
	"github.com/slavikm/govt"
)

const (
	numOfPositivesToConvict         = 7
	numOfPositivesToConvictForFiles = 3
	xfeScoreToConvict               = 7
	cyScoreToConvict                = -0.5
)

// Worker reads messages from the queue and does the actual work
type Worker struct {
	q    queue.Queue
	c    chan *domain.WorkRequest
	r    repo.Repo
	xfe  *goxforce.Client
	vt   *govt.Client
	cy   *infinigo.Client
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
	cy, err := infinigo.New(
		infinigo.SetKey(conf.Options.Cy),
		infinigo.SetErrorLog(log.New(conf.LogWriter, "VT:", log.Lshortfile)))
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
		cy:   cy,
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
				w.handleURL(msg, reply)
			}
			if ipReg.MatchString(msg.Text) {
				w.handleIP(msg, reply)
			}
			if md5Reg.MatchString(msg.Text) || sha1Reg.MatchString(msg.Text) || sha256Reg.MatchString(msg.Text) {
				w.handleHashes(msg, reply)
			}
		case "file":
			w.handleFile(msg, reply)
		}
		if err := w.q.PushWorkReply(msg.ReplyQueue, reply); err != nil {
			logrus.Warnf("Error pushing message to reply queue %+v - %v\n", msg, err)
		}
	}
}

// Start the worker process. To stop, just close the queue.
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

func (w *Worker) localVTXfe(request *domain.WorkRequest) (*goxforce.Client, *govt.Client) {
	vt := w.vt
	if request.VTKey != "" {
		vtTmp, err := govt.New(
			govt.SetApikey(request.VTKey),
			govt.SetErrorLog(log.New(conf.LogWriter, "VT:", log.Lshortfile)))
		if err == nil {
			vt = vtTmp
		}
	}
	xfe := w.xfe
	if request.XFEKey != "" && request.XFEPass != "" {
		xfeTmp, err := goxforce.New(
			goxforce.SetCredentials(request.XFEKey, request.XFEPass),
			goxforce.SetErrorLog(log.New(conf.LogWriter, "XFE:", log.Lshortfile)))
		if err == nil {
			xfe = xfeTmp
		}
	}
	return xfe, vt
}

func (w *Worker) handleURL(request *domain.WorkRequest, reply *domain.WorkReply) {
	text := request.Text
	online := request.Online
	xfe, vt := w.localVTXfe(request)
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
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			urlResp, err := xfe.URL(url)
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
			resolve, err := xfe.Resolve(url)
			if err == nil {
				reply.URLs[counter].XFE.Resolve = *resolve
			}
			if online {
				malware, err := xfe.URLMalware(url)
				if err == nil {
					reply.URLs[counter].XFE.URLMalware = *malware
				}
			}
		}()
		go func() {
			defer wg.Done()
			vtResp, err := vt.GetUrlReport(url)
			if err != nil {
				reply.URLs[counter].VT.Error = err.Error()
			} else {
				reply.URLs[counter].VT.URLReport = *vtResp
			}
		}()
		wg.Wait()
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

func (w *Worker) handleIP(request *domain.WorkRequest, reply *domain.WorkReply) {
	text := request.Text
	online := request.Online
	xfe, vt := w.localVTXfe(request)
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
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			ipResp, err := xfe.IPR(ip)
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
					hist, err := xfe.IPRHistory(ip)
					if err == nil {
						reply.IPs[counter].XFE.IPHistory = *hist
					}
				}
			}
		}()
		go func() {
			defer wg.Done()
			vtResp, err := vt.GetIpReport(ip)
			if err != nil {
				reply.IPs[counter].VT.Error = err.Error()
			} else {
				reply.IPs[counter].VT.IPReport = *vtResp
			}
		}()
		wg.Wait()
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

func (w *Worker) handleHashes(request *domain.WorkRequest, reply *domain.WorkReply) {
	text := request.Text
	xfe, vt := w.localVTXfe(request)
	hashes := md5Reg.FindAllString(text, -1)
	hashes = append(hashes, sha1Reg.FindAllString(text, -1)...)
	hashes = append(hashes, sha256Reg.FindAllString(text, -1)...)
	for _, hash := range hashes {
		var res domain.HashReply
		reply.Type |= domain.ReplyTypeHash
		res.Details = hash
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			xfeResp, err := xfe.MalwareDetails(hash)
			if err != nil {
				// Small hack - see if the file was not found
				if strings.Contains(err.Error(), "404") {
					res.XFE.NotFound = true
				} else {
					res.XFE.Error = err.Error()
				}
			} else {
				res.XFE.Malware = xfeResp.Malware
			}
		}()
		go func() {
			defer wg.Done()
			vtResp, err := vt.GetFileReport(hash)
			if err != nil {
				res.VT.Error = err.Error()
			} else {
				res.VT.FileReport = *vtResp
			}
		}()
		go func() {
			defer wg.Done()
			cyResp, err := w.cy.Query("", hash)
			if err != nil {
				res.Cy.Error = err.Error()
			} else {
				// Should be only one
				for k := range cyResp {
					res.Cy.Result = cyResp[k]
				}
			}
		}()
		wg.Wait()
		res.Result = domain.ResultUnknown
		if len(res.XFE.Malware.Family) > 0 || len(res.XFE.Malware.Origins.External.Family) > 0 ||
			res.VT.FileReport.Positives >= numOfPositivesToConvictForFiles ||
			res.Cy.Result.GeneralScore < cyScoreToConvict {
			// This is known bad scenario
			res.Result = domain.ResultDirty
		} else if !res.XFE.NotFound || res.VT.FileReport.ResponseCode == 1 || res.Cy.Result.StatusCode == 1 {
			// At least one of reputation services found this to be known good
			// Keep the default
			res.Result = domain.ResultClean
		}
		reply.Hashes = append(reply.Hashes, res)
	}
}

func (w *Worker) uploadToCylance(reply *domain.WorkReply, buf *bytes.Buffer) {
	// For now, just check Windows executables
	_, err := pe.NewFile(bytes.NewReader(buf.Bytes()))
	if err != nil {
		logrus.WithError(err).Infof("Error reading the file as PE file - %s", reply.File.Details.Name)
		return
	}
	logrus.Debugf("Sending file %s to Cylance", reply.File.Details.Name)
	resp, err := w.cy.Upload(reply.Hashes[0].Cy.Result.ConfirmCode, bytes.NewReader(buf.Bytes()))
	if err != nil {
		logrus.WithError(err).Infof("Error uploading the file - configuration code was %s", reply.Hashes[0].Cy.Result.ConfirmCode)
		return
	}
	for k := range resp {
		if resp[k].StatusCode == 1 {
			// Wait for 10 seconds and try getting reply again
			tries := 3
			for i := 0; i < tries; i++ {
				time.Sleep(10 * time.Second)
				cyResp, err := w.cy.Query("", reply.Hashes[0].Details)
				if err != nil {
					return
				} else {
					// Should be only one
					for k := range cyResp {
						if cyResp[k].StatusCode == 1 {
							reply.Hashes[0].Cy.Result = cyResp[k]
							return
						} else if cyResp[k].StatusCode != 2 {
							// If there is an error it means Cylance does not handle the file so no point in waiting
							return
						}
					}
				}
			}
		} else {
			logrus.Debugf("File was not accepted - %v [%s]", resp[k].StatusCode, resp[k].Error)
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
	req, err := http.NewRequest("GET", request.File.URL, nil)
	if err != nil {
		logrus.Errorf("Unable to create request for download file - %v\n", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+request.File.Token)
	resp, err := http.DefaultClient.Do(req)
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
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.Error(err)
				logrus.Error(stackerr.Wrap(err, 2).ErrorStack())
			}
			wg.Done()
		}()
		virus, err := w.clam.scan(request.File.Name, buf.Bytes())
		if (err == nil || err.Error() == "Virus(es) detected") && virus != "" {
			reply.File.Virus = virus
		} else if err != nil {
			reply.File.Error = err.Error()
		}
	}()
	request.Text = h
	w.handleHashes(request, reply)
	wg.Wait()
	reply.File.Result = domain.ResultUnknown
	if len(reply.Hashes) != 1 {
		logrus.Warnf("Handling file but did not get an MD5 reply - %+v", reply)
		return
	}
	// If Cylance does not know about the file but can handle it then handle it...
	if reply.Hashes[0].Cy.Result.StatusCode == 3 {
		w.uploadToCylance(reply, buf)
	}
	if reply.File.Virus != "" || reply.Hashes[0].Result == domain.ResultDirty {
		// This is known bad scenario
		reply.File.Result = domain.ResultDirty
	} else if reply.File.Virus == "" || reply.Hashes[0].Result == domain.ResultClean {
		// At least one of reputation services found this to be known good
		// Keep the default
		reply.File.Result = domain.ResultClean
	}
}
