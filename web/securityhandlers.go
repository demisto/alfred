package web

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/util"
	"github.com/demisto/slack"
	"github.com/gorilla/context"
	"github.com/wayn3h0/go-uuid/random"
	"golang.org/x/oauth2"
)

type simpleUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	RealName string `json:"real_name"`
}

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

const (
	slackOAuthEndpoint = "https://slack.com/oauth/authorize"
	slackOAuthExchange = "https://slack.com/api/oauth.access"
)

func (ac *AppContext) initiateOAuth(w http.ResponseWriter, r *http.Request) {
	// First, generate a random state
	uuid, err := random.New()
	if err != nil {
		panic(err)
	}
	conf := &oauth2.Config{
		ClientID:     conf.Options.Slack.ClientID,
		ClientSecret: conf.Options.Slack.ClientSecret,
		Scopes:       []string{"client"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  slackOAuthEndpoint,
			TokenURL: slackOAuthExchange,
		},
	}
	// Store state
	ac.r.SetOAuthState(&domain.OAuthState{State: uuid.String(), Timestamp: time.Now()})
	url := conf.AuthCodeURL(uuid.String())
	http.Redirect(w, r, url, http.StatusFound)
}

func (ac *AppContext) loginOAuth(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	code := r.FormValue("code")
	errStr := r.FormValue("error")
	if errStr != "" {
		WriteError(w, &Error{"oauth_err", 401, "Slack OAuth Error", errStr})
		return
	}
	if state == "" || code == "" {
		WriteError(w, ErrBadContentRequest)
		return
	}
	savedState, err := ac.r.OAuthState(state)
	if err != nil {
		WriteError(w, ErrBadContentRequest)
		return
	}
	// We allow only 5 min between requests
	if time.Since(savedState.Timestamp) > 5*time.Minute {
		WriteError(w, ErrBadRequest)
	}
	token, err := slack.OAuthAccess(conf.Options.Slack.ClientID,
		conf.Options.Slack.ClientSecret, code, "")
	if err != nil {
		WriteError(w, &Error{"oauth_err", 401, "Slack OAuth Error", err.Error()})
		return
	}
	log.Debugln("OAuth successful, creating Slack client")
	s, err := slack.New(slack.SetToken(token.AccessToken))
	if err != nil {
		panic(err)
	}
	log.Debugln("Slack client created")
	// Get our own user id
	test, err := s.AuthTest()
	if err != nil {
		panic(err)
	}
	team, err := s.TeamInfo()
	if err != nil {
		panic(err)
	}
	user, err := s.UserInfo(test.UserID)
	if err != nil {
		panic(err)
	}
	log.Debugln("Got all details about myself from Slack")
	ourTeam, err := ac.r.TeamByExternalID(team.Team.ID)
	if ourTeam == nil {
		teamID, err := random.New()
		if err != nil {
			panic(err)
		}
		ourTeam = &domain.Team{
			ID:          "T" + teamID.String(),
			Name:        team.Team.Name,
			EmailDomain: team.Team.EmailDomain,
			Domain:      team.Team.Domain,
			Plan:        team.Team.Plan,
			ExternalID:  team.Team.ID,
			Created:     time.Now(),
		}
	} else {
		ourTeam.Name, ourTeam.EmailDomain, ourTeam.Domain, ourTeam.Plan =
			team.Team.Name, team.Team.EmailDomain, team.Team.Domain, team.Team.Plan
	}
	newUser := false
	ourUser, err := ac.r.UserByExternalID(user.User.ID)
	if ourUser == nil {
		userID, err := random.New()
		if err != nil {
			panic(err)
		}
		ourUser = &domain.User{
			ID:                "U" + userID.String(),
			Team:              ourTeam.ID,
			Name:              user.User.Name,
			Type:              domain.UserTypeSlack,
			Status:            domain.UserStatusActive,
			RealName:          user.User.RealName,
			Email:             user.User.Profile.Email,
			IsBot:             user.User.IsBot,
			IsAdmin:           user.User.IsAdmin,
			IsOwner:           user.User.IsOwner,
			IsPrimaryOwner:    user.User.IsPrimaryOwner,
			IsRestricted:      user.User.IsRestricted,
			IsUltraRestricted: user.User.IsUltraRestricted,
			ExternalID:        user.User.ID,
			Token:             token.AccessToken,
			Created:           time.Now(),
		}
		newUser = true
	} else {
		ourUser.Name, ourUser.RealName, ourUser.Email, ourUser.Token =
			user.User.Name, user.User.RealName, user.User.Profile.Email, token.AccessToken
	}
	err = ac.r.SetTeamAndUser(ourTeam, ourUser)
	if err != nil {
		panic(err)
	}
	log.Infof("User %v logged in\n", ourUser.Name)
	if newUser {
		newConf := &domain.Configuration{All: true}
		err = ac.r.SetChannelsAndGroups(ourUser.ID, newConf)
		if err != nil {
			// If we got here, allow empty configuration
			log.Warnf("Unable to store initial configuration for user %s - %v\n", ourUser.ID, err)
		}
	}
	sess := session{ourUser.Name, ourUser.ID, time.Now()}
	secure := conf.Options.SSL.Key != ""
	val, _ := util.EncryptJSON(&sess, conf.Options.Security.SessionKey)
	// Set the cookie for the user
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: val, Path: "/", Expires: time.Now().Add(time.Duration(conf.Options.Security.Timeout) * time.Minute), MaxAge: conf.Options.Security.Timeout * 60, Secure: secure, HttpOnly: true})
	http.Redirect(w, r, "/conf", http.StatusFound)
}

func (ac *AppContext) logout(w http.ResponseWriter, r *http.Request) {
	secure := conf.Options.SSL.Key != ""
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", Expires: time.Now(), MaxAge: -1, Secure: secure, HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("\n"))
}

func (ac *AppContext) currUser(w http.ResponseWriter, r *http.Request) {
	u := context.Get(r, "user").(*domain.User)
	externalUser := simpleUser{u.Name, u.Email, u.RealName}
	json.NewEncoder(w).Encode(externalUser)
}
