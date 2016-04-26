package web

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/slack"
)

func (ac *AppContext) work(w http.ResponseWriter, r *http.Request) {
	team := r.FormValue("t")
	file := r.FormValue("f")
	message := r.FormValue("m")
	channel := r.FormValue("c")
	text := r.FormValue("text")
	if team == "" || file == "" && (message == "" || channel == "" || text == "") {
		WriteError(w, ErrBadRequest)
		return
	}
	// We need this just for a small verification that the team is one of ours
	t, err := ac.r.Team(team)
	if err != nil {
		logrus.Warnf("Error loading team - %v\n", err)
		WriteError(w, ErrInternalServer)
		return
	}
	var workReq *domain.WorkRequest
	// If we have the actual text to show details for
	if file == "" {
		workReq = &domain.WorkRequest{
			MessageID:  message,
			Type:       "message",
			Text:       text,
			ReplyQueue: ac.replyQueue,
			Online:     true,
			VTKey:      t.VTKey,
			XFEKey:     t.XFEKey,
			XFEPass:    t.XFEPass,
		}
	} else {
		// Bot scope does not have file info and history permissions so we need to iterate users
		users, err := ac.r.TeamMembers(team)
		if err != nil {
			logrus.Warnf("Error loading team members - %v\n", err)
			WriteError(w, ErrInternalServer)
			return
		}
		users = append([]domain.User{{Name: "dbot", Token: t.BotToken, ID: t.BotUserID, Status: domain.UserStatusActive}}, users...)
		for i := range users {
			if users[i].Status == domain.UserStatusActive {
				// The first one that can retrieve the info...
				s, err := slack.New(slack.SetToken(users[i].Token))
				if err != nil {
					logrus.Infof("Error creating Slack client for user %s (%s) - %v\n", users[i].ID, users[i].Name, err)
					continue
				}
				info, err := s.FileInfo(file, 0, 0)
				if err != nil {
					logrus.Infof("Error retrieving file info - %v\n", err)
					continue
				}
				workReq = &domain.WorkRequest{
					Type:       "file",
					File:       domain.File{URL: info.File.URLPrivate, Name: info.File.Name, Size: info.File.Size, Token: t.BotToken},
					ReplyQueue: ac.replyQueue,
					Context:    nil,
					Online:     true,
					VTKey:      t.VTKey,
					XFEKey:     t.XFEKey,
					XFEPass:    t.XFEPass,
				}
				break
			}
		}
		// Just retrieve the details for the MD5
		if workReq == nil {
			workReq = &domain.WorkRequest{
				MessageID:  "file-message",
				Type:       "message",
				Text:       text,
				ReplyQueue: ac.replyQueue,
				Online:     true,
				VTKey:      t.VTKey,
				XFEKey:     t.XFEKey,
				XFEPass:    t.XFEPass,
			}
		}
	}
	if workReq == nil {
		logrus.Infof("Unable to find a suitable user with credentials for team %s\n", team)
		WriteError(w, ErrInternalServer)
		return
	}
	err = ac.q.PushWork(workReq)
	if err != nil {
		logrus.Warnf("Error pushing work - %v\n", err)
		WriteError(w, ErrInternalServer)
		return
	}
	workReply, err := ac.q.PopWorkReply(ac.replyQueue, 0)
	json.NewEncoder(w).Encode(workReply)
}

type messageCount struct {
	Count int `json:"count"`
}

// totalMessages we ever saw and handled
func (ac *AppContext) totalMessages(w http.ResponseWriter, r *http.Request) {
	cnt, err := ac.r.TotalMessages()
	if err != nil {
		panic(err)
	}
	json.NewEncoder(w).Encode(messageCount{cnt})
}
