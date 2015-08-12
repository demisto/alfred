package bot

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/slack"
)

// Take care not to change the file comments because this is how we detect if we already commented on the file
const (
	poweredBy               = "\t-\tPowered by <http://slack.demisto.com|Demisto>"
	botName                 = "Alfred"
	reactionTooBig          = "warning"
	reactionGood            = "+1"
	reactionBad             = "imp"
	fileCommentGood         = "Alfred says this file is clean. Click %s for more details."
	fileCommentBig          = "Alfred says this file is too large to scan. Click %s for more details."
	fileCommentBad          = "Alfred says this file is malicious. Click %s for more details."
	fileCommentWarning      = "Alfred does not have details regarding this file. Click %s for more details."
	urlCommentGood          = "Alfred says this URL is clean. Click %s for more details."
	urlCommentBad           = "Alfred says this URL is malicious. Click %s for more details."
	urlCommentWarning       = "Alfred does not have details regarding this URL. Click %s for more details."
	ipCommentGood           = "Alfred says this IP is clean. Click %s for more details."
	ipCommentBad            = "Alfred says this IP is malicious. Click %s for more details."
	ipCommentWarning        = "Alfred does not have details regarding this IP. Click %s for more details."
	md5CommentGood          = "Alfred says this MD5 hash is clean. Click %s for more details."
	md5CommentBad           = "Alfred says this MD5 hash is malicious. Click %s for more details."
	md5CommentWarning       = "Alfred does not have details regarding this MD5 hash. Click %s for more details."
	mainMessage             = "Security check by Alfred - your Demisto butler. Click <%s|here> for configuration and details."
	numOfPositivesToConvict = 3
	xfeScoreToConvict       = 3
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

func mainMessageFormatted() string {
	return fmt.Sprintf(mainMessage, conf.Options.ExternalAddress)
}

func (b *Bot) handleFileReply(reply *domain.WorkReply, data *domain.Context) {
	link := fmt.Sprintf("%s/details?f=%s&u=%s", conf.Options.ExternalAddress, reply.File.Details.ID, data.User)
	if reply.File.FileTooLarge {
		err := b.fileComment(fmt.Sprintf(fileCommentBig, link), reactionTooBig, reply)
		if err != nil {
			logrus.Warnf("Error commenting on file - %v\n", err)
		}
		return
	}
	color := "warning"
	comment := fileCommentWarning
	reaction := reactionTooBig
	if reply.File.Virus != "" || len(reply.MD5.XFE.Malware.Family) > 0 || reply.MD5.VT.FileReport.Positives > numOfPositivesToConvict {
		// This is known bad scenario
		color = "danger"
		comment = fileCommentBad
		reaction = reactionBad
	} else if reply.File.Virus == "" && (!reply.MD5.XFE.NotFound || reply.MD5.VT.FileReport.ResponseCode == 1) {
		// At least one of reputation services found this to be known good
		// Keep the default
		color = "good"
		comment = fileCommentGood
		reaction = reactionGood
	}
	err := b.fileComment(fmt.Sprintf(comment, link), reaction, reply)
	if err != nil {
		logrus.Errorf("Unable to send comment to Slack - %v\n", err)
	}
	if data.Channel != "" {
		fileMessage := fmt.Sprintf(comment, fmt.Sprintf("<%s|here>", link))
		postMessage := &slack.PostMessageRequest{
			Channel:  data.Channel,
			Text:     mainMessageFormatted(),
			Username: botName,
			Attachments: []slack.Attachment{
				{
					Fallback:   fileMessage,
					Text:       fileMessage,
					AuthorName: botName,
					Color:      color,
				},
			},
		}
		err = b.post(postMessage, reply, data)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v\n", err)
			return
		}
	}
}

