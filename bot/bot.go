package bot

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/slack"
	"github.com/demisto/alfred/util"
)

// subscription holds the interest we have for each team
type subscription struct {
	team          *domain.Team          // the team we are subscribed to
	configuration *domain.Configuration // The configuration of channels, mainly for verbose
	s             *slack.Client         // the slack client on the bot token
	started       bool                  // did we start subscription for this guy
	ts            time.Time             // When did we start the WS
}

// Bot iterates on all subscriptions and listens / responds to messages
type Bot struct {
	stop          chan bool
	r             *repo.MySQL
	mu            sync.RWMutex // Guards the subscriptions
	subscriptions map[string]*subscription
	q             queue.Queue // Message queue for configuration updates
	replyQueue    string
	smu           sync.Mutex // Guards the statistics
	stats         map[string]*domain.Statistics
	firstMessages map[string]bool
}

// New returns a new bot
func New(r *repo.MySQL, q queue.Queue) (*Bot, error) {
	host, err := queue.ReplyQueueName()
	if err != nil {
		return nil, err
	}
	return &Bot{
		stop:          make(chan bool, 1),
		r:             r,
		subscriptions: make(map[string]*subscription),
		q:             q,
		replyQueue:    host,
		stats:         make(map[string]*domain.Statistics),
		firstMessages: make(map[string]bool),
	}, nil
}

// loadSubscriptions loads teams and configurations
func (b *Bot) loadSubscriptions() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	teams, err := b.r.Teams()
	if err != nil {
		return err
	}
	for i := range teams {
		teamSub := &subscription{team: &teams[i]}
		teamSub.configuration, err = b.r.ChannelsAndGroups(teams[i].ID)
		if err != nil {
			logrus.Warnf("Error loading team configuration - %v\n", err)
			continue
		}
		teamSub.s = &slack.Client{Token: teams[i].BotToken}
		b.subscriptions[teams[i].ID] = teamSub
	}
	return nil
}

func (b *Bot) loadSubscription(team string) (*subscription, error) {
	t, err := b.r.TeamByExternalID(team)
	if err != nil {
		return nil, err
	}
	teamSub := &subscription{team: t}
	teamSub.configuration, err = b.r.ChannelsAndGroups(t.ID)
	if err != nil {
		return nil, err
	}
	teamSub.s = &slack.Client{Token: t.BotToken}
	return teamSub, nil
}

var (
	ipReg     = regexp.MustCompile("\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b")
	md5Reg    = regexp.MustCompile("\\b[a-fA-F\\d]{32}\\b")
	sha1Reg   = regexp.MustCompile("\\b[a-fA-F\\d]{40}\\b")
	sha256Reg = regexp.MustCompile("\\b[a-fA-F\\d]{64}\\b")
)

func (b *Bot) HandleMessage(msg slack.Response) {
	if msg == nil {
		return
	}
	team := msg.S("team_id")
	if team == "" {
		logrus.Warnf("got empty team in message %s", util.ToJSONString(msg))
		return
	}
	sub := b.subscriptions[team]
	if sub == nil {
		var err error
		sub, err = b.loadSubscription(team)
		if err != nil {
			logrus.WithError(err).Warnf("Error loading team configuration for new team - %v", team)
			return
		}
		b.mu.Lock()
		b.subscriptions[team] = sub
		b.mu.Unlock()
	}
	msg = msg.R("event")
	msgType := msg.S("type")
	switch msgType {
	case "message":
		msgUser := msg.S("user")
		// If it's our message - no need to do anything
		if msgUser == sub.team.BotUserID {
			return
		}
		text := msg.S("text")
		ltext := strings.ToLower(text)
		channel := msg.S("channel")
		push := false
		// If this is an internal command to us we should not check hashes, etc.
		if !(msg.S("subtype") == "" && channel != "" && channel[0] == 'D' &&
			(strings.HasPrefix(ltext, "join ") || strings.HasPrefix(ltext, "verbose ") || ltext == "config" ||
				text == "?" || strings.HasPrefix(ltext, "help") || strings.HasPrefix(ltext, "vt ") ||
				strings.HasPrefix(ltext, "xfe "))) {
			if msg.S("subtype") == "" {
				push = strings.Contains(ltext, "<http") || ipReg.MatchString(text) || md5Reg.MatchString(text) || sha1Reg.MatchString(text) || sha256Reg.MatchString(text)
			}
		}
		// If we need to handle the message, pass it to the queue
		if push {
			logrus.Debugf("Handling message - %+v\n", msg)
			workReq := domain.WorkRequestFromMessage(msg, sub.team.BotToken, sub.team.VTKey, sub.team.XFEKey, sub.team.XFEPass)
			logrus.Debug("Pushing to queue")
			ctx := &domain.Context{Team: team, User: msgUser, Type: msgType, Channel: channel, OriginalUser: msgUser}
			workReq.ReplyQueue, workReq.Context = b.replyQueue, ctx
			b.q.PushWork(workReq)
		} else {
			// Handle some internal commands
			if channel != "" && channel[0] == 'D' {
				switch {
				case strings.HasPrefix(text, "join "):
					b.joinChannels(team, text, channel)
				case strings.HasPrefix(text, "verbose "):
					b.handleVerbose(team, text, channel) // Need the actual channel IDs
				case text == "config":
					b.handleConfig(team, msg)
				case text == "?" || strings.HasPrefix(text, "help"):
					b.showHelp(team, channel)
				case strings.HasPrefix(text, "vt "):
					b.handleVT(team, text, channel)
				case strings.HasPrefix(text, "xfe "):
					b.handleXFE(team, text, channel)
				}
			}
			b.smu.Lock()
			defer b.smu.Unlock()
			stats := b.stats[team]
			if stats == nil {
				stats = &domain.Statistics{Team: team}
				b.stats[team] = stats
			}
			stats.Messages++
		}
	case "file_shared":
		// TODO
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
	err := b.r.BotHeartbeat()
	if err != nil {
		return err
	}
	err = b.loadSubscriptions()
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
			return nil
		case <-ticker.C:
			err := b.r.BotHeartbeat()
			if err != nil {
				logrus.Errorf("Unable to update heartbeat - %v\n", err)
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
func (b *Bot) subscriptionChanged(configuration *domain.Configuration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := b.subscriptions[configuration.Team]
	if sub == nil {
		logrus.Debugf("Subscription for team not found: %s\n", configuration.Team)
		return
	}
	sub.configuration = configuration
}

func (b *Bot) monitorChanges() {
	for {
		configuration, err := b.q.PopConf(0)
		if err != nil || configuration == nil {
			logrus.Infof("Quiting monitoring changes - %v\n", err)
			break
		}
		logrus.Debugf("Configuration change received: %+v\n", configuration)
		b.subscriptionChanged(configuration)
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
