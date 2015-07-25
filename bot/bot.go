package bot

import (
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/slack"
)

// subscription holds the interest we have for each team
type subscription struct {
	user     *domain.User // the users for the team
	interest *domain.Configuration
	s        *slack.Slack // the slack client
}

type subscriptions struct {
	subscriptions []subscription
	info          *slack.RTMStartReply
}

func (subs *subscriptions) ChannelName(channel string, subscriber int) string {
	if channel == "" {
		return ""
	}
	// if not found, it might be a new channel or group
	switch channel[0] {
	case 'C':
		for i := range subs.info.Channels {
			if channel == subs.info.Channels[i].ID {
				return subs.info.Channels[i].Name
			}
		}
		info, err := subs.subscriptions[subscriber].s.ChannelInfo(channel)
		if err != nil {
			logrus.WithField("error", err).Warn("Unable to get channel info\n")
			return ""
		}
		subs.info.Channels = append(subs.info.Channels, info.Channel)
		return info.Channel.Name
	case 'G':
		for i := range subs.info.Groups {
			if channel == subs.info.Groups[i].ID {
				return subs.info.Groups[i].Name
			}
		}
		info, err := subs.subscriptions[subscriber].s.GroupInfo(channel)
		if err != nil {
			logrus.WithField("error", err).Warn("Unable to get group info\n")
			return ""
		}
		subs.info.Groups = append(subs.info.Groups, info.Group)
		return info.Group.Name
	}
	return ""
}

func (subs *subscriptions) FirstSubForChannel(channel string) *subscription {
	for i := range subs.subscriptions {
		channelName := subs.ChannelName(channel, i)
		logrus.Debugf("Channel %s (%s)\n", channel, channelName)
		if subs.subscriptions[i].interest.IsInterestedIn(channel, channelName) {
			return &subs.subscriptions[i]
		}
	}
	return nil
}

func (subs subscriptions) SubForUser(user string) *subscription {
	for i := range subs.subscriptions {
		if subs.subscriptions[i].user.ID == user {
			return &subs.subscriptions[i]
		}
	}
	return nil
}

// Bot iterates on all subscriptions and listens / responds to messages
type Bot struct {
	in            chan slack.Message
	stop          chan bool
	r             repo.Repo
	mu            sync.RWMutex // Guards the subscriptions
	subscriptions map[string]*subscriptions
	q             queue.Queue // Message queue for configuration updates
}

// New returns a new bot
func New(r repo.Repo, q queue.Queue) (*Bot, error) {
	return &Bot{
		in:            make(chan slack.Message),
		stop:          make(chan bool, 1),
		r:             r,
		subscriptions: make(map[string]*subscriptions),
		q:             q,
	}, nil
}

// loadSubscriptions loads all subscriptions per team
func (b *Bot) loadSubscriptions() error {
	teams, err := b.r.Teams()
	if err != nil {
		return err
	}
	for i := range teams {
		teamSubs := &subscriptions{}
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
			teamSubs.subscriptions = append(teamSubs.subscriptions, teamSub)
		}
		b.subscriptions[teams[i].ID] = teamSubs
	}
	return nil
}

// Context to push with each message to identify the relevant team and user
type Context struct {
	Team string
	User string
}

func (b *Bot) startWS() error {
	for k, v := range b.subscriptions {
		logrus.Infof("Starting subscription for team - %s\n", k)
		for i := range v.subscriptions {
			logrus.Infof("Starting WS for user - %s (%s)\n", v.subscriptions[i].user.ID, v.subscriptions[i].user.Name)
			info, err := v.subscriptions[i].s.RTMStart("slack.demisto.com", b.in, &Context{Team: k, User: v.subscriptions[i].user.ID})
			if err != nil {
				return err
			}
			// Just save the first one
			if i == 0 {
				v.info = info
			}
		}
	}
	return nil
}

func (b *Bot) stopWS() {
	for k, v := range b.subscriptions {
		logrus.Infof("Stoping subscription for team - %s\n", k)
		for i := range v.subscriptions {
			logrus.Infof("Stoping WS for user - %s (%s)\n", v.subscriptions[i].user.ID, v.subscriptions[i].user.Name)
			err := v.subscriptions[i].s.RTMStop()
			if err != nil {
				logrus.Warnf("Unable to stop subscription for user %s - %v\n", v.subscriptions[i].user.ID, err)
			}
		}
	}
}

var (
	ipReg  = regexp.MustCompile("\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b")
	md5Reg = regexp.MustCompile("\\b[a-fA-F\\d]{32}\\b")
)

func (b *Bot) isThereInterestIn(original *slack.Message) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	data := original.Context.(*Context)
	subs := b.subscriptions[data.Team]
	if subs == nil {
		return false
	}
	if sub := subs.FirstSubForChannel(original.Channel); sub != nil {
		return true
	}
	return false
}

func (b *Bot) handleMessage(msg *slack.Message) {
	logrus.Debugf("Handling message - %+v\n", msg)
	if !b.isThereInterestIn(msg) {
		logrus.Debugf("No one is interested in the channel %s\n", msg.Channel)
		return
	}
	switch msg.Type {
	case "message":
		logrus.Debugf("%s\n", msg.Text)
		if msg.Subtype == "bot_message" {
			return
		}
		// If we need to handle the message, pass it to the queue
		if msg.Subtype == "file_share" ||
			strings.Contains(msg.Text, "<http") ||
			ipReg.MatchString(msg.Text) ||
			md5Reg.MatchString(msg.Text) {
			b.q.PushMessage(msg)
		}
	}
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
	go func() {
		for {
			select {
			case <-b.stop:
				b.stopWS()
				return
			case msg := <-b.in:
				b.handleMessage(&msg)
			}
		}
	}()
	go b.monitorChanges()
	return nil
}

// Stop the monitoring process
func (b *Bot) Stop() {
	b.stop <- true
}

// subscriptionChanged updates the subscriptions if a user changes them
func (b *Bot) subscriptionChanged(user *domain.User, configuration *domain.Configuration) {
	b.mu.Lock()
	defer b.mu.Unlock()
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
		b.subscriptions[user.Team] = &subscriptions{subscriptions: []subscription{newSub}}
		info, err := newSub.s.RTMStart("slack.demisto.com", b.in, &Context{Team: user.Team, User: user.ID})
		if err != nil {
			logrus.WithField("error", err).Errorf("Error starting RTM for %s (%s)\n", user.ID, user.Name)
		} else {
			b.subscriptions[user.Team].info = info
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
}

func (b *Bot) monitorChanges() {
	for {
		user, configuration, err := b.q.PopConf(0)
		if err != nil {
			break
		}
		b.subscriptionChanged(user, configuration)
	}
}
