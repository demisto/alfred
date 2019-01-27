package queue

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/util"
)

// dbQueue implements the queue functionality using a database backend
type dbQueue struct {
	d            *repo.MySQL
	done         chan bool
	conf         chan string
	work         chan *domain.WorkRequest
	workReply    chan *domain.WorkReply
	webWorkReply map[string]chan *domain.WorkReply
	mux          sync.Mutex
	closed       bool
}

func NewDBQueue(r *repo.MySQL) *dbQueue {
	q := &dbQueue{
		d:            r,
		conf:         make(chan string, 1000),
		work:         make(chan *domain.WorkRequest, 1000),
		workReply:    make(chan *domain.WorkReply, 1000),
		webWorkReply: make(map[string]chan *domain.WorkReply),
		done:         make(chan bool),
	}
	go q.getMessages()
	return q
}

// PushConf ...
func (dq *dbQueue) PushConf(team string) error {
	m := domain.DBQueueMessage{MessageType: "conf", Message: team}
	return dq.d.PostMessageToAll(&m)
}

// PopConf ...
func (dq *dbQueue) PopConf(timeout time.Duration) (string, error) {
	team := <-dq.conf
	// If someone closed the channel
	if team == "" {
		return "", ErrClosed
	}
	return team, nil
}

// PushWork ...
func (dq *dbQueue) PushWork(work *domain.WorkRequest) error {
	_, err := domain.GetContext(work.Context)
	if err != nil {
		return err
	}
	m := domain.DBQueueMessage{MessageType: "work", Message: util.ToJSONStringNoIndent(work), Name: work.ReplyQueue}
	return dq.d.PostMessage(&m)
}

// PopWork ...
func (dq *dbQueue) PopWork(timeout time.Duration) (*domain.WorkRequest, error) {
	work := <-dq.work
	if work == nil {
		return nil, ErrClosed
	}
	return work, nil
}

// PushWorkReply ...
func (dq *dbQueue) PushWorkReply(replyQueue string, reply *domain.WorkReply) error {
	_, err := domain.GetContext(reply.Context)
	if err != nil {
		return err
	}
	m := domain.DBQueueMessage{MessageType: "workr", Message: util.ToJSONStringNoIndent(reply), Name: replyQueue}
	return dq.d.PostMessage(&m)
}

// PopWorkReply ...
func (dq *dbQueue) PopWorkReply(replyQueue string, timeout time.Duration) (work *domain.WorkReply, err error) {
	if replyQueue == util.Hostname {
		work = <-dq.workReply
	} else {
		var ch chan *domain.WorkReply
		var ok bool
		dq.mux.Lock()
		if ch, ok = dq.webWorkReply[replyQueue]; !ok {
			ch = make(chan *domain.WorkReply, 1)
			dq.webWorkReply[replyQueue] = ch
		}
		dq.mux.Unlock()
		work = <-ch
		close(ch)
		dq.mux.Lock()
		delete(dq.webWorkReply, replyQueue)
		dq.mux.Unlock()
	}
	if work == nil {
		return nil, ErrClosed
	}
	return work, nil
}

func (dq *dbQueue) Close() error {
	dq.done <- true
	if !dq.closed {
		dq.closed = true
		close(dq.conf)
		close(dq.work)
		close(dq.workReply)
		dq.mux.Lock()
		for _, ch := range dq.webWorkReply {
			close(ch)
		}
		dq.mux.Unlock()
	}
	return nil
}

func (dq *dbQueue) getMessages() {
	t := time.NewTicker(time.Duration(conf.Options.QueuePoll) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-dq.done:
			return
		case <-t.C:
			if conf.Options.Worker {
				messages, err := dq.d.QueueMessages(nil, "work")
				if err != nil {
					logrus.WithError(err).Error("Unable to load worker messages - going to retry")
				}
				for _, m := range messages {
					wr := &domain.WorkRequest{}
					if err := json.Unmarshal([]byte(m.Message), wr); err != nil {
						logrus.WithError(err).Error("Unable to parse work request message")
						continue
					}
					dq.work <- wr
				}
			}
			if conf.Options.Web {
				names := []string{util.Hostname}
				dq.mux.Lock()
				for k := range dq.webWorkReply {
					names = append(names, k)
				}
				dq.mux.Unlock()
				messages, err := dq.d.QueueMessages(names, "workr")
				if err != nil {
					logrus.WithError(err).Error("Unable to load web workr messages - going to retry")
				}
				for _, m := range messages {
					wr := &domain.WorkReply{}
					if err := json.Unmarshal([]byte(m.Message), wr); err != nil {
						logrus.WithError(err).Errorf("Unable to parse work reply message. got message - %s", m.Message)
						continue
					}
					// If this is a reply to Slack just push it to generic queue
					if m.Name == util.Hostname {
						dq.workReply <- wr
					} else {
						// Otherwise, make sure to push to the specific web waiter
						var ch chan *domain.WorkReply
						var ok bool
						dq.mux.Lock()
						if ch, ok = dq.webWorkReply[m.Name]; !ok {
							ch = make(chan *domain.WorkReply, 1)
							dq.webWorkReply[m.Name] = ch
						}
						dq.mux.Unlock()
						ch <- wr
					}
				}
			}
			if conf.Options.Web {
				messages, err := dq.d.QueueMessages([]string{util.Hostname}, "conf")
				if err != nil {
					logrus.WithError(err).Error("Unable to load web conf messages - going to retry")
				}
				for _, m := range messages {
					dq.conf <- m.Message
				}
			}
		}
	}
}
