package bot

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/slack"
)

const (
	poweredBy = "\t-\tPowered by <http://slack.demisto.com|Demisto>"
	botName   = "Alfred"
)

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

func (b *Bot) isThereInterestIn(original *slack.Message) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	data := original.Context.(*context)
	subs := b.subscriptions[data.Team]
	if subs == nil {
		return false
	}
	if sub := subs.FirstSubForChannel(original.Channel); sub != nil {
		return true
	}
	return false
}

func (b *Bot) alreadyHandled(original *slack.Message) bool {
	data := original.Context.(*context)
	handled := b.handledMessages[data.Team]
	if handled == nil {
		handled = make(map[string]*time.Time)
		b.handledMessages[data.Team] = handled
	}
	var field string
	// We care only about messages
	if original.Type == "message" {
		switch original.Subtype {
		case "file_shared":
			field = original.File.Name + "|" + original.User
		case "message_changed":
			field = original.Message.Text + "|" + original.User
		default:
			field = original.Text + "|" + original.User
		}
	}
	if field == "" {
		// Ignore the message
		return true
	}
	if handled[field] != nil {
		return true
	}
	now := time.Now()
	handled[field] = &now
	return false
}

// post uses the correct client to post to the channel
// TODO - terrible hack
func (b *Bot) post(message *slack.PostMessageRequest, original *slack.Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Get the correct team
	data := original.Context.(*context)
	subs := b.subscriptions[data.Team]
	if subs == nil {
		return fmt.Errorf("No team found for the Slack message - [%s, %s]", data.Team, data.User)
	}
	// First, let's see if the posting user is in our list
	if sub := subs.SubForUser(data.User); sub != nil {
		_, err := sub.s.PostMessage(message, false)
		return err
	}
	// If not, let's find the first user with interest in the channel
	if sub := subs.FirstSubForChannel(original.Channel); sub != nil {
		_, err := sub.s.PostMessage(message, false)
		return err
	}
	return fmt.Errorf("No interest in channel %s", message.Channel)
}

func (b *Bot) handleURL(message slack.Message) {
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
		urlResp, urlRespErr := b.xfe.URL(url)
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
		resolve, resolveErr := b.xfe.Resolve(url)
		if resolveErr == nil {
			for i := range resolve.MX {
				mx += fmt.Sprintf("%s (%d) ", resolve.MX[i].Exchange, resolve.MX[i].Priority)
			}
		}

		vtMessage := ""
		vtColor := "good"
		vtResp, err := b.vt.GetUrlReport(url)
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
		err = b.post(postMessage, &message)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v", err)
		}
	}
}

func (b *Bot) handleIP(message slack.Message, ip string) {
	xfeMessage := ""
	color := "good"
	ipResp, ipRespErr := b.xfe.IPR(ip)
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
	vtResp, err := b.vt.GetIpReport(ip)
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
	err = b.post(postMessage, &message)
	if err != nil {
		logrus.Errorf("Unable to send message to Slack - %v", err)
	}
}

func (b *Bot) handleMD5(message slack.Message, md5 string) {
	xfeMessage := ""
	color := "good"
	md5Resp, md5RespErr := b.xfe.MalwareDetails(md5)
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
	vtResp, err := b.vt.GetFileReport(md5)
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
	err = b.post(postMessage, &message)
	if err != nil {
		logrus.Errorf("Unable to send message to Slack - %v\n", err)
	}
}

func (b *Bot) handleFile(message slack.Message) {
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
	md5Resp, md5RespErr := b.xfe.MalwareDetails(h)
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
	vtResp, vtErr := b.vt.GetFileReport(h)
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
	virus, err := scan(message.File.Name, buf.Bytes())
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
	err = b.post(postMessage, &message)
	if err != nil {
		logrus.Errorf("Unable to send message to Slack - %v\n", err)
	}
}
