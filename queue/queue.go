// Package queue abstracts the various external (or internal) message queues we are using for notifications
package queue

import (
	"errors"
	"time"

	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
)

var (
	// ErrTimeout is returned if receive encounters a timeout
	ErrTimeout = errors.New("timeout occurred")
	// ErrClosed is returned if you try to access a closed queue
	ErrClosed = errors.New("queue is already closed")
)

// Queue abstracts the external / internal queues
type Queue interface {
	PushConf(team string) error
	PopConf(timeout time.Duration) (string, error)
	PushWork(work *domain.WorkRequest) error
	PopWork(timeout time.Duration) (*domain.WorkRequest, error)
	PushWorkReply(replyQueue string, reply *domain.WorkReply) error
	PopWorkReply(replyQueue string, timeout time.Duration) (*domain.WorkReply, error)
	Close() error
}

// New queue is returned depending on environment
func New(r *repo.MySQL) (Queue, error) {
	return NewDBQueue(r), nil
}
