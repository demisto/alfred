package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/util"
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
	Regexp   string   `json:"regexp"`
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
		if ch.Channels[i].IsMember {
			selected := util.In(savedChannels.Channels, ch.Channels[i].ID)
			res.Channels = append(res.Channels, idName{ID: ch.Channels[i].ID, Name: ch.Channels[i].Name, Selected: selected})
		}
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
	res.Regexp = savedChannels.Regexp
	json.NewEncoder(w).Encode(res)
}

type regexpMatch struct {
	Regexp string `json:"regexp"`
}

// match the regular expression to all channels / groups from
func (ac *AppContext) match(w http.ResponseWriter, r *http.Request) {
	req := context.Get(r, "body").(*regexpMatch)
	u := context.Get(r, "user").(*domain.User)
	var res []string
	if req.Regexp != "" {
		// First, let's compile the regexp
		re, err := regexp.Compile(req.Regexp)
		if err != nil {
			WriteError(w, &Error{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: fmt.Sprintf("Error parsing regexp - %v", err)})
			return
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
			if ch.Channels[i].IsMember {
				if re.MatchString(ch.Channels[i].Name) {
					res = append(res, ch.Channels[i].Name)
				}
			}
		}
		gr, err := s.GroupList(true)
		if err != nil {
			panic(err)
		}
		for i := range gr.Groups {
			if re.MatchString(gr.Groups[i].Name) {
				res = append(res, gr.Groups[i].Name)
			}
		}
	}
	json.NewEncoder(w).Encode(res)
}

func (ac *AppContext) save(w http.ResponseWriter, r *http.Request) {
	req := context.Get(r, "body").(*domain.Configuration)
	u := context.Get(r, "user").(*domain.User)
	// Before saving, validate that the regexp is valid
	if req.Regexp != "" {
		_, err := regexp.Compile(req.Regexp)
		if err != nil {
			WriteError(w, &Error{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: fmt.Sprintf("Error parsing regexp - %v", err)})
			return
		}
	}
	err := ac.r.SetChannelsAndGroups(u.ID, req)
	if err != nil {
		panic(err)
	}
	ac.b.SubscriptionChanged(u, req)
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("\n"))
}
