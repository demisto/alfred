package bot

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/goxforce"
	"github.com/demisto/slack"
	"github.com/slavikm/govt"
)

const (
	poweredBy = "\t-\tPowered by <http://slack.demisto.com|Demisto>"
	botName   = "Alfred"
)

// Worker reads messages from the queue and does the actual work
type Worker struct {
	q   queue.Queue
	c   chan *slack.Message
	r   repo.Repo
	xfe *goxforce.Client
	vt  *govt.Client
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
	return &Worker{
		r:   r,
		q:   q,
		c:   make(chan *slack.Message, runtime.NumCPU()),
		xfe: xfe,
		vt:  vt,
	}, nil
}

func (w *Worker) handle() {
	for msg := range w.c {
		if msg.Subtype == "file_share" {
			w.handleFile(msg)
			// If it's file share - don't bother with the rest
			continue
		}
		if strings.Contains(msg.Text, "<http") {
			w.handleURL(msg)
		}
		if ip := ipReg.FindString(msg.Text); ip != "" {
			w.handleIP(msg, ip)
		}
		if hash := md5Reg.FindString(msg.Text); hash != "" {
			w.handleMD5(msg, hash)
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
		if err != nil {
			logrus.Info("Stoping WorkManager process")
			close(w.c)
			return
		}
		w.c <- msg
	}
}

func joinMap(m map[string]bool) string {
	res := ""
	for k, v := range m {
		if v {
			res += k + ","
		}
	}
	if len(res) > 0 {
		return res[0 : len(res)-1]
	}
	return res
}

func joinMapInt(m map[string]int) string {
	res := ""
	for k, v := range m {
		res += fmt.Sprintf("%s (%d),", k, v)
	}
	if len(res) > 0 {
		return res[0 : len(res)-1]
	}
	return res
}

// post uses the correct client to post to the channel
// See if the original message poster is subscribed and if so use him.
// If not, use the first user we have that is subscribed to the channel.
func (w *Worker) post(message *slack.PostMessageRequest, original *slack.Message) error {
	u, err := w.r.UserByExternalID(original.User)
	if err != nil && err != repo.ErrNotFound {
		return err
	}
	var s *slack.Slack
	if err != nil {
		data := original.Context.(*Context)
		u, err = w.r.User(data.User)
		if err != nil {
			return err
		}
	}
	s, err = slack.New(slack.SetToken(u.Token))
	if err != nil {
		return err
	}
	_, err = s.PostMessage(message, false)
	return err
}

func (w *Worker) handleURL(message *slack.Message) {
	start := strings.Index(message.Text, "<http")
	end := strings.Index(message.Text[start:], ">")
	if end > 0 {
		end = end + start
		filter := strings.Index(message.Text[start:end], "|")
		if filter > 0 {
			end = start + filter
		}
		url := message.Text[start+1 : end]
		logrus.Debugf("URL found - %s\n", url)

		// Do the network commands in parallel
		c := make(chan int, 2)
		var urlResp *goxforce.URLResp
		var urlRespErr, resolveErr, err error
		var resolve *goxforce.ResolveResp
		var vtResp *govt.UrlReport
		go func() {
			urlResp, urlRespErr = w.xfe.URL(url)
			resolve, resolveErr = w.xfe.Resolve(url)
			c <- 1
		}()
		go func() {
			vtResp, err = w.vt.GetUrlReport(url)
			c <- 1
		}()
		for i := 0; i < 2; i++ {
			<-c
		}

		xfeMessage := ""
		color := "good"
		if urlRespErr != nil {
			// Small hack - see if the URL was not found
			if strings.Contains(urlRespErr.Error(), "404") {
				xfeMessage = "URL reputation not found"
			} else {
				xfeMessage = urlRespErr.Error()
			}
			color = "warning"
		} else {
			xfeMessage = fmt.Sprintf("Categories: %s. Score: %v", joinMap(urlResp.Result.Cats), urlResp.Result.Score)
			if urlResp.Result.Score >= 5 {
				color = "danger"
			} else if urlResp.Result.Score >= 1 {
				color = "warning"
			}
		}
		// If there is a problem, ignore it - the fields are going to be empty
		mx := ""
		if resolveErr == nil {
			for i := range resolve.MX {
				mx += fmt.Sprintf("%s (%d) ", resolve.MX[i].Exchange, resolve.MX[i].Priority)
			}
		}

		vtMessage := ""
		vtColor := "good"
		if err != nil {
			vtMessage = err.Error()
			vtColor = "warning"
		} else {
			if vtResp.ResponseCode != 1 {
				vtMessage = fmt.Sprintf("VT error %d (%s)", vtResp.ResponseCode, vtResp.VerboseMsg)
			} else {
				detected := 0
				for i := range vtResp.Scans {
					if vtResp.Scans[i].Detected {
						detected++
					}
				}
				if detected >= 5 {
					vtColor = "danger"
				} else if detected >= 1 {
					vtColor = "warning"
				}
				vtMessage = fmt.Sprintf("Scan Date: %s, Detected: %d, Total: %d", vtResp.ScanDate, detected, int(vtResp.Total))
			}
		}
		postMessage := &slack.PostMessageRequest{
			Channel:  message.Channel,
			Text:     "URL Reputation for " + url + poweredBy,
			Username: botName,
			Attachments: []slack.Attachment{
				{
					Fallback:   xfeMessage,
					AuthorName: "IBM X-Force Exchange",
					Color:      color,
				},
				{
					Fallback:   vtMessage,
					AuthorName: "VirusTotal",
					Text:       vtMessage,
					Color:      vtColor,
				},
			},
		}
		if resolveErr == nil {
			postMessage.Attachments[0].Fields = []slack.AttachmentField{
				{Title: "A", Value: strings.Join(resolve.A, ","), Short: true},
				{Title: "AAAA", Value: strings.Join(resolve.AAAA, ","), Short: true},
				{Title: "TXT", Value: strings.Join(resolve.TXT, ","), Short: true},
				{Title: "MX", Value: mx, Short: true},
			}
		}
		if urlRespErr == nil {
			postMessage.Attachments[0].Fields = append(postMessage.Attachments[0].Fields,
				slack.AttachmentField{Title: "Categories", Value: joinMap(urlResp.Result.Cats), Short: true},
				slack.AttachmentField{Title: "Score", Value: fmt.Sprintf("%v", urlResp.Result.Score), Short: true})
		} else {
			postMessage.Attachments[0].Text = xfeMessage
		}
		err = w.post(postMessage, message)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v", err)
		}
	}
}