func (b *Bot) handleReply(reply *domain.WorkReply) {
	data, err := GetContext(reply.Context)
	if err != nil {
		logrus.Warnf("Error getting context from reply - %+v\n", reply)
		return
	}
	if reply.Type&domain.ReplyTypeFile > 0 {
		b.handleFileReply(reply, data)
	} else {
		link := fmt.Sprintf("%s/details?c=%s&m=%s&u=%s", conf.Options.ExternalAddress, data.Channel, reply.MessageID, data.User)
		postMessage := &slack.PostMessageRequest{
			Channel:  data.Channel,
			Text:     mainMessageFormatted(),
			Username: botName,
		}
		if reply.Type&domain.ReplyTypeURL > 0 {
			color := "warning"
			comment := urlCommentWarning
			if reply.URL.XFE.URLDetails.Score > xfeScoreToConvict || reply.URL.VT.URLReport.Positives > numOfPositivesToConvict {
				color = "danger"
				comment = urlCommentBad
			} else if !reply.URL.XFE.NotFound || reply.URL.VT.URLReport.ResponseCode == 1 {
				color = "good"
				comment = urlCommentGood
			}
			urlMessage := fmt.Sprintf(comment, fmt.Sprintf("<%s|here>", link))
			postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
				Fallback:   urlMessage,
				Text:       urlMessage,
				AuthorName: botName,
				Color:      color,
			})
		}
		if reply.Type&domain.ReplyTypeIP > 0 {
			color := "warning"
			comment := ipCommentWarning
			var vtPositives uint16
			for i := range reply.IP.VT.IPReport.DetectedUrls {
				if reply.IP.VT.IPReport.DetectedUrls[i].Positives > vtPositives {
					vtPositives = reply.IP.VT.IPReport.DetectedUrls[i].Positives
				}
			}
			if reply.IP.XFE.IPReputation.Score > xfeScoreToConvict || vtPositives > numOfPositivesToConvict {
				color = "danger"
				comment = urlCommentBad
			} else if !reply.IP.XFE.NotFound || reply.IP.VT.IPReport.ResponseCode == 1 {
				color = "good"
				comment = urlCommentGood
			}
			ipMessage := fmt.Sprintf(comment, fmt.Sprintf("<%s|here>", link))
			postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
				Fallback:   ipMessage,
				Text:       ipMessage,
				AuthorName: botName,
				Color:      color,
			})
		}
		if reply.Type&domain.ReplyTypeMD5 > 0 {
			color := "warning"
			comment := md5CommentWarning
			if len(reply.MD5.XFE.Malware.Family) > 0 || reply.MD5.VT.FileReport.Positives > numOfPositivesToConvict {
				color = "danger"
				comment = md5CommentBad
			} else if !reply.MD5.XFE.NotFound || reply.MD5.VT.FileReport.ResponseCode == 1 {
				color = "good"
				comment = md5CommentGood
			}
			md5Message := fmt.Sprintf(comment, fmt.Sprintf("<%s|here>", link))
			postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
				Fallback:   md5Message,
				Text:       md5Message,
				AuthorName: botName,
				Color:      color,
			})
		}
		err = b.post(postMessage, reply, data)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v\n", err)
			return
		}
	}
}

// post uses the correct client to post to the channel
// See if the original message poster is subscribed and if so use him.
// If not, use the first user we have that is subscribed to the channel.
func (b *Bot) post(message *slack.PostMessageRequest, reply *domain.WorkReply, data *domain.Context) error {
	u, err := b.r.UserByExternalID(data.OriginalUser)
	if err != nil && err != repo.ErrNotFound {
		return err
	}
	if err != nil {
		u, err = b.r.User(data.User)
		if err != nil {
			return err
		}
	}
	s, err := slack.New(slack.SetToken(u.Token))
	if err != nil {
		return err
	}
	_, err = s.PostMessage(message, false)
	return err
}

func (b *Bot) fileComment(comment, reaction string, reply *domain.WorkReply) error {
	var u *domain.User
	data, err := GetContext(reply.Context)
	if err != nil {
		return err
	}
	// If the context did not have any channel, it is a file_created event
	if data.Channel == "" {
		u, err = b.r.User(data.User)
		if err != nil {
			return err
		}
	} else {
		var err error
		u, err = b.r.UserByExternalID(data.OriginalUser)
		if err != nil && err != repo.ErrNotFound {
			return err
		}
		// If the user creating the file is our user than don't comment because we already did the comment on file create event
		if err == nil {
			return nil
		}
		u, err = b.r.User(data.User)
		if err != nil {
			return err
		}
	}
	s, err := slack.New(slack.SetToken(u.Token))
	if err != nil {
		return err
	}
	info, err := s.FileInfo(reply.File.Details.ID, 0, 0)
	if err != nil {
		return err
	}
	for i := range info.Comments {
		if strings.HasPrefix(info.Comments[i].Comment, botName) {
			return nil
		}
	}
	// We got here - means we do not have comment
	_, err = s.FileAddComment(reply.File.Details.ID, comment, false)
	if err != nil {
		return err
	}
	_, err = s.ReactionsAdd(reaction, reply.File.Details.ID, "", "", "")
	return err
}
