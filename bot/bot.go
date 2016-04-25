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
	maxUsersPerBot = 200
)

// subscription holds the interest we have for each team
type subscription struct {
	team          *domain.Team          // the team we are subscribed to
	users         []domain.User         // The list of users that added us to the team (can be several)
	configuration *domain.Configuration // The configuration of channels, mainly for verbose
	s             *slack.Slack          // the slack client on the bot token
	started       bool                  // did we start subscription for this guy
	ts            time.Time             // When did we start the WS
	info          *slack.RTMStartReply  // The team info from Slack
}

func (sub *subscription) ChannelName(channel string) string {
	if channel == "" {
		return ""
	}
	// if not found, it might be a new channel or group
	switch channel[0] {
	case 'C':
		for i := range sub.info.Channels {
			if channel == sub.info.Channels[i].ID {
				return sub.info.Channels[i].Name
			}
		}
		info, err := sub.s.ChannelInfo(channel)
		if err != nil {
			logrus.WithField("error", err).Warn("Unable to get channel info\n")
			return ""
		}
		sub.info.Channels = append(sub.info.Channels, info.Channel)
		return info.Channel.Name
	case 'G':
		for i := range sub.info.Groups {
			if channel == sub.info.Groups[i].ID {
				return sub.info.Groups[i].Name
			}
		}
		info, err := sub.s.GroupInfo(channel)
		if err != nil {
			logrus.WithField("error", err).Warn("Unable to get group info\n")
			return ""
		}
		sub.info.Groups = append(sub.info.Groups, info.Group)
		return info.Group.Name
	}
	return ""
}

func (sub *subscription) ChannelID(channel string) string {
	if channel == "" {
		return ""
	}
	channel = strings.ToLower(channel)
	for i := range sub.info.Channels {
		if channel == strings.ToLower(sub.info.Channels[i].Name) {
			return sub.info.Channels[i].ID
		}
	}
	// Might be a new channel
	if list, err := sub.s.ChannelList(true); err == nil {
		for i := range list.Channels {
			if channel == strings.ToLower(list.Channels[i].Name) {
				return list.Channels[i].ID
			}
		}
	}
	for i := range sub.info.Groups {
		if strings.ToLower(channel) == strings.ToLower(sub.info.Groups[i].Name) {
			return sub.info.Groups[i].ID
		}
	}
	// Might be a new private channel
	if list, err := sub.s.GroupList(true); err == nil {
		for i := range list.Groups {
			if channel == strings.ToLower(list.Groups[i].Name) {
				return list.Groups[i].ID
			}
		}
	}
	return ""
}

// Bot iterates on all subscriptions and listens / responds to messages
type Bot struct {
	in            chan *slack.Message
	stop          chan bool
	r             repo.Repo
	mu            sync.RWMutex // Guards the subscriptions
	subscriptions map[string]*subscription
	q             queue.Queue // Message queue for configuration updates
	replyQueue    string
	smu           sync.Mutex // Guards the statistics
	stats         map[string]*domain.Statistics
	firstMessages map[string]bool
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
		subscriptions: make(map[string]*subscription),
		q:             q,
		replyQueue:    host,
		stats:         make(map[string]*domain.Statistics),
		firstMessages: make(map[string]bool),
	}, nil
}

