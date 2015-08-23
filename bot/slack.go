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

const (
	poweredBy          = "\t-\tPowered by <http://slack.demisto.com|Demisto>"
	botName            = "DBot"
	reactionTooBig     = "warning"
	reactionGood       = "+1"
	reactionBad        = "imp"
	fileCommentGood    = "File (%s) is clean. Click %s for more details."
	fileCommentBig     = "File (%s) is too large to scan. Click %s for more details."
	fileCommentBad     = "Warning: File (%s) is malicious. Click %s for more details."
	fileCommentWarning = "Unable to find details regarding this file (%s). Click %s for more details."
	urlCommentGood     = "URL (%s) is clean: %s."
	urlCommentBad      = "Warning: URL (%s) is malicious: %s."
	urlCommentWarning  = "Unable to find details regarding this URL (%s): %s."
	ipCommentGood      = "IP (%s) is clean: %s."
	ipCommentBad       = "Warning: IP (%s) is malicious: %s."
	ipCommentWarning   = "Unable to find details regarding this IP (%s): %s."
	md5CommentGood     = "MD5 hash (%s) is clean: %s."
	md5CommentBad      = "Warning: MD5 hash (%s) is malicious: %s."
	md5CommentWarning  = "Unable to find details regarding this MD5 hash (%s): %s."
	mainMessage        = "Security check by DBot - Demisto Bot. Click <%s|here> for configuration and details."
	firstMessage       = "<@%s|%s> has added <%s|DBot> by Demisto to monitor this channel."
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
		err := b.fileComment(fmt.Sprintf(fileCommentBig, reply.File.Details.Name, link), reactionTooBig, reply)
		if err != nil {
			logrus.Warnf("Error commenting on file - %v\n", err)
		}
		return
	}
	color := "warning"
	comment := fileCommentWarning
	if reply.File.Result == domain.ResultDirty {
		color = "danger"
		comment = fileCommentBad
	} else if reply.File.Result == domain.ResultClean {
		// At least one of reputation services found this to be known good
		// Keep the default
		color = "good"
		comment = fileCommentGood
	}
	if data.Channel != "" {
		fileMessage := fmt.Sprintf(comment, reply.File.Details.Name, fmt.Sprintf("<%s|Details>", link))
		postMessage := &slack.PostMessageRequest{
			Channel:  data.Channel,
			Text:     mainMessageFormatted(),
			Username: botName,
			Attachments: []slack.Attachment{
				{
					Fallback: fileMessage,
					Text:     fileMessage,
					Color:    color,
				},
			},
		}
		err := b.post(postMessage, reply, data)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v\n", err)
			return
		}
	}
}

func (b *Bot) handleReplyStats(reply *domain.WorkReply, ctx *domain.Context) {
	b.smu.Lock()
	defer b.smu.Unlock()
	stats := b.stats[ctx.Team]
	if stats == nil {
		stats = &domain.Statistics{Team: ctx.Team}
		b.stats[ctx.Team] = stats
	}
	stats.Messages++
	if reply.Type&domain.ReplyTypeFile > 0 {
		if reply.File.Result == domain.ResultClean {
			stats.FilesClean++
		} else if reply.File.Result == domain.ResultDirty {
			stats.FilesDirty++
		} else {
			stats.FilesUnknown++
		}
	} else {
		if reply.Type&domain.ReplyTypeMD5 > 0 {
			if reply.MD5.Result == domain.ResultClean {
				stats.HashesClean++
			} else if reply.MD5.Result == domain.ResultDirty {
				stats.HashesDirty++
			} else {
				stats.HashesUnknown++
			}
		}
		if reply.Type&domain.ReplyTypeURL > 0 {
			if reply.URL.Result == domain.ResultClean {
				stats.URLsClean++
			} else if reply.URL.Result == domain.ResultDirty {
				stats.URLsDirty++
			} else {
				stats.URLsUnknown++
			}
		}
		if reply.Type&domain.ReplyTypeIP > 0 {
			if reply.IP.Result == domain.ResultClean {
				stats.IPsClean++
			} else if reply.IP.Result == domain.ResultDirty {
				stats.IPsDirty++
			} else {
				stats.IPsUnknown++
			}
		}
	}
}

func (b *Bot) handleReply(reply *domain.WorkReply) {
	logrus.Debugf("Handling reply - %+v\n", reply)
	logrus.Debugf("Context- %v\n", reply.Context)
	data, err := GetContext(reply.Context)
	if err != nil {
		logrus.Warnf("Error getting context from reply - %+v\n", reply)
		return
	}
	b.handleReplyStats(reply, data)
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
			if reply.URL.Result == domain.ResultDirty {
				color = "danger"
				comment = urlCommentBad
			} else if reply.URL.Result == domain.ResultClean {
				color = "good"
				comment = urlCommentGood
			}
			urlMessage := fmt.Sprintf(comment, reply.URL.Details, fmt.Sprintf("<%s|Details>", link))
			postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
				Fallback: urlMessage,
				Text:     urlMessage,
				Color:    color,
			})
		}
		if reply.Type&domain.ReplyTypeIP > 0 {
			color := "warning"
			comment := ipCommentWarning
			if reply.IP.Result == domain.ResultDirty {
				color = "danger"
				comment = ipCommentBad
			} else if reply.IP.Result == domain.ResultClean {
				color = "good"
				comment = ipCommentGood
			}
			ipMessage := fmt.Sprintf(comment, reply.IP.Details, fmt.Sprintf("<%s|Details>", link))
			postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
				Fallback: ipMessage,
				Text:     ipMessage,
				Color:    color,
			})
		}
		if reply.Type&domain.ReplyTypeMD5 > 0 {
			color := "warning"
			comment := md5CommentWarning
			if reply.MD5.Result == domain.ResultDirty {
				color = "danger"
				comment = md5CommentBad
			} else if reply.MD5.Result == domain.ResultClean {
				color = "good"
				comment = md5CommentGood
			}
			md5Message := fmt.Sprintf(comment, reply.MD5.Details, fmt.Sprintf("<%s|Details>", link))
			postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
				Fallback: md5Message,
				Text:     md5Message,
				Color:    color,
			})
		}
		err = b.post(postMessage, reply, data)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v\n", err)
			return
		}
	}
}

func (b *Bot) maybeSendFirstMessage(u *domain.User, s *slack.Slack, data *domain.Context) error {
	if data.Channel != "" && !strings.HasPrefix(data.Channel, "D") {
		if b.firstMessages[data.Channel+"@"+data.Team] {
			return nil
		}
		sent, err := b.r.WasMessageSentOnChannel(data.Team, data.Channel)
		if err != nil {
			logrus.Infof("Error reading first message info - %v", err)
			return err
		}
		if !sent {
			// If there is an error here, it might happen because someone did this in parallel
			err = b.r.MessageSentOnChannel(data.Team, data.Channel)
			if err != nil {
				return nil
			}
			b.firstMessages[data.Channel+"@"+data.Team] = true
			postMessage := &slack.PostMessageRequest{
				Channel:  data.Channel,
				Text:     fmt.Sprintf(firstMessage, u.ExternalID, u.Name, conf.Options.ExternalAddress),
				Username: botName,
			}
			_, err = s.PostMessage(postMessage, false)
			if err != nil {
				logrus.Infof("Unable to send first message to Slack - %v\n", err)
				return err
			}
		}
	}
	return nil
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
	message.IconURL = conf.Options.ExternalAddress + "/img/favicon/D icon 5757.png"
	s, err := slack.New(slack.SetToken(u.Token))
	if err != nil {
		return err
	}
	err = b.maybeSendFirstMessage(u, s, data)
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
