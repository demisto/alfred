package bot

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/slack"
	"github.com/demisto/alfred/util"
	"github.com/slavikm/govt"
)

const (
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
	ipCommentPrivate   = "IP (%s) is a private (internal) IP so we cannot provide reputation information: %s."
	hashCommentGood    = "Hash (%s) is clean: %s."
	hashCommentBad     = "Warning: hash (%s) is malicious: %s."
	hashCommentWarning = "Unable to find details regarding this hash (%s): %s."
	mainMessage        = "Security check by DBot - Demisto Bot. Click <%s|here> for configuration and details."
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

func joinMapFloat32(m map[string]float32) string {
	res := ""
	for k, v := range m {
		res += fmt.Sprintf("%s: %v,", k, v)
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
	// First, make sure it is a valid reply and if not, do nothing
	if len(reply.Hashes) != 1 {
		logrus.Warnf("Weird, invalid reply with no MD5 part - %+v", reply)
		return
	}
	link := fmt.Sprintf("%s/details?f=%s&t=%s", conf.Options.ExternalAddress, reply.File.Details.ID, data.Team)
	color := "warning"
	comment := fileCommentWarning
	shouldPost := false
	if reply.File.FileTooLarge {
		comment = fileCommentBig
		shouldPost = true
	} else if reply.File.Result == domain.ResultDirty {
		color = "danger"
		comment = fileCommentBad
	} else if reply.File.Result == domain.ResultClean {
		// At least one of reputation services found this to be known good
		// Keep the default
		color = "good"
		comment = fileCommentGood
	}
	fileMessage := fmt.Sprintf(comment, reply.File.Details.Name, fmt.Sprintf("<%s&text=%s|Details>", link, url.QueryEscape(reply.Hashes[0].Details)))
	attachments := []map[string]interface{}{{"fallback": fileMessage, "text": fileMessage, "color": color}}
	postMessage := map[string]interface{}{"channel": data.Channel}
	if data.Channel != "" {
		if reply.Hashes[0].Cy.Error == "" && reply.Hashes[0].Cy.Result.StatusCode == 1 {
			cyColor := "good"
			if reply.Hashes[0].Cy.Result.GeneralScore < cyScoreToConvict {
				cyColor = "danger"
			}
			attachments = append(attachments, map[string]interface{}{
				"fallback":   fmt.Sprintf("Score: %v, Classifiers: %v", reply.Hashes[0].Cy.Result.GeneralScore, reply.Hashes[0].Cy.Result.Classifiers),
				"color":      cyColor,
				"title":      "Cylance Infinity",
				"title_link": "https://www.cylance.com",
				"fields": []map[string]interface{}{
					{"title": "Score", "value": fmt.Sprintf("%v", reply.Hashes[0].Cy.Result.GeneralScore), "short": true},
					{"title": "Classifiers", "value": joinMapFloat32(reply.Hashes[0].Cy.Result.Classifiers), "short": true},
				},
			})
		}
		if !reply.Hashes[0].XFE.NotFound && reply.Hashes[0].XFE.Error == "" {
			xfeColor := "good"
			if len(reply.Hashes[0].XFE.Malware.Family) > 0 || len(reply.Hashes[0].XFE.Malware.Origins.External.Family) > 0 {
				xfeColor = "danger"
			}
			attachments = append(attachments, map[string]interface{}{
				"fallback":   fmt.Sprintf("Mime Type: %s, Family: %s", reply.Hashes[0].XFE.Malware.MimeType, strings.Join(reply.Hashes[0].XFE.Malware.Family, ",")),
				"color":      xfeColor,
				"title":      "IBM X-Force Exchange",
				"title_link": fmt.Sprintf("https://exchange.xforce.ibmcloud.com/malware/%s", reply.Hashes[0].Details),
				"fields": []map[string]interface{}{
					{"title": "Family", "value": strings.Join(reply.Hashes[0].XFE.Malware.Family, ","), "short": true},
					{"title": "MIME Type", "value": reply.Hashes[0].XFE.Malware.MimeType, "short": true},
					{"title": "Created", "value": reply.Hashes[0].XFE.Malware.Created.String(), "short": true},
				},
			})
		}
		if reply.Hashes[0].VT.FileReport.ResponseCode == 1 {
			vtColor := "good"
			if reply.Hashes[0].VT.FileReport.Positives >= numOfPositivesToConvictForFiles {
				vtColor = "danger"
			}
			attachments = append(attachments, map[string]interface{}{
				"fallback":   fmt.Sprintf("Scan Date: %s, Positives: %v, Total: %v", reply.Hashes[0].VT.FileReport.ScanDate, reply.Hashes[0].VT.FileReport.Positives, reply.Hashes[0].VT.FileReport.Total),
				"color":      vtColor,
				"title":      "VirusTotal",
				"title_link": reply.Hashes[0].VT.FileReport.Permalink,
				"fields": []map[string]interface{}{
					{"title": "Scan Date", "value": reply.Hashes[0].VT.FileReport.ScanDate, "short": true},
					{"title": "Positives", "value": fmt.Sprintf("%v", reply.Hashes[0].VT.FileReport.Positives), "short": true},
					{"title": "Total", "value": fmt.Sprintf("%v", reply.Hashes[0].VT.FileReport.Total), "short": true},
				},
			})
		}
		if reply.File.Virus != "" {
			attachments = append(attachments, map[string]interface{}{
				"fallback":    fmt.Sprintf("Virus name: %s", reply.File.Virus),
				"text":        fmt.Sprintf("Virus name: %s", reply.File.Virus),
				"color":       "danger",
				"author_name": "ClamAV",
				"title":       "ClamAV",
			})
		}
		if verbose {
			shouldPost = true
		} else if reply.File.Result == domain.ResultDirty {
			shouldPost = true
		}
	}
	if shouldPost {
		postMessage["attachments"] = attachments
		err := b.post(postMessage, reply, data, sub)
		if err != nil {
			logrus.Errorf("Unable to send message to Slack - %v\n", err)
			return
		}
	}
}

func (b *Bot) handleReplyStats(reply *domain.WorkReply, sub *subscription) {
	b.smu.Lock()
	defer b.smu.Unlock()
	stats, ok := b.stats[sub.team.ExternalID]
	if !ok {
		stats = &domain.Statistics{Team: sub.team.ID}
		b.stats[sub.team.ExternalID] = stats
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
		for i := range reply.Hashes {
			if reply.Hashes[i].Result == domain.ResultClean {
				stats.HashesClean++
			} else if reply.Hashes[i].Result == domain.ResultDirty {
				stats.HashesDirty++
			} else {
				stats.HashesUnknown++
			}
		}
		for i := range reply.URLs {
			if reply.URLs[i].Result == domain.ResultClean {
				stats.URLsClean++
			} else if reply.URLs[i].Result == domain.ResultDirty {
				stats.URLsDirty++
			} else {
				stats.URLsUnknown++
			}
		}
		for i := range reply.IPs {
			if reply.IPs[i].Result == domain.ResultClean {
				stats.IPsClean++
			} else if reply.IPs[i].Result == domain.ResultDirty {
				stats.IPsDirty++
			} else {
				stats.IPsUnknown++
			}
		}
	}
}

func (b *Bot) handleConvicted(reply *domain.WorkReply, ctx *domain.Context, sub *subscription) {
	if reply.Type&domain.ReplyTypeFile > 0 && reply.File.Result == domain.ResultDirty {
		// First, make sure it is a valid reply and if not, do nothing
		if len(reply.Hashes) != 1 {
			logrus.Warnf("Weird, invalid reply with no MD5 part - %+v", reply)
			return
		}
		vtScore := fmt.Sprintf("%v / %v", reply.Hashes[0].VT.FileReport.Positives, reply.Hashes[0].VT.FileReport.Total)
		xfeScore := strings.Join(reply.Hashes[0].XFE.Malware.Family, ",")
		cyScore := fmt.Sprintf("%v - %v", reply.Hashes[0].Cy.Result.GeneralScore, reply.Hashes[0].Cy.Result.Classifiers)
		if err := b.r.StoreMaliciousContent(&domain.MaliciousContent{
			Team:        sub.team.ID,
			Channel:     ctx.Channel,
			MessageID:   reply.File.Details.ID,
			ContentType: domain.ReplyTypeFile,
			Content:     reply.Hashes[0].Details,
			FileName:    reply.File.Details.Name,
			VT:          vtScore,
			XFE:         xfeScore,
			Cy:          cyScore,
			ClamAV:      reply.File.Virus}); err != nil {
			logrus.WithError(err).Warnf("Unable to store convicted for team [%s]", sub.team.ID)
		}
	} else {
		for i := range reply.Hashes {
			if reply.Hashes[i].Result == domain.ResultDirty {
				vtScore := fmt.Sprintf("%v / %v", reply.Hashes[i].VT.FileReport.Positives, reply.Hashes[i].VT.FileReport.Total)
				xfeScore := strings.Join(reply.Hashes[i].XFE.Malware.Family, ",")
				cyScore := fmt.Sprintf("%v - %v", reply.Hashes[i].Cy.Result.GeneralScore, reply.Hashes[i].Cy.Result.Classifiers)
				if err := b.r.StoreMaliciousContent(&domain.MaliciousContent{
					Team:        sub.team.ID,
					Channel:     ctx.Channel,
					MessageID:   reply.MessageID,
					ContentType: domain.ReplyTypeHash,
					Content:     reply.Hashes[i].Details,
					VT:          vtScore,
					XFE:         xfeScore,
					Cy:          cyScore}); err != nil {
					logrus.WithError(err).Warnf("Unable to store convicted for team [%s]", sub.team.ID)
				}
			}
		}
		for i := range reply.URLs {
			if reply.URLs[i].Result == domain.ResultDirty {
				vtScore := fmt.Sprintf("%v / %v", reply.URLs[i].VT.URLReport.Positives, reply.URLs[i].VT.URLReport.Total)
				xfeScore := fmt.Sprintf("%v", reply.URLs[i].XFE.URLDetails.Score)
				if err := b.r.StoreMaliciousContent(&domain.MaliciousContent{
					Team:        sub.team.ID,
					Channel:     ctx.Channel,
					MessageID:   reply.MessageID,
					ContentType: domain.ReplyTypeURL,
					Content:     reply.URLs[i].Details,
					VT:          vtScore,
					XFE:         xfeScore}); err != nil {
					logrus.WithError(err).Warnf("Unable to store convicted for team [%s]", sub.team.ID)
				}
			}
		}
		for i := range reply.IPs {
			if reply.IPs[i].Result == domain.ResultDirty {
				vtScore := fmt.Sprintf("%v", len(reply.IPs[i].VT.IPReport.DetectedUrls))
				xfeScore := fmt.Sprintf("%v", reply.IPs[i].XFE.IPReputation.Score)
				if err := b.r.StoreMaliciousContent(&domain.MaliciousContent{
					Team:        sub.team.ID,
					Channel:     ctx.Channel,
					MessageID:   reply.MessageID,
					ContentType: domain.ReplyTypeIP,
					Content:     reply.IPs[i].Details,
					VT:          vtScore,
					XFE:         xfeScore}); err != nil {
					logrus.WithError(err).Warnf("Unable to store convicted for team [%s]", sub.team.ID)
				}
			}
		}
	}
}

// IPByDate sorting
type IPByDate []govt.DetectedUrl

func (a IPByDate) Len() int           { return len(a) }
func (a IPByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a IPByDate) Less(i, j int) bool { return a[i].ScanDate < a[j].ScanDate }

func (b *Bot) relevantTeam(team string) *subscription {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.subscriptions[team]
}

func nilOrUnknown(v interface{}) string {
	if v == nil {
		return "Unknown"
	}
	return fmt.Sprintf("%v", v)
}

func defangURL(u string) string {
	return strings.Replace(strings.Replace(strings.Replace(u, "https://", "https[://]", 1), "http://", "http[://]", 1), ".", "[.]", -1)
}

func (b *Bot) handleReply(reply *domain.WorkReply) {
	logrus.Debugf("Handling reply - %s", reply.MessageID)
	data, err := domain.GetContext(reply.Context)
	if err != nil {
		logrus.Warnf("Error getting context from reply - %+v\n", reply)
		return
	}
	sub := b.relevantTeam(data.Team)
	if sub == nil {
		logrus.Warnf("Team not found in subscriptions for message %s", reply.MessageID)
	}
	b.handleReplyStats(reply, sub)
	b.handleConvicted(reply, data, sub)
	verbose := false
	if data.Channel != "" {
		if data.Channel[0] == 'D' {
			// Since it's a direct message to me, I need to reply verbose
			verbose = true
		} else {
			verbose = sub.configuration.IsVerbose(data.Channel)
		}
	}
	if reply.Type&domain.ReplyTypeFile > 0 {
		b.handleFileReply(reply, data, sub, verbose)
	} else {
		link := fmt.Sprintf("%s/details?c=%s&m=%s&t=%s", conf.Options.ExternalAddress, data.Channel, reply.MessageID, sub.team.ID)
		postMessage := slack.Response{"channel": data.Channel}
		attachments := make([]map[string]interface{}, 0)
		for i := range reply.URLs {
			color := "warning"
			comment := urlCommentWarning
			if reply.URLs[i].Result == domain.ResultDirty {
				color = "danger"
				comment = urlCommentBad
			} else if reply.URLs[i].Result == domain.ResultClean {
				color = "good"
				comment = urlCommentGood
			}
			urlMessage := fmt.Sprintf(comment, defangURL(reply.URLs[i].Details), fmt.Sprintf("<%s&text=%s|Details>", link, url.QueryEscape("<"+reply.URLs[i].Details+">")))
			if verbose || color != "good" {
				attachments = append(attachments, map[string]interface{}{
					"fallback": urlMessage,
					"text":     urlMessage,
					"color":    color,
				})
			}
			if verbose {
				if !reply.URLs[i].XFE.NotFound && reply.URLs[i].XFE.Error == "" {
					xfeColor := "good"
					if reply.URLs[i].XFE.URLDetails.Score >= xfeScoreToConvict {
						xfeColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback": fmt.Sprintf("Score: %v, A Records: %s, Categories: %s",
							reply.URLs[i].XFE.URLDetails.Score,
							strings.Join(reply.URLs[i].XFE.Resolve.A, ","),
							joinMap(reply.URLs[i].XFE.URLDetails.Cats)),
						"color":      xfeColor,
						"title":      "IBM X-Force Exchange",
						"title_link": fmt.Sprintf("https://exchange.xforce.ibmcloud.com/url/%s", reply.URLs[i].Details),
						"fields": []map[string]interface{}{
							{"title": "Score", "value": fmt.Sprintf("%v", reply.URLs[i].XFE.URLDetails.Score), "short": true},
							{"title": "A Records", "value": strings.Join(reply.URLs[i].XFE.Resolve.A, ","), "short": true},
							{"title": "Categories", "value": joinMap(reply.URLs[i].XFE.URLDetails.Cats), "short": true},
						},
					})
					if len(reply.URLs[i].XFE.Resolve.AAAA) > 0 {
						attachments[len(attachments)-1]["fields"] = append(attachments[len(attachments)-1]["fields"].([]map[string]interface{}),
							map[string]interface{}{"title": "A Records", "value": strings.Join(reply.URLs[i].XFE.Resolve.AAAA, ","), "short": true})
					}
				}
				if reply.URLs[i].VT.URLReport.ResponseCode == 1 {
					vtColor := "good"
					if reply.URLs[i].VT.URLReport.Positives >= numOfPositivesToConvict {
						vtColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback":   fmt.Sprintf("Scan Date: %s, Positives: %v, Total: %v", reply.URLs[i].VT.URLReport.ScanDate, reply.URLs[i].VT.URLReport.Positives, reply.URLs[i].VT.URLReport.Total),
						"color":      vtColor,
						"title":      "VirusTotal",
						"title_link": reply.URLs[i].VT.URLReport.Permalink,
						"fields": []map[string]interface{}{
							{"title": "Scan Date", "value": reply.URLs[i].VT.URLReport.ScanDate, "short": true},
							{"title": "Positives", "value": fmt.Sprintf("%v", reply.URLs[i].VT.URLReport.Positives), "short": true},
							{"title": "Total", "value": fmt.Sprintf("%v", reply.URLs[i].VT.URLReport.Total), "short": true},
						},
					})
				}
			}
		}
		for i := range reply.IPs {
			color := "warning"
			comment := ipCommentWarning
			if reply.IPs[i].Private {
				color = "good"
				comment = ipCommentPrivate
			} else if reply.IPs[i].Result == domain.ResultDirty {
				color = "danger"
				comment = ipCommentBad
			} else if reply.IPs[i].Result == domain.ResultClean {
				color = "good"
				comment = ipCommentGood
			}
			ipMessage := fmt.Sprintf(comment, reply.IPs[i].Details, fmt.Sprintf("<%s&text=%s|Details>", link, url.QueryEscape(reply.IPs[i].Details)))
			if verbose || color != "good" {
				attachments = append(attachments, map[string]interface{}{
					"fallback": ipMessage,
					"text":     ipMessage,
					"color":    color,
				})
			}
			if verbose {
				if !reply.IPs[i].XFE.NotFound && reply.IPs[i].XFE.Error == "" {
					xfeColor := "good"
					if reply.IPs[i].XFE.IPReputation.Score >= xfeScoreToConvict {
						xfeColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback": fmt.Sprintf("Score: %v, Categories: %s, Geo: %v",
							reply.IPs[i].XFE.IPReputation.Score, joinMapInt(reply.IPs[i].XFE.IPReputation.Cats), nilOrUnknown(reply.IPs[i].XFE.IPReputation.Geo["country"])),
						"color":      xfeColor,
						"title":      "IBM X-Force Exchange",
						"title_link": fmt.Sprintf("https://exchange.xforce.ibmcloud.com/ip/%s", reply.IPs[i].Details),
						"fields": []map[string]interface{}{
							{"title": "Score", "value": fmt.Sprintf("%v", reply.IPs[i].XFE.IPReputation.Score), "short": true},
							{"title": "Categories", "value": joinMapInt(reply.IPs[i].XFE.IPReputation.Cats), "short": true},
							{"title": "Geo", "value": nilOrUnknown(reply.IPs[i].XFE.IPReputation.Geo["country"]), "short": true},
						},
					})
				}
				if reply.IPs[i].VT.IPReport.ResponseCode == 1 {
					var vtPositives uint16
					listOfURLs := ""
					now := time.Now()
					detectedURLs := reply.IPs[i].VT.IPReport.DetectedUrls
					sort.Sort(sort.Reverse(IPByDate(detectedURLs)))
					for j := range detectedURLs {
						t, err := time.Parse("2006-01-02 15:04:05", detectedURLs[j].ScanDate)
						if err != nil {
							logrus.Debugf("Error parsing scan date - %v", err)
							continue
						}
						if detectedURLs[j].Positives > vtPositives && t.Add(365*24*time.Hour).After(now) {
							vtPositives = detectedURLs[j].Positives
						}
						if j < 20 {
							listOfURLs += fmt.Sprintf("URL: %s, Positives: %v, Total: %v, Date: %s", defangURL(detectedURLs[j].Url), detectedURLs[j].Positives, detectedURLs[j].Total, detectedURLs[j].ScanDate) + "\n"
						}
					}
					vtColor := "good"
					if vtPositives >= numOfPositivesToConvict {
						vtColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback":   listOfURLs,
						"text":       listOfURLs,
						"color":      vtColor,
						"title":      "VirusTotal",
						"title_link": "https://www.virustotal.com/en/search?query=" + reply.IPs[i].Details,
					})
				}
			}
		}
		// We will handle hashes only for verbose channels
		if verbose {
			for i := range reply.Hashes {
				color := "warning"
				comment := hashCommentWarning
				if reply.Hashes[i].Result == domain.ResultDirty {
					color = "danger"
					comment = hashCommentBad
				} else if reply.Hashes[i].Result == domain.ResultClean {
					color = "good"
					comment = hashCommentGood
				}
				hashMessage := fmt.Sprintf(comment, reply.Hashes[i].Details, fmt.Sprintf("<%s&text=%s|Details>", link, url.QueryEscape(reply.Hashes[i].Details)))
				attachments = append(attachments, map[string]interface{}{
					"fallback": hashMessage,
					"text":     hashMessage,
					"color":    color,
				})
				if reply.Hashes[i].Cy.Error == "" && reply.Hashes[0].Cy.Result.StatusCode == 1 {
					cyColor := "good"
					if reply.Hashes[0].Cy.Result.GeneralScore < cyScoreToConvict {
						cyColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback":   fmt.Sprintf("Score: %v, Classifiers: %v", reply.Hashes[0].Cy.Result.GeneralScore, reply.Hashes[0].Cy.Result.Classifiers),
						"color":      cyColor,
						"title":      "Cylance Infinity",
						"title_link": "https://www.cylance.com",
						"fields": []map[string]interface{}{
							{"title": "Score", "value": fmt.Sprintf("%v", reply.Hashes[0].Cy.Result.GeneralScore), "short": true},
							{"title": "Classifiers", "value": joinMapFloat32(reply.Hashes[0].Cy.Result.Classifiers), "short": true},
						},
					})
				}
				if !reply.Hashes[i].XFE.NotFound && reply.Hashes[i].XFE.Error == "" {
					xfeColor := "good"
					if len(reply.Hashes[i].XFE.Malware.Family) > 0 || len(reply.Hashes[i].XFE.Malware.Origins.External.Family) > 0 {
						xfeColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback":   fmt.Sprintf("Mime Type: %s, Family: %s", reply.Hashes[i].XFE.Malware.MimeType, strings.Join(reply.Hashes[i].XFE.Malware.Family, ",")),
						"color":      xfeColor,
						"title":      "IBM X-Force Exchange",
						"title_link": fmt.Sprintf("https://exchange.xforce.ibmcloud.com/malware/%s", reply.Hashes[i].Details),
						"fields": []map[string]interface{}{
							{"title": "Family", "value": strings.Join(reply.Hashes[i].XFE.Malware.Family, ","), "short": true},
							{"title": "MIME Type", "value": reply.Hashes[i].XFE.Malware.MimeType, "short": true},
							{"title": "Created", "value": reply.Hashes[i].XFE.Malware.Created.String(), "short": true},
						},
					})
				}
				if reply.Hashes[i].VT.FileReport.ResponseCode == 1 {
					vtColor := "good"
					if reply.Hashes[i].VT.FileReport.Positives >= numOfPositivesToConvictForFiles {
						vtColor = "danger"
					}
					attachments = append(attachments, map[string]interface{}{
						"fallback":   fmt.Sprintf("Scan Date: %s, Positives: %v, Total: %v", reply.Hashes[i].VT.FileReport.ScanDate, reply.Hashes[i].VT.FileReport.Positives, reply.Hashes[i].VT.FileReport.Total),
						"color":      vtColor,
						"title":      "VirusTotal",
						"title_link": reply.Hashes[i].VT.FileReport.Permalink,
						"fields": []map[string]interface{}{
							{"title": "Scan Date", "value": reply.Hashes[i].VT.FileReport.ScanDate, "short": true},
							{"title": "Positives", "value": fmt.Sprintf("%v", reply.Hashes[i].VT.FileReport.Positives), "short": true},
							{"title": "Total", "value": fmt.Sprintf("%v", reply.Hashes[i].VT.FileReport.Total), "short": true},
						},
					})
				}
			}
		}
		clean := true
		if !verbose {
			for i := range attachments {
				if attachments[i]["color"] != "good" {
					clean = false
					break
				}
			}
		}
		if verbose || !clean {
			postMessage["attachments"] = attachments
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

// post uses the correct client to post to the channel
// See if the original message poster is subscribed and if so use him.
// If not, use the first user we have that is subscribed to the channel.
func (b *Bot) post(message map[string]interface{}, reply *domain.WorkReply, data *domain.Context, sub *subscription) error {
	message["text"] = mainMessageFormatted()
	message["as_user"] = true
	var err error
	_, err = sub.s.Do("POST", "chat.postMessage", message)
	return err
}

func parseChannels(sub *subscription, text string, pos int) ([]string, []string, error) {
	parts := strings.Split(text, " ")
	if len(parts) <= pos {
		return nil, nil, fmt.Errorf("not enough parameters in '%s'", text)
	}
	var channels []string
	conversations, err := sub.s.Conversations("public_channel,private_channel")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve the list of conversations - %v", err)
	}
	for i := pos; i < len(parts); i++ {
		subparts := strings.Split(parts[i], ",")
		for j := range subparts {
			subpart := strings.TrimSpace(subparts[j])
			if subpart != "" {
				var ch string
				switch {
				case strings.Contains(subpart, "<#"): // if this is #channel
					ch = subpart[2 : len(subpart)-1]
					if strings.Contains(ch, "|") {
						ch = strings.Split(ch, "|")[0]
					}
				case strings.HasPrefix(subpart, "#"): // if this is a group but someone chose # as start
					subpart = subpart[1:]
					fallthrough
				default:
					for _, conversation := range conversations {
						if strings.EqualFold(conversation.S("name"), subpart) {
							ch = conversation.S("id")
							break
						}
					}
				}
				if ch != "" {
					channels = append(channels, ch)
				}
			}
		}
	}
	return parts, channels, nil
}

