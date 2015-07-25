// Package queue abstracts the various external (or internal) message queues we are using for notifications
package queue

import (
	"fmt"
	"time"

	"github.com/demisto/alfred/domain"
	"github.com/demisto/slack"
)

// ConfigurationMessage including the user and configuration
type ConfigurationMessage struct {
	User          domain.User
	Configuration domain.Configuration
}

// Queue abstracts the external / internal queues
type Queue interface {
	PushConf(u *domain.User, c *domain.Configuration) error
	PopConf(timeout time.Duration) (*domain.User, *domain.Configuration, error)
	PushMessage(msg *slack.Message) error
	PopMessage(timeout time.Duration) (*slack.Message, error)
	PushWork(msg *slack.Message) error
	PopWork(timeout time.Duration) (*slack.Message, error)
	Close() error
}

// New queue is returned depending on environment
func New() Queue {
	var q queueChannel
	q.Conf = make(chan ConfigurationMessage, 100)
	q.Dedup = make(chan slack.Message, 100)
	q.Work = make(chan slack.Message, 100)
	return &q
}

type queueChannel struct {
	Conf  chan ConfigurationMessage
	Dedup chan slack.Message
	Work  chan slack.Message
}

func (q *queueChannel) PushConf(u *domain.User, c *domain.Configuration) error {
	q.Conf <- ConfigurationMessage{User: *u, Configuration: *c}
	return nil
}

// Pop a value from the queue - the simple channl implementation ignores timeout
func (q *queueChannel) PopConf(timeout time.Duration) (*domain.User, *domain.Configuration, error) {
	conf := <-q.Conf
	// If someone closed the channel
	if conf.User.ID == "" {
		return nil, nil, fmt.Errorf("Closed")
	}
	return &conf.User, &conf.Configuration, nil
}

func (q *queueChannel) PushMessage(data *slack.Message) error {
	q.Dedup <- *data
	return nil
}

func (q *queueChannel) PopMessage(timeout time.Duration) (*slack.Message, error) {
	msg := <-q.Dedup
	return &msg, nil
}

func (q *queueChannel) PushWork(data *slack.Message) error {
	q.Work <- *data
	return nil
}

func (q *queueChannel) PopWork(timeout time.Duration) (*slack.Message, error) {
	work := <-q.Work
	return &work, nil
}

func (q *queueChannel) Close() error {
	close(q.Conf)
	close(q.Work)
	return nil
}
