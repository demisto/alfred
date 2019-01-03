// Package queue abstracts the various external (or internal) message queues we are using for notifications
package queue

import (
	"errors"
	"os"
	"time"

	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
)

var (
	// ErrTimeout is returned if receive encounters a timeout
	ErrTimeout = errors.New("timeout occured")
	// ErrClosed is returned if you try to access a closed queue
	ErrClosed = errors.New("queue is already closed")
)

// Queue abstracts the external / internal queues
type Queue interface {
	PushConf(c *domain.Configuration) error
	PopConf(timeout time.Duration) (*domain.Configuration, error)
	PushWork(work *domain.WorkRequest) error
	PopWork(timeout time.Duration) (*domain.WorkRequest, error)
	PushWorkReply(replyQueue string, reply *domain.WorkReply) error
	PopWorkReply(replyQueue string, timeout time.Duration) (*domain.WorkReply, error)
	Close() error
}

// New queue is returned depending on environment
func New(r *repo.MySQL) (Queue, error) {
	var q Queue
	var err error
	switch {
	case conf.Options.DB.ConnectString != "":
		q = NewDBQueue(r)
	default:
		q = &queueChannel{
			Conf:      make(chan *domain.Configuration, 1000),
			Work:      make(chan *domain.WorkRequest, 1000),
			WorkReply: make(chan *domain.WorkReply, 1000),
		}
	}
	return q, err
}

type queueChannel struct {
	Conf      chan *domain.Configuration
	Work      chan *domain.WorkRequest
	WorkReply chan *domain.WorkReply
	closed    bool
}

func (q *queueChannel) PushConf(c *domain.Configuration) error {
	q.Conf <- c
	return nil
}

// Pop a value from the queue - the simple channel implementation ignores timeout
func (q *queueChannel) PopConf(timeout time.Duration) (*domain.Configuration, error) {
	conf := <-q.Conf
	// If someone closed the channel
	if conf == nil {
		return nil, ErrClosed
	}
	return conf, nil
}

func (q *queueChannel) PushWork(data *domain.WorkRequest) error {
	q.Work <- data
	return nil
}

func (q *queueChannel) PopWork(timeout time.Duration) (*domain.WorkRequest, error) {
	work := <-q.Work
	if work == nil {
		return nil, ErrClosed
	}
	return work, nil
}

func (q *queueChannel) PushWorkReply(replyQueue string, reply *domain.WorkReply) error {
	q.WorkReply <- reply
	return nil
}

func (q *queueChannel) PopWorkReply(replyQueue string, timeout time.Duration) (*domain.WorkReply, error) {
	work := <-q.WorkReply
	if work == nil {
		return nil, ErrClosed
	}
	return work, nil
}

// ReplyQueueName returns the default name
func ReplyQueueName() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return host, nil
}

func (q *queueChannel) Close() error {
	if !q.closed {
		close(q.Conf)
		close(q.Work)
		close(q.WorkReply)
	}
	q.closed = true
	return nil
}