func (b *Bot) joinChannels(team, text, channel string, sub *subscription) {
	postMessage := map[string]interface{}{
		"channel": channel,
		"as_user": true,
	}
	users, err := b.r.TeamMembers(sub.team.ID)
	if err != nil {
		logrus.Warnf("Unable to retrieve team members - %v", err)
		return
	}
	parts, incomingChannels, err := parseChannels(sub, text, 1)
	ch, err := sub.s.Conversations("")
	if err != nil {
		logrus.WithError(err).Warn("Error retrieving my channels")
		postMessage["text"] = "Error retrieving current configuration. Rest assured we are looking into the issue."
	} else {
		var channels []string
		var channelFound bool
	usersLoop:
		for i := range users {
			if users[i].Status == domain.UserStatusActive {
				s := &slack.Client{Token: users[i].Token}
				if err != nil {
					logrus.Infof("Error creating Slack client for user %s (%s) - %v\n", users[i].ID, users[i].Name, err)
					continue
				}
				for _, c := range ch {
					if !c.B("is_member") && !util.In(channels, c.S("name")) &&
						(strings.ToLower(parts[1]) == "all" || util.In(incomingChannels, c.S("id"))) {
						channelFound = true
						_, err = s.Do("POST", "conversations.invite", map[string]interface{}{
							"channel": c.S("id"),
							"users":   sub.team.BotUserID,
						})
						if err != nil {
							logrus.Infof("Error inviting us - %v\n", err)
							continue usersLoop
						}
						channels = append(channels, c.S("name"))
					}
				}
				break
			}
		}
		if len(channels) > 0 {
			text := fmt.Sprintf("I've started monitoring the following channels: %s", strings.Join(channels, ", "))
			postMessage["text"] = text
		} else {
			if channelFound {
				postMessage["text"] = "I could not invite myself to the public channels, rest assured we are looking into the issue."
			} else {
				postMessage["text"] = "I was already monitoring all public channels but thanks for thinking of me."
			}
		}
	}
	_, err = sub.s.Do("POST", "chat.postMessage", postMessage)
	if err != nil {
		logrus.Warnf("Error posting config message - %v", err)
	}
}

