package queue

import (
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/util"
)

// dbQueue implements the queue functionality using a database backend
type dbQueue struct {
	d    *repo.MySQL
	qc   *queueChannel
	done chan bool
}

func NewDBQueue(r *repo.MySQL) *dbQueue {
	q := &dbQueue{
		d: r,
		qc: &queueChannel{
			Conf:      make(chan *domain.Configuration, 1000),
			Work:      make(chan *domain.WorkRequest, 1000),
			WorkReply: make(chan *domain.WorkReply, 1000),
		},
	}
	go q.getMessages()
	return q
}

// PushConf ...
func (dq *dbQueue) PushConf(c *domain.Configuration) error {
	m := domain.DBQueueMessage{MessageType: "conf", Message: util.ToJSONStringNoIndent(c)}
	return dq.d.PostMessage(&m)
}

// PopConf ...
func (dq *dbQueue) PopConf(timeout time.Duration) (*domain.Configuration, error) {
	return dq.qc.PopConf(timeout)
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
	return dq.qc.PopWork(timeout)
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
func (dq *dbQueue) PopWorkReply(replyQueue string, timeout time.Duration) (*domain.WorkReply, error) {
	return dq.qc.PopWorkReply(replyQueue, timeout)
}

func (dq *dbQueue) Close() error {
	dq.done <- true
	return dq.qc.Close()
}

func (dq *dbQueue) getMessages() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-dq.done:
			return
		case <-t.C:
			if conf.Options.Worker {
				messages, err := dq.d.QueueMessages(false, "work")
				if err != nil {
					logrus.WithError(err).Error("Unable to load worker messages - going to retry")
				}
				for _, m := range messages {
					wr := &domain.WorkRequest{}
					if err := json.Unmarshal([]byte(m.Message), wr); err != nil {
						logrus.WithError(err).Error("Unable to parse work request message")
						continue
					}
					dq.qc.PushWork(wr)
				}
			}
			if conf.Options.Web {
				messages, err := dq.d.QueueMessages(true, "workr")
				if err != nil {
					logrus.WithError(err).Error("Unable to load web workr messages - going to retry")
				}
				for _, m := range messages {
					wr := &domain.WorkReply{}
					if err := json.Unmarshal([]byte(m.Message), wr); err != nil {
						logrus.WithError(err).Error("Unable to parse work reply message")
						continue
					}
					dq.qc.PushWorkReply("", wr)
				}
			}
			if conf.Options.Web {
				messages, err := dq.d.QueueMessages(true, "conf")
				if err != nil {
					logrus.WithError(err).Error("Unable to load web conf messages - going to retry")
				}
				for _, m := range messages {
					cr := &domain.Configuration{}
					if err := json.Unmarshal([]byte(m.Message), cr); err != nil {
						logrus.WithError(err).Error("Unable to parse configuration message")
						continue
					}
					dq.qc.PushConf(cr)
				}
			}
		}
	}
}
