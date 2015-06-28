package bot

import (
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/goxforce"
	"github.com/demisto/slack"
	"github.com/slavikm/govt"
)

// subscription holds the interest we have for each team
type subscription struct {
	user     *domain.User // the users for the team
	interest *domain.Configuration
	s        *slack.Slack // the slack client
}

type subscriptions []subscription

func (subs subscriptions) FirstSubForChannel(channel string) *subscription {
	for i := range subs {
		if subs[i].interest.IsInterestedIn(channel) {
			return &subs[i]
		}
	}
	return nil
}

func (subs subscriptions) SubForUser(user string) *subscription {
	for i := range subs {
		if subs[i].user.ID == user {
			return &subs[i]
		}
	}
	return nil
}

// Bot iterates on all subscriptions and listens / responds to messages
type Bot struct {
	in              chan slack.Message
	stop            chan bool
	r               repo.Repo
	xfe             *goxforce.Client
	vt              *govt.Client
	mu              sync.RWMutex // Guards the subscriptions
	subscriptions   map[string]subscriptions
	handledMessages map[string]map[string]*time.Time // message map per team of messages we already handled
}

// New returns a new bot
func New(r repo.Repo) (*Bot, error) {
	xfe, err := goxforce.New(goxforce.SetErrorLog(log.New(conf.LogWriter, "XFE:", log.Lshortfile)))
	if err != nil {
		return nil, err
	}
	vt, err := govt.New(govt.SetApikey(conf.Options.VT), govt.SetErrorLog(log.New(os.Stderr, "VT:", log.Lshortfile)))
	if err != nil {
		return nil, err
	}
	return &Bot{
		in:              make(chan slack.Message),
		stop:            make(chan bool, 1),
		r:               r,
		xfe:             xfe,
		vt:              vt,
		subscriptions:   make(map[string]subscriptions),
		handledMessages: make(map[string]map[string]*time.Time),
	}, nil
}

// loadSubscriptions loads all subscriptions per team
func (b *Bot) loadSubscriptions() error {
	teams, err := b.r.Teams()
	if err != nil {
		return err
	}
	for i := range teams {
		var teamSubs subscriptions
		users, err := b.r.TeamMembers(teams[i].ID)
		if err != nil {
			return err
		}
		for j := range users {
			subs, err := b.r.ChannelsAndGroups(users[j].ID)
			if err != nil {
				return err
			}
			if !subs.IsActive() {
				continue
			}
			teamSub := subscription{user: &users[j], interest: subs}
			s, err := slack.New(slack.SetToken(users[j].Token),
				slack.SetErrorLog(log.New(conf.LogWriter, "SLACK:", log.Lshortfile)))
			if err != nil {
				return err
			}
			teamSub.s = s
			teamSubs = append(teamSubs, teamSub)
		}
		b.subscriptions[teams[i].ID] = teamSubs
	}
	return nil
}

type context struct {
	Team string
	User string
}

func (b *Bot) startWS() error {
	for k, v := range b.subscriptions {
		logrus.Infof("Starting subscription for team - %s\n", k)
		for i := range v {
			logrus.Infof("Starting WS for user - %s (%s)\n", v[i].user.ID, v[i].user.Name)
			_, err := v[i].s.RTMStart("slack.demisto.com", b.in, &context{Team: k, User: v[i].user.ID})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Start the monitoring process - will start a separate Go routine
func (b *Bot) Start() error {
	err := b.loadSubscriptions()
	if err != nil {
		return err
	}
	err = b.startWS()
	if err != nil {
		return err
	}
	ipReg := regexp.MustCompile("\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b")
	md5Reg := regexp.MustCompile("\\b[a-fA-F\\d]{32}\\b")
	go func() {
		// Clean messages every 10 minutes
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-b.stop:
				break
			case <-ticker.C:
				// Time to clean the messages
				for _, v := range b.handledMessages {
					for k, t := range v {
						if time.Since(*t) > 5*time.Minute {
							delete(v, k)
						}
					}
				}
			case msg := <-b.in:
				logrus.Debugf("Handling message - %v\n", msg)
				switch msg.Type {
				case "message":
					logrus.Debugf("%s\n", msg.Text)
					if msg.Subtype == "bot_message" {
						continue
					}
					if b.alreadyHandled(&msg) {
						continue
					}
					if !b.isThereInterestIn(&msg) {
						logrus.Debugf("No one is interested in the channel %s\n", msg.Channel)
						continue
					}
					if strings.Contains(msg.Text, "<http") {
						go b.handleURL(msg)
					}
					if ip := ipReg.FindString(msg.Text); ip != "" {
						go b.handleIP(msg, ip)
					}
					if md5 := md5Reg.FindString(msg.Text); md5 != "" {
						go b.handleMD5(msg, md5)
					}
				}
			}
		}
	}()
	return nil
}

// Stop the monitoring process
func (b *Bot) Stop() {
	b.stop <- true
}

// SubscriptionChanged updates the subscriptions if a user changes them
func (b *Bot) SubscriptionChanged(user *domain.User, configuration *domain.Configuration) {
	b.mu.Lock()
	subs := b.subscriptions[user.Team]
	sub := subs.SubForUser(user.ID)
	if sub == nil {
		newSub := subscription{user: user, interest: configuration}
		var err error
		newSub.s, err = slack.New(slack.SetToken(user.Token),
			slack.SetErrorLog(log.New(conf.LogWriter, "SLACK:", log.Lshortfile)))
		if err != nil {
			logrus.WithField("error", err).Errorf("Error creating slack client for %s (%s)\n", user.ID, user.Name)
		}
		b.subscriptions[user.Team] = append(subs, newSub)
		_, err = newSub.s.RTMStart("slack.demisto.com", b.in, &context{Team: user.Team, User: user.ID})
		if err != nil {
			logrus.WithField("error", err).Errorf("Error starting RTM for %s (%s)\n", user.ID, user.Name)
		}
	} else {
		// We already have subscription - if the new one is still active, no need to touch WS
		if configuration.IsActive() {
			sub.interest = configuration
		} else {
			// Since we are not registered to anything, need to close the WS
			// TODO - sub.s.RTMStop()
			// TODO - delete the user
			sub.interest = configuration
		}
	}
	defer b.mu.Unlock()
}