func (b *Bot) handleVerbose(team, text, channel string, sub *subscription) {
	postMessage := map[string]interface{}{
		"channel": channel,
		"as_user": true,
	}
	changed := false
	b.mu.Lock()
	defer b.mu.Unlock()
	parts, channels, err := parseChannels(sub, text, 2)
	if err != nil {
		postMessage["text"] = "I could not understand your command. Verbose command is:\nverbose on #channel1,#channel2 - to turn on verbose mode on for a list of channels.\nverbose off #channel1,#channel2 - to turn off verbose mode on for a list of channels."
	} else {
		for _, ch := range channels {
			if ch == "" {
				continue
			}
			if strings.ToLower(parts[1]) == "on" {
				if ch[0] == 'C' {
					if !util.In(sub.configuration.VerboseChannels, ch) {
						sub.configuration.VerboseChannels = append(sub.configuration.VerboseChannels, ch)
						changed = true
					}
				} else if ch[0] == 'G' {
					if !util.In(sub.configuration.VerboseGroups, ch) {
						sub.configuration.VerboseGroups = append(sub.configuration.VerboseGroups, ch)
						changed = true
					}
				}
			} else if strings.ToLower(parts[1]) == "off" {
				if ch[0] == 'C' {
					if util.In(sub.configuration.VerboseChannels, ch) {
						index := util.Index(sub.configuration.VerboseChannels, ch)
						if index >= 0 {
							sub.configuration.VerboseChannels = sub.configuration.VerboseChannels[:index+copy(sub.configuration.VerboseChannels[index:], sub.configuration.VerboseChannels[index+1:])]
						}
						changed = true
					}
				} else if ch[0] == 'G' {
					if util.In(sub.configuration.VerboseGroups, ch) {
						index := util.Index(sub.configuration.VerboseGroups, ch)
						if index >= 0 {
							sub.configuration.VerboseGroups = sub.configuration.VerboseGroups[:index+copy(sub.configuration.VerboseGroups[index:], sub.configuration.VerboseGroups[index+1:])]
						}
						changed = true
					}
				}
			}
		}
	}
	if changed {
		err := b.r.SetChannelsAndGroups(sub.configuration)
		if err != nil {
			logrus.Warnf("Error storing verbose configuration for team %s - %v", team, err)
			postMessage["text"] = "I had an issue saving the verbose state."
		} else {
			postMessage["text"] = "Verbose state was changed."
		}
	} else {
		postMessage["text"] = "Verbose state did not change - could not find anything new to change"
	}
	_, err = sub.s.Do("POST", "chat.postMessage", postMessage)
	if err != nil {
		logrus.Warnf("Error posting config message - %v", err)
	}
}

