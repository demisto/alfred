package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/util"
	"github.com/demisto/slack"
	"github.com/gorilla/context"
	"github.com/wayn3h0/go-uuid"
	"golang.org/x/oauth2"
)

type simpleUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	RealName string `json:"real_name"`
	TeamName string `json:"team_name"`
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
	// First - check that you are not from a banned country
	if isBanned(r.RemoteAddr) {
		http.Redirect(w, r, "/banned", http.StatusFound)
		return
	}
	// Now, generate a random state
	uuid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	conf := &oauth2.Config{
		ClientID:     conf.Options.Slack.ClientID,
		ClientSecret: conf.Options.Slack.ClientSecret,
		Scopes: []string{
			"bot", "files:read", "channels:write", "team:read", "users:read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  slackOAuthEndpoint,
			TokenURL: slackOAuthExchange,
		},
	}
	// Store state
	ac.r.SetOAuthState(&domain.OAuthState{State: uuid.String(), Timestamp: time.Now()})
	url := conf.AuthCodeURL(uuid.String())
	logrus.Debugf("Redirecting to URL - %s", url)
	http.Redirect(w, r, url, http.StatusFound)
}

func sendThanks(team *domain.Team, user *domain.User) {
	s, err := slack.New(
		slack.SetToken(team.BotToken),
		slack.SetErrorLog(log.New(conf.LogWriter, "", log.Lshortfile)),
	)
	if err != nil {
		logrus.Warnf("Unable to create client for first message - %v", err)
		return
	}
	imList, err := s.IMList()
	if err != nil {
		logrus.Warnf("Unable to retrieve im list for first message - %v", err)
		return
	}
	var ch string
	for i := range imList.IMs {
		if user.ExternalID == imList.IMs[i].User {
			ch = imList.IMs[i].ID
			break
		}
	}
	if ch == "" {
		logrus.Warn("Unable to user channel")
		return
	}
	postMessage := &slack.PostMessageRequest{
		Channel: ch,
		AsUser:  true,
		Text: fmt.Sprintf(`Hi %s, thanks for inviting me to this team.
If you want me to monitor conversations, please add me to the relevant channels and groups.
Here are the commands I understand:
config: list the current channels I'm listening on
join all/#channel1,#channel2...: I will join all/specified public channels and start monitoring them.
verbose on/off #channel1,#channel2... - turn on verbose mode on the specified channels
verbose mode is usually used by security professionals. When in verbose mode, dbot will display reputation details about any URL, IP or file including clean ones.`, user.Name),
	}
	_, err = s.PostMessage(postMessage, false)
	if err != nil {
		logrus.Warnf("Error posting welcome message - %v", err)
	}
}

func (ac *AppContext) loginOAuth(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	code := r.FormValue("code")
	errStr := r.FormValue("error")
	if errStr != "" {
		WriteError(w, &Error{"oauth_err", 401, "Slack OAuth Error", errStr})
		logrus.Warnf("Got an error from Slack - %s", errStr)
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
		logrus.Warnf("Got an error exchanging code for token - %v", err)
		return
	}
	logrus.Debugln("OAuth successful, creating Slack client")
	s, err := slack.New(
		slack.SetToken(token.AccessToken),
		slack.SetErrorLog(log.New(conf.LogWriter, "", log.Lshortfile)),
	)
	if err != nil {
		panic(err)
	}
	if logrus.GetLevel() == logrus.DebugLevel {
		slack.SetTraceLog(log.New(conf.LogWriter, "", log.Lshortfile))(s)
	}
	logrus.Debugln("Slack client created")
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
	logrus.Debugln("Got all details about myself from Slack")
	ourTeam, err := ac.r.TeamByExternalID(team.Team.ID)
	if err != nil {
		logrus.Debugf("Got a new team registered - %s", team.Team.Name)
		teamID, err := uuid.NewRandom()
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
			BotUserID:   token.Bot.BotUserID,
			BotToken:    token.Bot.BotAccessToken,
		}
	} else {
		logrus.Debugf("Got an existing team - %s", team.Team.Name)
		ourTeam.Name, ourTeam.EmailDomain, ourTeam.Domain, ourTeam.Plan, ourTeam.BotUserID, ourTeam.BotToken =
			team.Team.Name, team.Team.EmailDomain, team.Team.Domain, team.Team.Plan, token.Bot.BotUserID, token.Bot.BotAccessToken
	}
	newUser := false
	logrus.Debugln("Finding the user...")
	ourUser, err := ac.r.UserByExternalID(user.User.ID)
	if err != nil {
		logrus.Infof("Got a new user registered - %s", user.User.Name)
		userID, err := uuid.NewRandom()
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
		ourUser.Name, ourUser.RealName, ourUser.Email, ourUser.Token, ourUser.Status =
			user.User.Name, user.User.RealName, user.User.Profile.Email, token.AccessToken, domain.UserStatusActive
	}
	logrus.Debugln("Saving to the DB...")
	err = ac.r.SetTeamAndUser(ourTeam, ourUser)
	if err != nil {
		panic(err)
	}
	logrus.Infof("User %v logged in\n", ourUser.Name)
	if newUser {
		newConf := &domain.Configuration{All: true}
		err = ac.r.SetChannelsAndGroups(ourTeam.ID, newConf)
		if err != nil {
			// If we got here, allow empty configuration
			logrus.Warnf("Unable to store initial configuration for user %s - %v\n", ourUser.ID, err)
		}
	}
	// Send the first DM message to the user
	sendThanks(ourTeam, ourUser)
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
	t, err := ac.r.Team(u.Team)
	if err != nil {
		panic(err)
	}
	externalUser := simpleUser{u.Name, u.Email, u.RealName, t.Name}
	json.NewEncoder(w).Encode(externalUser)
}
