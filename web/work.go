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
	if team == "" || file == "" && (message == "" || channel == "") {
		WriteError(w, ErrBadRequest)
		return
	}
	// Bot scope does not have file info and history permissions so we need to iterate users
	users, err := ac.r.TeamMembers(team)
	if err != nil {
		logrus.Warnf("Error loading team members - %v\n", err)
		WriteError(w, ErrInternalServer)
		return
	}
	var workReq *domain.WorkRequest
	for i := range users {
		if users[i].Status == domain.UserStatusActive {
			// The first one that can retrieve the info...
			s, err := slack.New(slack.SetToken(users[i].Token))
			if err != nil {
				logrus.Infof("Error creating Slack client for user %s (%s) - %v\n", users[i].ID, users[i].Name, err)
				continue
			}
			if file != "" {
				info, err := s.FileInfo(file, 0, 0)
				if err != nil {
					logrus.Infof("Error retrieving file info - %v\n", err)
					continue
				}
				workReq = &domain.WorkRequest{
					Type:       "file",
					File:       domain.File{URL: info.File.URL, Name: info.File.Name, Size: info.File.Size},
					ReplyQueue: ac.replyQueue,
					Context:    nil,
					Online:     true,
				}
				break
			} else {
				resp, err := s.History(channel, message, message, true, false, 1)
				if err != nil {
					logrus.Infof("Error retrieving message history - %v\n", err)
					continue
				}
				if len(resp.Messages) == 0 {
					logrus.Infof("Error retrieving message history - message %s not found on channel %s\n", message, channel)
					WriteError(w, ErrInternalServer)
					return
				}
				workReq = domain.WorkRequestFromMessage(&resp.Messages[0])
				workReq.ReplyQueue = ac.replyQueue
				workReq.Online = true
				break
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