func (b *Bot) handleConfig(team string, msg slack.Response, sub *subscription) {
	postMessage := map[string]interface{}{
		"channel": msg.S("channel"),
		"as_user": true,
	}
	ch, err := sub.s.Conversations("public_channel,private_channel")
	if err != nil {
		logrus.Warnf("Error retrieving my channels - %v", err)
		postMessage["text"] = "Error retrieving configuration. Rest assured we are looking into the issue."
	} else {
		var channels []string
		var verboseChannels []string
		var groups []string
		var verboseGroups []string
		for _, c := range ch {
			if c.B("is_member") {
				if sub.configuration.IsVerbose(c.S("id")) {
					if c.B("is_channel") {
						verboseChannels = append(verboseChannels, c.S("name"))
					} else {
						verboseGroups = append(verboseGroups, c.S("name"))
					}
				} else {
					if c.B("is_channel") {
						channels = append(channels, c.S("name"))
					} else {
						groups = append(groups, c.S("name"))
					}
				}
			}
		}
		text := fmt.Sprintf("Channels I'm monitoring: %s", strings.Join(channels, ", "))
		if len(verboseChannels) > 0 {
			text = text + fmt.Sprintf("\nChannels I'm monitoring and providing extra info: %s", strings.Join(verboseChannels, ", "))
		}
		if len(groups) > 0 {
			text = text + fmt.Sprintf("\nPrivate channels I'm monitoring: %s", strings.Join(groups, ", "))
		}
		if len(verboseGroups) > 0 {
			text = text + fmt.Sprintf("\nPrivate channels I'm monitoring and providing extra info: %s", strings.Join(verboseGroups, ", "))
		}
		if sub.team.VTKey != "" {
			l := len(sub.team.VTKey)
			text = text + "\nUsing your own VirusTotal key ending with " + sub.team.VTKey[l-4:]
		}
		if sub.team.XFEKey != "" {
			l := len(sub.team.XFEKey)
			text = text + "\nUsing your own IBM X-Force Exchange key ending with " + sub.team.XFEKey[l-4:]
		}
		postMessage["text"] = text
	}
	if _, err = sub.s.Do("POST", "chat.postMessage", postMessage); err != nil {
		logrus.Warnf("Error posting config message - %v", err)
	}
}