func (w *Worker) handleIP(message *slack.Message, ip string) {
	xfeMessage := ""
	color := "good"
	// Do the network commands in parallel
	c := make(chan int, 2)
	var ipResp *goxforce.IPReputation
	var vtResp *govt.IpReport
	var ipRespErr, err error
	go func() {
		ipResp, ipRespErr = w.xfe.IPR(ip)
		c <- 1
	}()
	go func() {
		vtResp, err = w.vt.GetIpReport(ip)
		c <- 1
	}()
	for i := 0; i < 2; i++ {
		<-c
	}
	if ipRespErr != nil {
		// Small hack - see if the URL was not found
		if strings.Contains(ipRespErr.Error(), "404") {
			xfeMessage = "IP reputation not found"
		} else {
			xfeMessage = ipRespErr.Error()
		}
		color = "warning"
	} else {
		xfeMessage = fmt.Sprintf("Categories: %s. Country: %s. Score: %v", joinMapInt(ipResp.Cats), ipResp.Geo["country"].(string), ipResp.Score)
		if ipResp.Score >= 5 {
			color = "danger"
		} else if ipResp.Score >= 1 {
			color = "warning"
		}
	}
	vtMessage := ""
	vtColor := "good"
	if err != nil {
		vtMessage = err.Error()
		vtColor = "warning"
	} else {
		if vtResp.ResponseCode != 1 {
			vtMessage = fmt.Sprintf("VT error %d (%s)", vtResp.ResponseCode, vtResp.VerboseMsg)
			vtColor = "warning"
		} else {
			detected := 0
			vtMessage = "Detected URLs:\n"
			for i := range vtResp.DetectedUrls {
				vtMessage += fmt.Sprintf("URL: %s, Detected: %d, Total: %d, Scan Date: %s\n",
					vtResp.DetectedUrls[i].Url, int(vtResp.DetectedUrls[i].Positives), int(vtResp.DetectedUrls[i].Total), vtResp.DetectedUrls[i].ScanDate)
				detected += int(vtResp.DetectedUrls[i].Positives)
			}
			if detected >= 10 {
				vtColor = "danger"
			} else if detected >= 5 {
				vtColor = "warning"
			}
		}
	}
	postMessage := &slack.PostMessageRequest{
		Channel:  message.Channel,
		Text:     "IP Reputation for " + ip + poweredBy,
		Username: botName,
		Attachments: []slack.Attachment{
			{
				Fallback:   xfeMessage,
				AuthorName: "IBM X-Force Exchange",
				Color:      color,
			},
			{
				Fallback:   vtMessage,
				AuthorName: "VirusTotal",
				Text:       vtMessage,
				Color:      vtColor,
			},
		},
	}
	if ipRespErr == nil {
		postMessage.Attachments[0].Fields = []slack.AttachmentField{
			{Title: "Categories", Value: joinMapInt(ipResp.Cats), Short: true},
			{Title: "Country", Value: ipResp.Geo["country"].(string), Short: true},
			{Title: "Score", Value: fmt.Sprintf("%v", ipResp.Score), Short: true},
		}
	} else {
		postMessage.Attachments[0].Text = xfeMessage
	}
	err = w.post(postMessage, message)
	if err != nil {
		logrus.Errorf("Unable to send message to Slack - %v", err)
	}
}

