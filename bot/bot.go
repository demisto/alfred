package bot

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/slack"
)

const (
	maxUsersPerBot = 1000
)

// subscription holds the interest we have for each team
type subscription struct {
	user     *domain.User // the users for the team
	interest *domain.Configuration
	s        *slack.Slack // the slack client
	started  bool         // did we start subscription for this guy
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
	in            chan *slack.Message
	stop          chan bool
	r             repo.Repo
	mu            sync.RWMutex // Guards the subscriptions
	subscriptions map[string]*subscriptions
	q             queue.Queue // Message queue for configuration updates
	replyQueue    string
	smu           sync.Mutex // Guards the statistics
	stats         map[string]*domain.Statistics
}

// New returns a new bot
func New(r repo.Repo, q queue.Queue) (*Bot, error) {
	host, err := queue.ReplyQueueName()
	if err != nil {
		return nil, err
	}
	return &Bot{
		in:            make(chan *slack.Message),
		stop:          make(chan bool, 1),
		r:             r,
		subscriptions: make(map[string]*subscriptions),
		q:             q,
		replyQueue:    host,
		stats:         make(map[string]*domain.Statistics),
	}, nil
}

// loadSubscriptions loads all subscriptions per team
func (b *Bot) loadSubscriptions() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	cnt := 0
	for _, v := range b.subscriptions {
		cnt += len(v.subscriptions)
	}
	// Fully loaded, no room for others
	if cnt >= maxUsersPerBot {
		return nil
	}
	cnt = 0
	// Everything must run in separate transactions as it is written here
	openUsers, err := b.r.OpenUsers()
	if err != nil {
		return err
	}
	for i := range openUsers {
		locked, err := b.r.LockUser(&openUsers[i])
		logrus.Debugf("Trying to lock user %s\n", openUsers[i].User)
		if err != nil || !locked {
			logrus.Debugf("Unable to lock user %v\n", err)
			continue
		}
		u, err := b.r.User(openUsers[i].User)
		if err != nil {
			logrus.Warnf("Error loading user %s - %v\n", openUsers[i].User, err)
			continue
		}
		teamSubs := b.subscriptions[u.Team]
		if teamSubs == nil {
			teamSubs = &subscriptions{}
			b.subscriptions[u.Team] = teamSubs
		}
		// Just precaution, if we already have the user just skip this
		// This is required for bolt repo implementation
		if teamSubs.SubForUser(u.ID) != nil {
			continue
		}
		subs, err := b.r.ChannelsAndGroups(u.ID)
		if err != nil {
			logrus.Warnf("Error loading user configuration - %v\n", err)
			continue
		}
		s, err := slack.New(slack.SetToken(u.Token),
			slack.SetErrorLog(log.New(conf.LogWriter, "SLACK:", log.Lshortfile)))
		if err != nil {
			logrus.Warnf("Error opening Slack for user %s (%s) - %v\n", u.ID, u.Name, err)
			continue
		}
		teamSub := subscription{user: u, interest: subs, s: s}
		teamSubs.subscriptions = append(teamSubs.subscriptions, teamSub)
		cnt++
		if cnt >= maxUsersPerBot {
			break
		}
	}
	return nil
}

func (b *Bot) startWS() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range b.subscriptions {
		for i := range v.subscriptions {
			if !v.subscriptions[i].started && v.subscriptions[i].interest.IsActive() {
				logrus.Infof("Starting WS for team %s, user - %s (%s)\n", k, v.subscriptions[i].user.ID, v.subscriptions[i].user.Name)
				info, err := v.subscriptions[i].s.RTMStart("dbot.demisto.com", b.in, &domain.Context{Team: k, User: v.subscriptions[i].user.ID})
				if err != nil {
					logrus.Warnf("Unable to start WS for user %s (%s) - %v\n", v.subscriptions[i].user.ID, v.subscriptions[i].user.Name, err)
					continue
				}
				v.info = info
				v.subscriptions[i].started = true
			}
		}
	}
	return nil
}

