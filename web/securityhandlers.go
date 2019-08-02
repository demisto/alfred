package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/slack"
	"github.com/demisto/alfred/util"
	"github.com/wayn3h0/go-uuid"
	"golang.org/x/oauth2"
)

type simpleUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	RealName string `json:"real_name"`
	TeamName string `json:"team_name"`
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
	uid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	con := &oauth2.Config{
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
	ac.r.SetOAuthState(&domain.OAuthState{State: uid.String(), Timestamp: time.Now()})
	url := con.AuthCodeURL(uid.String())
	logrus.Debugf("Redirecting to URL - %s", url)
	http.Redirect(w, r, url, http.StatusFound)
}

func sendThanks(team *domain.Team, user *domain.User) {
	s := &slack.Client{Token: team.BotToken}
	channel, err := s.Do("POST", "im.open", map[string]interface{}{
		"user": user.ExternalID,
	})
	if err != nil {
		logrus.WithError(err).Warnf("unable to open im for first message for user [%s (%s)], team [%s (%s)]", user.Name, user.ExternalID, team.Name, team.ExternalID)
		return
	}
	_, err = s.Do("POST", "chat.postMessage", map[string]interface{}{
		"channel": channel.S("channel.id"),
		"as_user": true,
		"text": fmt.Sprintf(`Hi %s, thanks for inviting me to this team.
If you want me to monitor conversations, please add me to the relevant channels and groups.
`+conf.DefaultHelpMessage, user.Name),
	})
	if err != nil {
		logrus.Warnf("Error posting welcome message - %v", err)
	}
	return
}

func (ac *AppContext) loginOAuth(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	code := r.FormValue("code")
	errStr := r.FormValue("error")
	if errStr != "" {
		WriteError(w, &Error{"oauth_err", 401, "Slack OAuth Error", errStr})
		logrus.Warnf("got an error from Slack - %s", errStr)
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
	s := &slack.Client{}
	oauthAccess, err := s.Do("GET", "oauth.access", map[string]string{
		"client_id":     conf.Options.Slack.ClientID,
		"client_secret": conf.Options.Slack.ClientSecret,
		"code":          code,
	})
	if err != nil {
		WriteError(w, &Error{"oauth_err", 401, "Slack OAuth Error", err.Error()})
		logrus.WithError(err).Warnf("got an error exchanging code for token")
		return
	}
	logrus.Debugln("OAuth successful, creating Slack client")
	s.Token = oauthAccess.S("access_token")
	// Get our own user id
	test, err := s.Do("POST", "auth.test", nil)
	if err != nil {
		panic(err)
	}
	team, err := s.Do("GET", "team.info", nil)
	if err != nil {
		panic(err)
	}
	user, err := s.Do("GET", "users.info", map[string]string{"user": test.S("user_id")})
	if err != nil {
		panic(err)
	}
	logrus.Debugln("Got all details about myself from Slack")
	ourTeam, err := ac.r.TeamByExternalID(team.S("team.id"))
	if err != nil {
		logrus.Debugf("Got a new team registered - %s", team.S("team.name"))
		teamID, err := uuid.NewRandom()
		if err != nil {
			panic(err)
		}
		ourTeam = &domain.Team{
			ID:          "T" + teamID.String(),
			Name:        team.S("team.name"),
			EmailDomain: team.S("team.email_domain"),
			Domain:      team.S("team.domain"),
			Plan:        team.S("team.enterprise_id") + "," + team.S("team.enterprise_name"),
			ExternalID:  team.S("team.id"),
			Created:     time.Now(),
			BotUserID:   oauthAccess.S("bot.bot_user_id"),
			BotToken:    oauthAccess.S("bot.bot_access_token"),
			Status:      domain.UserStatusActive,
		}
	} else {
		logrus.Debugf("Got an existing team - %s", team.S("team.name"))
		ourTeam.Name, ourTeam.EmailDomain, ourTeam.Domain, ourTeam.Plan, ourTeam.BotUserID, ourTeam.BotToken, ourTeam.Status =
			team.S("team.name"), team.S("team.email_domain"), team.S("team.domain"), team.S("team.enterprise_id")+","+team.S("team.enterprise_name"),
			oauthAccess.S("bot.bot_user_id"), oauthAccess.S("bot.bot_access_token"), domain.UserStatusActive
	}
	logrus.Debugln("Finding the user...")
	ourUser, err := ac.r.UserByExternalID(user.S("user.id"))
	if err != nil {
		logrus.Infof("Got a new user registered - %s", user.S("user.name"))
		userID, err := uuid.NewRandom()
		if err != nil {
			panic(err)
		}
		ourUser = &domain.User{
			ID:                "U" + userID.String(),
			Team:              ourTeam.ID,
			Name:              user.S("user.name"),
			Type:              domain.UserTypeSlack,
			Status:            domain.UserStatusActive,
			RealName:          user.S("user.real_name"),
			Email:             user.S("user.profile.email"),
			IsBot:             user.B("user.is_bot"),
			IsAdmin:           user.B("user.is_admin"),
			IsOwner:           user.B("user.is_owner"),
			IsPrimaryOwner:    user.B("user.is_primary_owner"),
			IsRestricted:      user.B("user.is_restricted"),
			IsUltraRestricted: user.B("user.is_ultra_restricted"),
			ExternalID:        user.S("user.id"),
			Token:             s.Token,
			Created:           time.Now(),
		}
	} else {
		ourUser.Name, ourUser.RealName, ourUser.Email, ourUser.Token, ourUser.Status =
			user.S("user.name"), user.S("user.real_name"), user.S("user.profile.email"), s.Token, domain.UserStatusActive
	}
	logrus.Debugln("Saving to the DB...")
	err = ac.r.SetTeamAndUser(ourTeam, ourUser)
	if err != nil {
		panic(err)
	}
	if err = ac.q.PushConf(ourTeam.ExternalID); err != nil {
		logrus.WithError(err).Warnf("Unable to push configuration reload for team [%s]", ourTeam.ExternalID)
	}
	logrus.Infof("User %v logged in\n", ourUser.Name)
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
	u := getRequestUser(r)
	if u == nil {
		WriteError(w, ErrAuth)
		return
	}
	t, err := ac.r.Team(u.Team)
	if err != nil {
		panic(err)
	}
	externalUser := simpleUser{u.Name, u.Email, u.RealName, t.Name}
	json.NewEncoder(w).Encode(externalUser)
}
