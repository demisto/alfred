package web

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/slack"
)

func (ac *AppContext) work(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("u")
	file := r.FormValue("f")
	message := r.FormValue("m")
	channel := r.FormValue("c")
	if user == "" || file == "" && (message == "" || channel == "") {
		WriteError(w, ErrBadRequest)
		return
	}
	u, err := ac.r.User(user)
	if err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	s, err := slack.New(slack.SetToken(u.Token))
	if err != nil {
		WriteError(w, ErrInternalServer)
		return
	}
	var workReq *domain.WorkRequest
	if file != "" {
		info, err := s.FileInfo(file, 0, 0)
		if err != nil {
			WriteError(w, ErrInternalServer)
			return
		}
		workReq = &domain.WorkRequest{
			Type:       "file",
			File:       domain.File{URL: info.File.URL, Name: info.File.Name, Size: info.File.Size},
			ReplyQueue: ac.replyQueue,
			Context:    nil,
			Online:     true,
		}
	} else {
		resp, err := s.History(channel, message, message, true, 1)
		if err != nil {
			WriteError(w, ErrInternalServer)
			return
		}
		if len(resp.Messages) == 0 {
			WriteError(w, ErrInternalServer)
			return
		}
		workReq = domain.WorkRequestFromMessage(&resp.Messages[0])
		workReq.ReplyQueue = ac.replyQueue
		workReq.Online = true
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