func (w *Worker) handleMD5(message *slack.Message, md5 string) {
	xfeMessage := ""
	color := "good"
	// Do the network commands in parallel
	c := make(chan int, 2)
	var md5Resp *goxforce.MalwareResp
	var vtResp *govt.FileReport
	var md5RespErr, err error
	go func() {
		md5Resp, md5RespErr = w.xfe.MalwareDetails(md5)
		c <- 1
	}()
	go func() {
		vtResp, err = w.vt.GetFileReport(md5)
		c <- 1
	}()
	for i := 0; i < 2; i++ {
		<-c
	}
	if md5RespErr != nil {
		// Small hack - see if the file was not found
		if strings.Contains(md5RespErr.Error(), "404") {
			xfeMessage = "File reputation not found"
		} else {
			xfeMessage = md5RespErr.Error()
		}
		color = "warning"
	} else {
		xfeMessage = fmt.Sprintf("Type: %s, Created: %s, Family: %s, MIME: %s, External: %s (%d)",
			md5Resp.Malware.Type, md5Resp.Malware.Created.String(), strings.Join(md5Resp.Malware.Family, ","), md5Resp.Malware.MimeType,
			strings.Join(md5Resp.Malware.Origins.External.Family, ","), md5Resp.Malware.Origins.External.DetectionCoverage)
		if len(md5Resp.Malware.Family) > 0 || md5Resp.Malware.Origins.External.DetectionCoverage > 5 {
			color = "danger"
		}
	}

	vtMessage := ""
	vtColor := "good"
	if err != nil {
		vtMessage = err.Error()
		vtColor = "warning"
	} else {
		if vtResp.ResponseCode != 1 {
			vtMessage = fmt.Sprintf("VT error %d (%s)", vtResp.ResponseCode, vtResp.VerboseMsg)
			if vtResp.ResponseCode != 0 {
				vtColor = "warning"
			}
		} else {
			vtMessage = fmt.Sprintf("Scan Date %s, Positives: %d, Total: %d\n", vtResp.ScanDate, int(vtResp.Positives), int(vtResp.Total))
			if vtResp.Positives >= 5 {
				vtColor = "danger"
			} else if vtResp.Positives >= 1 {
				vtColor = "warning"
			}
		}
	}
	postMessage := &slack.PostMessageRequest{
		Channel:  message.Channel,
		Text:     "File Reputation for " + md5 + poweredBy,
		Username: botName,
		Attachments: []slack.Attachment{
			{
				Fallback:   xfeMessage,
				AuthorName: "IBM X-Force Exchange",
				Color:      color,
			},
			{
				Fallback:   vtMessage,
				AuthorName: "VirusTotal",
				Text:       vtMessage,
				Color:      vtColor,
			},
		},
	}
	if md5RespErr == nil {
		postMessage.Attachments[0].Fields = []slack.AttachmentField{
			{Title: "Type", Value: md5Resp.Malware.Type, Short: true},
			{Title: "Created", Value: md5Resp.Malware.Created.String(), Short: true},
			{Title: "Family", Value: strings.Join(md5Resp.Malware.Family, ","), Short: true},
			{Title: "MIME Type", Value: md5Resp.Malware.MimeType, Short: true},
			{Title: "External", Value: fmt.Sprintf("%s (%d)", strings.Join(md5Resp.Malware.Origins.External.Family, ","), md5Resp.Malware.Origins.External.DetectionCoverage), Short: true},
		}
	} else {
		postMessage.Attachments[0].Text = xfeMessage
	}
	err = w.post(postMessage, message)
	if err != nil {
		logrus.Errorf("Unable to send message to Slack - %v\n", err)
	}
}

