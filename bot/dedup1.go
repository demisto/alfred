package bot

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/queue"
)

// Dedup is responsible for pulling messages from the queue and passing only relevant ones to work
type Dedup struct {
	handledMessages map[string]map[string]*time.Time // message map per team of messages we already handled
	q               queue.Queue
	cleanTime       time.Time
}

// NewDedup with the queue
func NewDedup(q queue.Queue) *Dedup {
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
		if err != nil || msg == nil {
			logrus.Infoln("Stoping DEDUP process")
			return
		}
		logrus.Debugf("Deduping message %s\n", msg.MessageID)
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
			logrus.Debugf("Pushing message %s to work\n", msg.MessageID)
			d.q.PushWork(msg)
		}
		counter++
	}
}

func (d *Dedup) alreadyHandled(data *domain.WorkRequest) bool {
	context, err := GetContext(data.Context)
	if err != nil {
		logrus.Warnf("Unknown context for message %+v\n", data)
		return true
	}
	handled := d.handledMessages[context.Team]
	if handled == nil {
		handled = make(map[string]*time.Time)
		d.handledMessages[context.Team] = handled
	}
	if data.MessageID == "" {
		// Ignore the message
		return true
	}
	if handled[data.MessageID] != nil {
		return true
	}
	now := time.Now()
	handled[data.MessageID] = &now
	return false
}