// loadSubscriptions loads all subscriptions per team
func (b *Bot) loadSubscriptions(includingMine bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	cnt := len(b.subscriptions)
	// Fully loaded, no room for others
	if cnt >= maxUsersPerBot {
		return nil
	}
	// Everything must run in separate transactions as it is written here
	openTeams, err := b.r.OpenTeams(includingMine)
	if err != nil {
		return err
	}
	for i := range openTeams {
		// Don't lock our own users
		if !includingMine || openTeams[i].Bot != b.r.BotName() {
			logrus.Debugf("Trying to lock team %s", openTeams[i].Team)
			locked, err := b.r.LockTeam(&openTeams[i])
			if err != nil || !locked {
				logrus.Debugf("Unable to lock team %v", err)
				continue
			}
		} else {
			logrus.Debugf("Team %s is mine", openTeams[i].Team)
		}
		t, err := b.r.Team(openTeams[i].Team)
		if err != nil {
			logrus.Warnf("Error loading team %s - %v\n", openTeams[i].Team, err)
			continue
		}
		// Just precaution, if we already have the team just skip this
		// This is required for bolt repo implementation
		teamSub := b.subscriptions[t.ID]
		if teamSub != nil {
			continue
		}
		teamSub = &subscription{team: t}
		teamSub.configuration, err = b.r.ChannelsAndGroups(t.ID)
		if err != nil {
			logrus.Warnf("Error loading team configuration - %v\n", err)
			continue
		}
		teamSub.users, err = b.r.TeamMembers(t.ID)
		if err != nil {
			logrus.Warnf("Error loading team members - %v\n", err)
			continue
		}
		teamSub.s, err = slack.New(slack.SetToken(t.BotToken),
			slack.SetErrorLog(log.New(conf.LogWriter, "SLACK:", log.Lshortfile)))
		if err != nil {
			logrus.Warnf("Error opening Slack for team %s (%s) - %v\n", t.ID, t.Name, err)
			continue
		}
		b.subscriptions[t.ID] = teamSub
		cnt++
		if cnt >= maxUsersPerBot {
			break
		}
	}
	return nil
}

func (b *Bot) startWSForTeam(team string, teamSub *subscription) error {
	logrus.Infof("Starting WS for team %s (%s)\n", team, teamSub.team.Name)
	info, err := teamSub.s.RTMStart("dbot.demisto.com", b.in, &domain.Context{Team: team})
	if err != nil {
		logrus.Warnf("Unable to start WS for team %s (%s) - %v\n", team, teamSub.team.Name, err)
		// For revoked tokens, the user is not active anymore
		errStr := err.Error()
		if strings.Contains(errStr, "token_revoked") || strings.Contains(errStr, "account_inactive") {
			teamSub.team.Status = domain.UserStatusDeleted
			logrus.Infof("Updating team status for %s (%s)", team, teamSub.team.Name)
			dbErr := b.r.SetTeam(teamSub.team)
			if dbErr != nil {
				logrus.Warnf("Unable to change team status - %v", dbErr)
			}
			logrus.Infof("Unlocking team %s (%s)", team, teamSub.team.Name)
			dbErr = b.r.UnlockTeam(team)
			if dbErr != nil {
				logrus.Warnf("Unable to unlock team - %v", dbErr)
			}
		}
		return err
	}
	teamSub.info = info
	teamSub.started = true
	teamSub.ts = time.Now()
	return nil
}

func (b *Bot) startWS() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var subsToClean []string
	for k, v := range b.subscriptions {
		if !v.started && v.team.Status == domain.UserStatusActive {
			err := b.startWSForTeam(k, v)
			if err != nil && (strings.Contains(err.Error(), "token_revoked") || strings.Contains(err.Error(), "account_inactive")) {
				subsToClean = append(subsToClean, k)
			}
		}
	}
	// Remove all the ones that were rejected
	for _, s := range subsToClean {
		logrus.Debugf("Cleaning team %s", s)
		delete(b.subscriptions, s)
	}
	return nil
}

func (b *Bot) stopWS() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range b.subscriptions {
		logrus.Infof("Stoping subscription for team - %s\n", k)
		logrus.Infof("Stoping WS for team - %s (%s)\n", k, v.team.Name)
		err := v.s.RTMStop()
		if err != nil {
			logrus.Warnf("Unable to stop subscription for team %s - %v\n", k, err)
		}
		v.started = false
	}
}

var (
	ipReg  = regexp.MustCompile("\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b")
	md5Reg = regexp.MustCompile("\\b[a-fA-F\\d]{32}\\b")
)

