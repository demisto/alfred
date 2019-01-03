package web

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/slack"
)

func (ac *AppContext) events(w http.ResponseWriter, r *http.Request) {
	msg := getRequestBody(r).(*slack.Response)
	ac.b.HandleMessage(*msg)
	w.Write([]byte{'\n'})
}

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

	logrus.Debugf("Working on request for team - %s, file - %s, message - %s, channel - %s, text - %s.", team, file, message, channel, text)

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
			Context:    &domain.Context{},
		}
	} else {
		// Bot scope does not have file info and history permissions so we need to iterate users
		users, err := ac.r.TeamMembers(team)
		if err != nil {
			logrus.Errorf("Error loading team members - %v\n", err)
			WriteError(w, ErrCouldNotFindTeam)
			return
		}
		users = append([]domain.User{{Name: "dbot", Token: t.BotToken, ID: t.BotUserID, Status: domain.UserStatusActive}}, users...)
		for i := range users {
			if users[i].Status == domain.UserStatusActive {
				// The first one that can retrieve the info...
				s := &slack.Client{Token: users[i].Token}
				info, err := s.Do("GET", "files.info", map[string]string{"file": file, "count": "0", "page": "0"})
				if err != nil {
					logrus.Infof("Error retrieving file info - %v\n", err)
					continue
				}
				workReq = &domain.WorkRequest{
					Type:       "file",
					File:       domain.File{URL: info.S("file.url_private"), Name: info.S("file.name"), Size: info.I("file.size"), Token: t.BotToken},
					ReplyQueue: ac.replyQueue,
					Context:    &domain.Context{},
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
				Context:    &domain.Context{},
				ReplyQueue: ac.replyQueue,
				Online:     true,
				VTKey:      t.VTKey,
				XFEKey:     t.XFEKey,
				XFEPass:    t.XFEPass,
			}
		}
	}
	if workReq == nil {
		logrus.Errorf("Unable to find a suitable user with credentials for team %s\n", team)
		WriteError(w, ErrInternalServer)
		return
	}
	err = ac.q.PushWork(workReq)
	if err != nil {
		logrus.WithError(err).Error("Error pushing work")
		WriteError(w, ErrInternalServer)
		return
	}
	workReply, err := ac.q.PopWebReply(ac.replyQueue, 0)
	json.NewEncoder(w).Encode(workReply)
}

type messageCount struct {
	Count int `json:"count"`
}

// totalMessages we ever saw and handled
func (ac *AppContext) totalMessages(w http.ResponseWriter, r *http.Request) {
	cnt, err := ac.r.TotalMessages()
	if err != nil {
		logrus.WithError(err).Error("Failed getting total messages")
		WriteError(w, ErrInternalServer)
		return
	}
	json.NewEncoder(w).Encode(messageCount{cnt})
}
