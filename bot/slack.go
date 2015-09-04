package bot

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/slack"
	"github.com/slavikm/govt"
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

func (b *Bot) handleFileReply(reply *domain.WorkReply, data *domain.Context, sub *subscription, verbose bool) {
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
	shouldPost := false
	fileMessage := fmt.Sprintf(comment, reply.File.Details.Name, fmt.Sprintf("<%s|Details>", link))
	postMessage := &slack.PostMessageRequest{
		Channel:     data.Channel,
		Attachments: []slack.Attachment{{Fallback: fileMessage, Text: fileMessage, Color: color}},
	}
	if data.Channel != "" {
		if verbose {
			shouldPost = true
			if !reply.MD5.XFE.NotFound && reply.MD5.XFE.Error == "" {
				xfeColor := "good"
				if len(reply.MD5.XFE.Malware.Family) > 0 {
					xfeColor = "danger"
				}
				postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
					Fallback:   fmt.Sprintf("Mime Type: %s, Family: %s", reply.MD5.XFE.Malware.MimeType, strings.Join(reply.MD5.XFE.Malware.Family, ",")),
					Color:      xfeColor,
					AuthorName: "XFE",
					AuthorLink: fmt.Sprintf("https://exchange.xforce.ibmcloud.com/malware/%s", reply.MD5.Details),
					Title:      "IBM X-Force Exchange",
					TitleLink:  fmt.Sprintf("https://exchange.xforce.ibmcloud.com/malware/%s", reply.MD5.Details),
					Fields: []slack.AttachmentField{
						slack.AttachmentField{Title: "Family", Value: strings.Join(reply.MD5.XFE.Malware.Family, ","), Short: true},
						slack.AttachmentField{Title: "MIME Type", Value: reply.MD5.XFE.Malware.MimeType, Short: true},
						slack.AttachmentField{Title: "Created", Value: reply.MD5.XFE.Malware.Created.String(), Short: true},
					},
				})
			}
			if reply.MD5.VT.FileReport.ResponseCode == 1 {
				vtColor := "good"
				if reply.MD5.VT.FileReport.Positives >= numOfPositivesToConvictForFiles {
					vtColor = "danger"
				}
				postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
					Fallback:  fmt.Sprintf("Scan Date: %s, Positives: %v, Total: %v", reply.MD5.VT.FileReport.ScanDate, reply.MD5.VT.FileReport.Positives, reply.MD5.VT.FileReport.Total),
					Color:     vtColor,
					Title:     "VirusTotal",
					TitleLink: reply.MD5.VT.FileReport.Permalink,
					Fields: []slack.AttachmentField{
						slack.AttachmentField{Title: "Scan Date", Value: reply.MD5.VT.FileReport.ScanDate, Short: true},
						slack.AttachmentField{Title: "Positives", Value: fmt.Sprintf("%v", reply.MD5.VT.FileReport.Positives), Short: true},
						slack.AttachmentField{Title: "Total", Value: fmt.Sprintf("%v", reply.MD5.VT.FileReport.Total), Short: true},
					},
				})
			}
			if reply.File.Virus != "" {
				postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
					Fallback:   fmt.Sprintf("Virus name: %s", reply.File.Virus),
					Text:       fmt.Sprintf("Virus name: %s", reply.File.Virus),
					Color:      "danger",
					AuthorName: "ClamAV",
					Title:      "ClamAV",
				})
			}
		} else if reply.File.Result != domain.ResultClean {
			shouldPost = true
		}
	}
	if shouldPost {
		err := b.post(postMessage, reply, data, sub)
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

// IPByDate sorting
type IPByDate []govt.DetectedUrl

func (a IPByDate) Len() int           { return len(a) }
func (a IPByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a IPByDate) Less(i, j int) bool { return a[i].ScanDate < a[j].ScanDate }

func (b *Bot) relevantUser(ctx *domain.Context) *subscription {
	b.mu.RLock()
	defer b.mu.RUnlock()
	subs := b.subscriptions[ctx.Team]
	if subs == nil {
		return nil
	}
	sub := subs.SubForUser(ctx.OriginalUser)
	if sub != nil {
		return sub
	}
	return subs.SubForUser(ctx.User)
}

func nilOrUnknown(v interface{}) string {
	if v == nil {
		return "Unknown"
	}
	return fmt.Sprintf("%v", v)
}

func (b *Bot) handleReply(reply *domain.WorkReply) {
	logrus.Debugf("Handling reply - %s", reply.MessageID)
	data, err := GetContext(reply.Context)
	if err != nil {
		logrus.Warnf("Error getting context from reply - %+v\n", reply)
		return
	}
	b.handleReplyStats(reply, data)
	sub := b.relevantUser(data)
	if sub == nil {
		logrus.Warnf("User not found in subscriptions for message %s", reply.MessageID)
	}
	verbose := false
	if data.Channel != "" {
		if data.Channel[0] == 'D' {
			verbose = sub.interest.VerboseIM
		} else {
			verbose, err = b.r.IsVerboseChannel(data.Team, data.Channel)
			if err != nil {
				logrus.Warnf("Error reading verbose channel status %v", err)
				verbose = false
			}
		}
	}
	if reply.Type&domain.ReplyTypeFile > 0 {
		b.handleFileReply(reply, data, sub, verbose)
	} else {
		link := fmt.Sprintf("%s/details?c=%s&m=%s&u=%s", conf.Options.ExternalAddress, data.Channel, reply.MessageID, data.User)
		postMessage := &slack.PostMessageRequest{Channel: data.Channel}
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
			if verbose {
				if !reply.URL.XFE.NotFound && reply.URL.XFE.Error == "" {
					xfeColor := "good"
					if reply.URL.XFE.URLDetails.Score >= xfeScoreToConvict {
						xfeColor = "danger"
					}
					postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
						Fallback: fmt.Sprintf("Score: %v, A Records: %s, Categories: %s",
							reply.URL.XFE.URLDetails.Score,
							strings.Join(reply.URL.XFE.Resolve.A, ","),
							joinMap(reply.URL.XFE.URLDetails.Cats)),
						Color:     xfeColor,
						Title:     "IBM X-Force Exchange",
						TitleLink: fmt.Sprintf("https://exchange.xforce.ibmcloud.com/url/%s", reply.URL.Details),
						Fields: []slack.AttachmentField{
							slack.AttachmentField{Title: "Score", Value: fmt.Sprintf("%v", reply.URL.XFE.URLDetails.Score), Short: true},
							slack.AttachmentField{Title: "A Records", Value: strings.Join(reply.URL.XFE.Resolve.A, ","), Short: true},
							slack.AttachmentField{Title: "Categories", Value: joinMap(reply.URL.XFE.URLDetails.Cats), Short: true},
						},
					})
					if len(reply.URL.XFE.Resolve.AAAA) > 0 {
						postMessage.Attachments[len(postMessage.Attachments)-1].Fields = append(postMessage.Attachments[len(postMessage.Attachments)-1].Fields,
							slack.AttachmentField{Title: "A Records", Value: strings.Join(reply.URL.XFE.Resolve.AAAA, ","), Short: true})
					}
				}
				if reply.URL.VT.URLReport.ResponseCode == 1 {
					vtColor := "good"
					if reply.URL.VT.URLReport.Positives >= numOfPositivesToConvict {
						vtColor = "danger"
					}
					postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
						Fallback:  fmt.Sprintf("Scan Date: %s, Positives: %v, Total: %v", reply.URL.VT.URLReport.ScanDate, reply.URL.VT.URLReport.Positives, reply.URL.VT.URLReport.Total),
						Color:     vtColor,
						Title:     "VirusTotal",
						TitleLink: reply.URL.VT.URLReport.Permalink,
						Fields: []slack.AttachmentField{
							slack.AttachmentField{Title: "Scan Date", Value: reply.URL.VT.URLReport.ScanDate, Short: true},
							slack.AttachmentField{Title: "Positives", Value: fmt.Sprintf("%v", reply.URL.VT.URLReport.Positives), Short: true},
							slack.AttachmentField{Title: "Total", Value: fmt.Sprintf("%v", reply.URL.VT.URLReport.Total), Short: true},
						},
					})
				}
			}
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
			if verbose {
				if !reply.IP.XFE.NotFound && reply.MD5.XFE.Error == "" {
					xfeColor := "good"
					if reply.IP.XFE.IPReputation.Score >= xfeScoreToConvict {
						xfeColor = "danger"
					}
					postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
						Fallback: fmt.Sprintf("Score: %v, Categories: %s, Geo: %v",
							reply.IP.XFE.IPReputation.Score, joinMapInt(reply.IP.XFE.IPReputation.Cats), nilOrUnknown(reply.IP.XFE.IPReputation.Geo["country"])),
						Color:     xfeColor,
						Title:     "IBM X-Force Exchange",
						TitleLink: fmt.Sprintf("https://exchange.xforce.ibmcloud.com/ip/%s", reply.IP.Details),
						Fields: []slack.AttachmentField{
							slack.AttachmentField{Title: "Score", Value: fmt.Sprintf("%v", reply.IP.XFE.IPReputation.Score), Short: true},
							slack.AttachmentField{Title: "Categories", Value: joinMapInt(reply.IP.XFE.IPReputation.Cats), Short: true},
							slack.AttachmentField{Title: "Geo", Value: nilOrUnknown(reply.IP.XFE.IPReputation.Geo["country"]), Short: true},
						},
					})
				}
				if reply.IP.VT.IPReport.ResponseCode == 1 {
					var vtPositives uint16
					listOfURLs := ""
					now := time.Now()
					detectedURLs := reply.IP.VT.IPReport.DetectedUrls
					sort.Sort(sort.Reverse(IPByDate(detectedURLs)))
					for i := range detectedURLs {
						t, err := time.Parse("2006-01-02 15:04:05", detectedURLs[i].ScanDate)
						if err != nil {
							logrus.Debugf("Error parsing scan date - %v", err)
							continue
						}
						if detectedURLs[i].Positives > vtPositives && t.Add(365*24*time.Hour).After(now) {
							vtPositives = detectedURLs[i].Positives
						}
						if i < 20 {
							listOfURLs += fmt.Sprintf("URL: %s, Positives: %v, Total: %v, Date: %s", detectedURLs[i].Url, detectedURLs[i].Positives, detectedURLs[i].Total, detectedURLs[i].ScanDate) + "\n"
						}
					}
					vtColor := "good"
					if vtPositives >= numOfPositivesToConvict {
						vtColor = "danger"
					}
					postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
						Fallback:   listOfURLs,
						Text:       listOfURLs,
						Color:      vtColor,
						AuthorName: "VirusTotal",
						AuthorLink: "https://www.virustotal.com/en/search?query=" + reply.IP.Details,
						Title:      "VirusTotal",
						TitleLink:  "https://www.virustotal.com/en/search?query=" + reply.IP.Details,
					})
				}
			}
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
			if verbose {
				if !reply.MD5.XFE.NotFound && reply.MD5.XFE.Error == "" {
					xfeColor := "good"
					if len(reply.MD5.XFE.Malware.Family) > 0 {
						xfeColor = "danger"
					}
					postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
						Fallback:   fmt.Sprintf("Mime Type: %s, Family: %s", reply.MD5.XFE.Malware.MimeType, strings.Join(reply.MD5.XFE.Malware.Family, ",")),
						Color:      xfeColor,
						AuthorName: "XFE",
						AuthorLink: fmt.Sprintf("https://exchange.xforce.ibmcloud.com/malware/%s", reply.MD5.Details),
						Title:      "IBM X-Force Exchange",
						TitleLink:  fmt.Sprintf("https://exchange.xforce.ibmcloud.com/malware/%s", reply.MD5.Details),
						Fields: []slack.AttachmentField{
							slack.AttachmentField{Title: "Family", Value: strings.Join(reply.MD5.XFE.Malware.Family, ","), Short: true},
							slack.AttachmentField{Title: "MIME Type", Value: reply.MD5.XFE.Malware.MimeType, Short: true},
							slack.AttachmentField{Title: "Created", Value: reply.MD5.XFE.Malware.Created.String(), Short: true},
						},
					})
				}
				if reply.MD5.VT.FileReport.ResponseCode == 1 {
					vtColor := "good"
					if reply.MD5.VT.FileReport.Positives >= numOfPositivesToConvictForFiles {
						vtColor = "danger"
					}
					postMessage.Attachments = append(postMessage.Attachments, slack.Attachment{
						Fallback:   fmt.Sprintf("Scan Date: %s, Positives: %v, Total: %v", reply.MD5.VT.FileReport.ScanDate, reply.MD5.VT.FileReport.Positives, reply.MD5.VT.FileReport.Total),
						Color:      vtColor,
						AuthorName: "VirusTotal",
						AuthorLink: reply.MD5.VT.FileReport.Permalink,
						Title:      "VirusTotal",
						TitleLink:  reply.MD5.VT.FileReport.Permalink,
						Fields: []slack.AttachmentField{
							slack.AttachmentField{Title: "Scan Date", Value: reply.MD5.VT.FileReport.ScanDate, Short: true},
							slack.AttachmentField{Title: "Positives", Value: fmt.Sprintf("%v", reply.MD5.VT.FileReport.Positives), Short: true},
							slack.AttachmentField{Title: "Total", Value: fmt.Sprintf("%v", reply.MD5.VT.FileReport.Total), Short: true},
						},
					})
				}
			}
		}
		clean := true
		if !verbose {
			for i := range postMessage.Attachments {
				if postMessage.Attachments[i].Color != "good" {
					clean = false
					break
				}
			}
		}
		if verbose || !clean {
			err = b.post(postMessage, reply, data, sub)
			if err != nil {
				logrus.Errorf("Unable to send message to Slack - %v\n", err)
				return
			}
		} else {
			logrus.Debugf("Reply %s clean, ignoring", reply.MessageID)
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
func (b *Bot) post(message *slack.PostMessageRequest, reply *domain.WorkReply, data *domain.Context, sub *subscription) error {
	u := sub.user
	message.IconURL = conf.Options.ExternalAddress + "/img/favicon/D%20icon%205757.png"
	message.Text = mainMessageFormatted()
	message.Username = botName

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