func (b *Bot) handleMessage(msg *slack.Message) {
	if msg == nil {
		return
	}
	ctx, err := GetContext(msg.Context)
	if err != nil {
		logrus.Warnf("Unable to get context from message - %+v\n", msg)
		return
	}
	sub := b.subscriptions[ctx.Team]
	if sub == nil {
		logrus.Warnf("Unable to find team %s in subscriptions", ctx.Team)
		return
	}
	switch msg.Type {
	case "message":
		// If it's our message - no need to do anything
		if msg.User == sub.team.BotUserID {
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
		// If we need to handle the message, pass it to the queue
		if push {
			logrus.Debugf("Handling message - %+v\n", msg)
			workReq := domain.WorkRequestFromMessage(msg, sub.team.BotToken)
			t, err := slack.TimestampToTime(workReq.MessageID)
			if err != nil {
				logrus.Warnf("Unable to get message timestamp %s - %v", workReq.MessageID, err)
				return
			}
			if sub.started && sub.ts.Before(t) {
				logrus.Debug("Pushing to queue")
				ctx.OriginalUser, ctx.Channel, ctx.Type = msg.User, msg.Channel, msg.Type
				workReq.ReplyQueue, workReq.Context = b.replyQueue, ctx
				b.q.PushWork(workReq)
			} else {
				logrus.Infof("Got old message from Slack - %s", workReq.MessageID)
			}
		} else {
			b.smu.Lock()
			defer b.smu.Unlock()
			// Handle some internal commands
			if msg.Channel != "" && msg.Channel[0] == 'D' {
				text := strings.ToLower(msg.Text)
				switch {
				case strings.HasPrefix(text, "join "):
					b.joinChannels(ctx.Team, msg.Text, msg.Channel)
				case strings.HasPrefix(text, "verbose "):
					b.handleVerbose(ctx.Team, msg.Text, msg.Channel) // Need the actual channel IDs
				case text == "config":
					b.handleConfig(ctx.Team, msg)
				case text == "?" || strings.HasPrefix(text, "help"):
					b.showHelp(ctx.Team, msg.Channel)
				}
			}
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
	err := b.r.BotHeartbeat()
	if err != nil {
		return err
	}
	err = b.loadSubscriptions(true)
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
					logrus.Fatal("Message channel from Slack closed - should never happen")
				} else {
					logrus.Infof("Got error message from channel %+v\n", msg)
					// If this is an unmarshall error just ignore the message
					if msg.Error.Unmarshall {
						continue
					}
					// Check if we need to restart the channel
					ctx, err := GetContext(msg.Context)
					if err != nil {
						logrus.Warnf("Unable to get context from message, not restarting - %+v", msg)
					} else {
						b.mu.Lock()
						teamSub := b.subscriptions[ctx.Team]
						if teamSub != nil {
							if teamSub.started {
								b.startWSForTeam(ctx.Team, teamSub)
							}
						} else {
							logrus.Warn("Team subscription not found, not restarting")
						}
						b.mu.Unlock()
					}
				}
				continue
			}
			b.handleMessage(msg)
		case <-ticker.C:
			err := b.r.BotHeartbeat()
			if err != nil {
				logrus.Errorf("Unable to update heartbeat - %v\n", err)
			}
			err = b.loadSubscriptions(false)
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
func (b *Bot) subscriptionChanged(team string, configuration *domain.Configuration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := b.subscriptions[team]
	if sub == nil {
		logrus.Debugf("Subscription for team not found: %s\n", team)
		return
	}
	sub.configuration = configuration
}

func (b *Bot) monitorChanges() {
	for {
		team, configuration, err := b.q.PopConf(0)
		if err != nil || team == "" || configuration == nil {
			logrus.Infof("Quiting monitoring changes - %v\n", err)
			// Go down
			b.Stop()
			break
		}
		logrus.Debugf("Configuration change received: %s, %+v\n", team, configuration)
		b.subscriptionChanged(team, configuration)
	}
}

func (b *Bot) monitorReplies() {
	for {
		reply, err := b.q.PopWorkReply(b.replyQueue, 0)
		if err != nil || reply == nil {
			logrus.Infof("Quiting monitoring replies - %v\n", err)
			// Go down
			b.Stop()
			break
		}
		b.handleReply(reply)
	}
}
