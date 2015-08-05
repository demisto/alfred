package dedup

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/slack"
)

// Dedup is responsible for pulling messages from the queue and passing only relevant ones to work
type Dedup struct {
	handledMessages map[string]map[string]*time.Time // message map per team of messages we already handled
	q               queue.Queue
	cleanTime       time.Time
}

// New Dedup with the queue
func New(q queue.Queue) *Dedup {
	return &Dedup{
		q:               q,
		handledMessages: make(map[string]map[string]*time.Time),
		cleanTime:       time.Now(),
	}
}

// Start the dedup process. To stop, just close the queue.
func (d *Dedup) Start() {
	counter := 0
	for {
		msg, err := d.q.PopMessage(0)
		if err != nil {
			logrus.Info("Stoping DEDUP process")
			return
		}
		// Check time only every 1000 messages the messages we should clear
		if counter >= 1000 {
			counter = 0
			if time.Now().Sub(d.cleanTime) > 5*time.Minute {
				for _, v := range d.handledMessages {
					for k, t := range v {
						if time.Since(*t) > 5*time.Minute {
							delete(v, k)
						}
					}
				}
			}
		}
		if !d.alreadyHandled(msg) {
			d.q.PushWork(msg)
		}
		counter++
	}
}

func (d *Dedup) alreadyHandled(original *slack.Message) bool {
	var data *bot.Context
	switch c := original.Context.(type) {
	case *bot.Context:
		data = c
	case map[string]interface{}:
		data = &bot.Context{Team: c["Team"].(string), User: c["User"].(string)}
	default:
		logrus.Warnf("Unknown context for message %+v\n", original)
		return true
	}
	handled := d.handledMessages[data.Team]
	if handled == nil {
		handled = make(map[string]*time.Time)
		d.handledMessages[data.Team] = handled
	}
	var field string
	// We care only about messages
	if original.Type == "message" {
		switch original.Subtype {
		case "file_share":
			// field = original.File.Name + "|" + original.User
			field = original.Timestamp
		case "message_changed":
			// field = original.Message.Text + "|" + original.User
			field = original.Message.Timestamp
		default:
			// field = original.Text + "|" + original.User
			field = original.Timestamp
		}
	}
	if field == "" {
		// Ignore the message
		return true
	}
	if handled[field] != nil {
		return true
	}
	now := time.Now()
	handled[field] = &now
	return false
}
