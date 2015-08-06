package queue

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/slack"
)

type logger struct {
}

func (l *logger) Log(args ...interface{}) {
	logrus.Info(args...)
}

type queueSQS struct {
	svc        *sqs.SQS
	confURL    *string
	messageURL *string
	workURL    *string
	closed     bool
}

func newSQS() (*queueSQS, error) {
	svc := sqs.New(aws.NewConfig().WithCredentials(
		credentials.NewStaticCredentials(conf.Options.AWS.ID, conf.Options.AWS.Secret, "")).WithLogLevel(
		aws.LogDebug).WithLogger(&logger{}).WithMaxRetries(-1).WithRegion("us-west-2").WithHTTPClient(http.DefaultClient))
	// Make sure that the queues we are interested in exist
	queues := []string{conf.Options.AWS.ConfQueueName, conf.Options.AWS.MessageQueueName, conf.Options.AWS.WorkQueueName}
	names := make([]*string, len(queues))
	for i, q := range queues {
		r, err := svc.CreateQueue(&sqs.CreateQueueInput{
			Attributes: map[string]*string{
				"MaximumMessageSize":     aws.String("262144"),
				"MessageRetentionPeriod": aws.String("360"),
				"VisibilityTimeout":      aws.String("60"),
			},
			QueueName: aws.String(q),
		})
		if err != nil {
			return nil, err
		}
		names[i] = r.QueueURL
	}
	return &queueSQS{svc: svc, confURL: names[0], messageURL: names[1], workURL: names[2]}, nil
}

func (q *queueSQS) push(qname *string, body interface{}) error {
	if q.closed {
		return ErrClosed
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = q.svc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(b)),
		QueueURL:    qname})
	return err
}

func (q *queueSQS) pop(qname *string, timeout time.Duration, body interface{}) error {
	start := time.Now()
	for !q.closed && (timeout <= 0 || start.Add(timeout).Before(time.Now())) {
		r, err := q.svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			AttributeNames:  []*string{aws.String("All")},
			QueueURL:        qname,
			WaitTimeSeconds: aws.Int64(20),
		})
		if err != nil {
			return err
		}
		if len(r.Messages) == 0 {
			continue
		}
		err = json.Unmarshal([]byte(*r.Messages[0].Body), body)
		if err != nil {
			return err
		}
		// Now, let's just delete the message
		_, err = q.svc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueURL:      qname,
			ReceiptHandle: r.Messages[0].ReceiptHandle,
		})
		if err != nil {
			logrus.Warnf("Unable to delete message from queue %s - %v", *qname, err)
		}
		return nil
	}
	if q.closed {
		return ErrClosed
	}
	return ErrTimeout
}

func (q *queueSQS) PushConf(u *domain.User, c *domain.Configuration) error {
	confMessage := ConfigurationMessage{User: *u, Configuration: *c}
	return q.push(q.confURL, &confMessage)
}

func (q *queueSQS) PopConf(timeout time.Duration) (*domain.User, *domain.Configuration, error) {
	var msg ConfigurationMessage
	err := q.pop(q.confURL, timeout, &msg)
	if err != nil {
		return nil, nil, err
	}
	return &msg.User, &msg.Configuration, nil
}

func (q *queueSQS) PushMessage(data *slack.Message) error {
	return q.push(q.messageURL, data)
}

func (q *queueSQS) PopMessage(timeout time.Duration) (*slack.Message, error) {
	msg := &slack.Message{}
	err := q.pop(q.messageURL, timeout, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (q *queueSQS) PushWork(data *slack.Message) error {
	return q.push(q.workURL, data)
}

func (q *queueSQS) PopWork(timeout time.Duration) (*slack.Message, error) {
	msg := &slack.Message{}
	err := q.pop(q.workURL, timeout, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (q *queueSQS) Close() error {
	q.closed = true
	return nil
}
