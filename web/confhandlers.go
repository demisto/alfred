package web

import (
	"encoding/json"
	"net/http"

	"github.com/demisto/alfred/domain"
	"github.com/demisto/server/util"
	"github.com/demisto/slack"
	"github.com/gorilla/context"
)

type idName struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type infoResponse struct {
	Channels []idName `json:"channels"`
	Groups   []idName `json:"groups"`
	IM       bool     `json:"im"`
}

func (ac *AppContext) info(w http.ResponseWriter, r *http.Request) {
	u := context.Get(r, "user").(*domain.User)
	var res infoResponse
	// First, get the current selection (if at all)
	savedChannels, err := ac.r.ChannelsAndGroups(u.ID)
	if err != nil {
		panic(err)
	}
	s, err := slack.New(slack.SetToken(u.Token))
	if err != nil {
		panic(err)
	}
	ch, err := s.ChannelList(true)
	if err != nil {
		panic(err)
	}
	for i := range ch.Channels {
		selected := util.In(savedChannels.Channels, ch.Channels[i].ID)
		res.Channels = append(res.Channels, idName{ID: ch.Channels[i].ID, Name: ch.Channels[i].Name, Selected: selected})
	}
	gr, err := s.GroupList(true)
	if err != nil {
		panic(err)
	}
	for i := range gr.Groups {
		selected := util.In(savedChannels.Groups, gr.Groups[i].ID)
		res.Groups = append(res.Groups, idName{ID: gr.Groups[i].ID, Name: gr.Groups[i].Name, Selected: selected})
	}
	res.IM = savedChannels.IM
	json.NewEncoder(w).Encode(res)
}

func (ac *AppContext) save(w http.ResponseWriter, r *http.Request) {
	req := context.Get(r, "body").(*domain.Configuration)
	u := context.Get(r, "user").(*domain.User)
	err := ac.r.SetChannelsAndGroups(u.ID, req)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("\n"))
}