func (b *Bot) stopWS() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range b.subscriptions {
		logrus.Infof("Stoping subscription for team - %s\n", k)
		for i := range v.subscriptions {
			logrus.Infof("Stoping WS for user - %s (%s)\n", v.subscriptions[i].user.ID, v.subscriptions[i].user.Name)
			err := v.subscriptions[i].s.RTMStop()
			if err != nil {
				logrus.Warnf("Unable to stop subscription for user %s - %v\n", v.subscriptions[i].user.ID, err)
			}
			v.subscriptions[i].started = false
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
	data := original.Context.(*domain.Context)
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
	if msg == nil {
		return
	}
	switch msg.Type {
	case "message":
		if msg.Subtype == "bot_message" {
			return
		}
		if !b.isThereInterestIn(msg) {
			logrus.Debugf("No one is interested in the channel %s\n", msg.Channel)
			return
		}
		push := false
		switch msg.Subtype {
		case "":
			push = strings.Contains(msg.Text, "<http") || ipReg.MatchString(msg.Text) || md5Reg.MatchString(msg.Text)
		// case "message_changed":
		// 	push = strings.Contains(msg.Message.Text, "<http") || ipReg.MatchString(msg.Message.Text) || md5Reg.MatchString(msg.Message.Text)
		case "file_share":
			push = true
		case "file_comment":
			push = !strings.Contains(msg.Comment.Comment, conf.Options.ExternalAddress) && (strings.Contains(msg.Comment.Comment, "<http") || ipReg.MatchString(msg.Comment.Comment) || md5Reg.MatchString(msg.Comment.Comment))
		case "file_mention":
			push = true
		}
		ctx, err := GetContext(msg.Context)
		if err != nil {
			logrus.Warnf("Unable to get context from message - %+v\n", msg)
			return
		}
		// If we need to handle the message, pass it to the queue
		if push {
			workReq := domain.WorkRequestFromMessage(msg)
			ctx.OriginalUser, ctx.Channel, ctx.Type = msg.User, msg.Channel, msg.Type
			workReq.ReplyQueue, workReq.Context = b.replyQueue, ctx
			b.q.PushMessage(workReq)
		} else {
			b.smu.Lock()
			defer b.smu.Unlock()
			stats := b.stats[ctx.Team]
			if stats == nil {
				stats = &domain.Statistics{Team: ctx.Team}
				b.stats[ctx.Team] = stats
			}
			stats.Messages++
		}
	}
}

func (b *Bot) storeStatistics() {
	b.smu.Lock()
	defer b.smu.Unlock()
	for _, v := range b.stats {
		err := b.r.UpdateStatistics(v)
		if err == nil {
			v.Reset()
		} else {
			logrus.Warnf("Unable to store statistics - %v\n", err)
			return
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
	go b.monitorChanges()
	go b.monitorReplies()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-b.stop:
			b.stopWS()
			return nil
		case msg := <-b.in:
			// TODO - error handling - something wrong with channel closing in case of error
			if msg == nil || msg.Type == "error" {
				if msg == nil {
					logrus.Warnf("Message channel closed")
				} else {
					logrus.Warnf("Got error message from channel %+v\n", msg)
				}
				// TODO - should we restart the WS for this user?
				continue
			}
			b.handleMessage(msg)
		case <-ticker.C:
			err := b.r.BotHeartbeat()
			if err != nil {
				logrus.Errorf("Unable to update heartbeat - %v\n", err)
			}
			err = b.loadSubscriptions()
			if err != nil {
				logrus.Errorf("Unable to load subscriptions - %v\n", err)
			}
			err = b.startWS()
			if err != nil {
				logrus.Errorf("Error starting WS - %v\n", err)
			}
			b.storeStatistics()
		}
	}
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
	if subs == nil {
		logrus.Debugf("Subscription for team not found: %+v\n", user)
		return
	}
	sub := subs.SubForUser(user.ID)
	if sub != nil {
		// We already have subscription - if the new one is still active, no need to touch WS
		sub.interest = configuration
		if !configuration.IsActive() {
			if sub.started {
				sub.s.RTMStop()
				sub.started = false
			}
		} else if !sub.started {
			logrus.Infof("Starting WS for team %s, user - %s (%s)\n", user.Team, user.ID, user.Name)
			info, err := sub.s.RTMStart("dbot.demisto.com", b.in, &domain.Context{Team: user.Team, User: user.ID})
			if err != nil {
				logrus.Warnf("Unable to start WS for user %s (%s) - %v\n", user.ID, user.Name, err)
				return
			}
			subs.info = info
			sub.started = true
		}
	}
}

func (b *Bot) monitorChanges() {
	for {
		user, configuration, err := b.q.PopConf(0)
		if err != nil || user == nil || configuration == nil {
			logrus.Infof("Quiting monitoring changes - %v\n", err)
			break
		}
		logrus.Debugf("Configuration change received: %+v, %+v\n", user, configuration)
		b.subscriptionChanged(user, configuration)
	}
}

func (b *Bot) monitorReplies() {
	for {
		reply, err := b.q.PopWorkReply(b.replyQueue, 0)
		if err != nil || reply == nil {
			logrus.Infof("Quiting monitoring replies - %v\n", err)
			break
		}
		b.handleReply(reply)
	}
}
