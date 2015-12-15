package queue

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	pubsub "google.golang.org/api/pubsub/v1"
)

type queuePubSub struct {
	svc    *pubsub.Service
	closed bool
	host   string
}

// Returns a fully qualified resource name for Cloud Pub/Sub.
func fqrn(res, proj, name string) string {
	return fmt.Sprintf("projects/%s/%s/%s", proj, res, name)
}

func fullTopicName(proj, topic string) string {
	return fqrn("topics", proj, topic)
}

func fullSubName(proj, topic string) string {
	return fqrn("subscriptions", proj, topic)
}

func newPubSub() (*queuePubSub, error) {
	// Start with an OAuth http client wrapper
	jsonCreds, err := json.Marshal(conf.Options.G.Credentials)
	if err != nil {
		return nil, err
	}
	config, err := google.JWTConfigFromJSON(jsonCreds, pubsub.PubsubScope)
	if err != nil {
		return nil, err
	}
	client := config.Client(oauth2.NoContext)
	svc, err := pubsub.New(client)
	if err != nil {
		return nil, err
	}
	var host string
	names := []string{conf.Options.G.ConfName, conf.Options.G.WorkName}
	// If we are a bot or a web tier, create a reply queue for us
	if conf.Options.Bot || conf.Options.Web {
		host, err = ReplyQueueName()
		if err != nil {
			return nil, err
		}
		if conf.Options.Bot {
			names = append(names, host)
		}
		if conf.Options.Web {
			names = append(names, host+"-web")
		}
	}
	for _, n := range names {
		// Register the topics while ignoring already exists errors
		_, err := svc.Projects.Topics.Create(fullTopicName(conf.Options.G.Project, n), &pubsub.Topic{}).Do()
		if err != nil && !strings.Contains(err.Error(), "409") {
			return nil, err
		}
		sub := &pubsub.Subscription{Topic: fullTopicName(conf.Options.G.Project, n)}
		// Each bot will have it's own subscription
		if conf.Options.Bot && n == conf.Options.G.ConfName {
			n += "-" + host
		}
		_, err = svc.Projects.Subscriptions.Create(fullSubName(conf.Options.G.Project, n), sub).Do()
		if err != nil && !strings.Contains(err.Error(), "409") {
			return nil, err
		}
	}
	return &queuePubSub{svc: svc, host: host}, nil
}

func (q *queuePubSub) push(qname string, body interface{}) error {
	if q.closed {
		return ErrClosed
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	pubsubMessage := &pubsub.PubsubMessage{
		Data: base64.StdEncoding.EncodeToString(b),
	}
	publishRequest := &pubsub.PublishRequest{
		Messages: []*pubsub.PubsubMessage{pubsubMessage},
	}
	_, err = q.svc.Projects.Topics.Publish(
		fullTopicName(conf.Options.G.Project, qname), publishRequest).Do()
	return err
}

func (q *queuePubSub) pop(qname string, timeout time.Duration, body interface{}) error {
	if q.closed {
		return ErrClosed
	}
	pullRequest := &pubsub.PullRequest{
		ReturnImmediately: false,
		MaxMessages:       1,
	}
	subName := fullSubName(conf.Options.G.Project, qname)
	var pullResponse *pubsub.PullResponse
	var err error
	started := time.Now()
	for {
		pullResponse, err = q.svc.Projects.Subscriptions.Pull(subName, pullRequest).Do()
		if err != nil {
			return err
		}
		if len(pullResponse.ReceivedMessages) > 0 {
			break
		}
		if q.closed {
			return ErrClosed
		}
		if started.Add(timeout).After(time.Now()) {
			return ErrTimeout
		}
	}
	for _, receivedMessage := range pullResponse.ReceivedMessages {
		data, err := base64.StdEncoding.DecodeString(receivedMessage.Message.Data)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, body)
		ackRequest := &pubsub.AcknowledgeRequest{
			AckIds: []string{receivedMessage.AckId},
		}
		if _, err = q.svc.Projects.Subscriptions.Acknowledge(subName, ackRequest).Do(); err != nil {
			logrus.Warnf("Unable to acknoledge message - %v", err)
			return err
		}
	}
	return nil
}

func (q *queuePubSub) PushConf(t string, c *domain.Configuration) error {
	confMessage := ConfigurationMessage{Team: t, Configuration: *c}
	return q.push(conf.Options.G.ConfName, &confMessage)
}

func (q *queuePubSub) PopConf(timeout time.Duration) (string, *domain.Configuration, error) {
	var msg ConfigurationMessage
	err := q.pop(conf.Options.G.ConfName+"-"+q.host, timeout, &msg)
	if err != nil {
		return "", nil, err
	}
	return msg.Team, &msg.Configuration, nil
}

func (q *queuePubSub) PushWork(data *domain.WorkRequest) error {
	return q.push(conf.Options.G.WorkName, data)
}

func (q *queuePubSub) PopWork(timeout time.Duration) (*domain.WorkRequest, error) {
	data := &domain.WorkRequest{}
	err := q.pop(conf.Options.G.WorkName, timeout, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (q *queuePubSub) PushWorkReply(replyQueue string, reply *domain.WorkReply) error {
	return q.push(replyQueue, reply)
}

func (q *queuePubSub) PopWorkReply(replyQueue string, timeout time.Duration) (*domain.WorkReply, error) {
	workReply := &domain.WorkReply{}
	err := q.pop(replyQueue, timeout, workReply)
	if err != nil {
		return nil, err
	}
	return workReply, nil
}

func (q *queuePubSub) Close() error {
	q.closed = true
	return nil
}