func (w *Worker) handleFile(message *slack.Message) {
	hash := md5.New()
	resp, err := http.Get(message.File.URL)
	if err != nil {
		logrus.Errorf("Unable to download file - %v\n", err)
		return
	}
	defer resp.Body.Close()
	buf := &bytes.Buffer{}
	io.Copy(buf, resp.Body)
	io.Copy(hash, bytes.NewReader(buf.Bytes()))
	h := fmt.Sprintf("%x", hash.Sum(nil))
	logrus.Debugf("MD5 for file %s is %s\n", message.File.Name, h)
	xfeMessage := ""
	color := "good"
	// Do the network commands in parallel
	c := make(chan int, 3)
	var md5Resp *goxforce.MalwareResp
	var vtResp *govt.FileReport
	var virus string
	var md5RespErr, vtErr error
	go func() {
		md5Resp, md5RespErr = w.xfe.MalwareDetails(h)
		c <- 1
	}()
	go func() {
		vtResp, vtErr = w.vt.GetFileReport(h)
		c <- 1
	}()
	go func() {
		virus, err = scan(message.File.Name, buf.Bytes())
		c <- 1
	}()
	for i := 0; i < 3; i++ {
		<-c
	}
	if md5RespErr != nil {
		// Small hack - see if the URL was not found
		if strings.Contains(md5RespErr.Error(), "404") {
			xfeMessage = "File reputation not found"
		} else {
			xfeMessage = md5RespErr.Error()
		}
		color = "warning"
	} else {
		xfeMessage = fmt.Sprintf("Type: %s, Created: %s, Family: %s, MIME: %s, External: %s (%d)",
			md5Resp.Malware.Type, md5Resp.Malware.Created.String(), strings.Join(md5Resp.Malware.Family, ","), md5Resp.Malware.MimeType,
			strings.Join(md5Resp.Malware.Origins.External.Family, ","), md5Resp.Malware.Origins.External.DetectionCoverage)
		if len(md5Resp.Malware.Family) > 0 || md5Resp.Malware.Origins.External.DetectionCoverage > 5 {
			color = "danger"
		}
	}
	vtMessage := ""
	vtColor := "good"
	if vtErr != nil {
		vtMessage = vtErr.Error()
		vtColor = "warning"
	} else {
		if vtResp.ResponseCode != 1 {
			vtMessage = fmt.Sprintf("VT error %d (%s)", vtResp.ResponseCode, vtResp.VerboseMsg)
			if vtResp.ResponseCode != 0 {
				vtColor = "warning"
			}
		} else {
			vtMessage = fmt.Sprintf("Scan Date %s, Positives: %d, Total: %d\n", vtResp.ScanDate, int(vtResp.Positives), int(vtResp.Total))
			if vtResp.Positives >= 5 {
				vtColor = "danger"
			} else if vtResp.Positives >= 1 {
				vtColor = "warning"
			}
		}
	}
	postMessage := &slack.PostMessageRequest{
		Channel:  message.Channel,
		Text:     "File Reputation for " + message.File.Name + poweredBy,
		Username: botName,
		Attachments: []slack.Attachment{
			{
				Fallback:   xfeMessage,
				AuthorName: "IBM X-Force Exchange",
				Color:      color,
			},
			{
				Fallback:   vtMessage,
				AuthorName: "VirusTotal",
				Text:       vtMessage,
				Color:      vtColor,
			},
		},
	}
	if md5RespErr == nil {
		postMessage.Attachments[0].Fields = []slack.AttachmentField{
			{Title: "Type", Value: md5Resp.Malware.Type, Short: true},
			{Title: "Created", Value: md5Resp.Malware.Created.String(), Short: true},
			{Title: "Family", Value: strings.Join(md5Resp.Malware.Family, ","), Short: true},
			{Title: "MIME Type", Value: md5Resp.Malware.MimeType, Short: true},
			{Title: "External", Value: fmt.Sprintf("%s (%d)", strings.Join(md5Resp.Malware.Origins.External.Family, ","), md5Resp.Malware.Origins.External.DetectionCoverage), Short: true},
		}
	} else {
		postMessage.Attachments[0].Text = xfeMessage
	}
	// If both reputation services are in error or not familiar with the file
	// if md5RespErr != nil && (vtErr != nil || vtResp.Status.ResponseCode != 1) {
	if (err == nil || err.Error() == "Virus(es) detected") && virus != "" {
		clamMessage := fmt.Sprintf("Virus [%s] found", virus)
		postMessage.Attachments = append(postMessage.Attachments,
			slack.Attachment{
				Fallback:   clamMessage,
				AuthorName: "ClamAV",
				Text:       clamMessage,
				Color:      "danger",
			})
	}
	// }
	err = w.post(postMessage, message)
	if err != nil {
		logrus.Errorf("Unable to send message to Slack - %v\n", err)
	}
}