func (b *Bot) handleVT(team, text, channel string, sub *subscription) {
	postMessage := map[string]interface{}{
		"channel": channel,
		"as_user": true,
	}
	parts := strings.Split(text, " ")
	if len(parts) == 2 {
		if parts[1] == "-" {
			sub.team.VTKey = ""
			err := b.r.SetTeam(sub.team)
			if err == nil {
				postMessage["text"] = "Cleared VT key - using default"
			} else {
				postMessage["text"] = "Error clearing VT key - no worries, we are handling it"
				logrus.WithError(err).Warnf("Unable to clear VT key for team %s", team)
			}
		} else {
			sub.team.VTKey = parts[1]
			err := b.r.SetTeam(sub.team)
			if err == nil {
				postMessage["text"] = "VT key set."
			} else {
				postMessage["text"] = "Error setting VT key - no worries, we are handling it"
				logrus.WithError(err).Warnf("Unable to set VT key for team %s", team)
			}
		}
	} else {
		postMessage["text"] = "Sorry, I could not understand you."
	}
	if _, err := sub.s.Do("POST", "chat.postMessage", postMessage); err != nil {
		logrus.Warnf("Error posting config message - %v", err)
	}
}

func (b *Bot) handleXFE(team, text, channel string, sub *subscription) {
	postMessage := map[string]interface{}{
		"channel": channel,
		"as_user": true,
	}
	parts := strings.Split(text, " ")
	if len(parts) == 2 && parts[1] == "-" || len(parts) == 3 && parts[1] == "-" {
		sub.team.XFEKey, sub.team.XFEPass = "", ""
		err := b.r.SetTeam(sub.team)
		if err == nil {
			postMessage["text"] = "Cleared XFE key - using default"
		} else {
			postMessage["text"] = "Error clearing XFE key - no worries, we are handling it"
			logrus.WithError(err).Warnf("Unable to clear XFE key for team %s", team)
		}
	} else if len(parts) == 3 {
		sub.team.XFEKey, sub.team.XFEPass = parts[1], parts[2]
		err := b.r.SetTeam(sub.team)
		if err == nil {
			postMessage["text"] = "XFE key set."
		} else {
			postMessage["text"] = "Error setting XFE key - no worries, we are handling it"
			logrus.WithError(err).Warnf("Unable to set XFE key for team %s", team)
		}
	} else {
		postMessage["text"] = "Sorry, I could not understand you."
	}
	if _, err := sub.s.Do("POST", "chat.postMessage", postMessage); err != nil {
		logrus.Warnf("Error posting config message - %v", err)
	}
}

func (b *Bot) showHelp(team, channel string) {
	postMessage := map[string]interface{}{
		"channel": channel,
		"as_user": true,
		"text":    conf.DefaultHelpMessage}
	sub := b.subscriptions[team]
	if _, err := sub.s.Do("POST", "chat.postMessage", postMessage); err != nil {
		logrus.Warnf("Error posting config message - %v", err)
	}
}
