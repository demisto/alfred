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
	"github.com/demisto/slack"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	pubsub "google.golang.org/api/pubsub/v1"
)

const serviceCredentials = `{
  "private_key_id": "439de39dce3c772880dc3080049ee044343ff13b",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDM/JCMTRP0Hfx1\njKnS1lBpgkET9Z8KGvdtLGq9EN9DdjjYA5nj8eAq+aHO7Pv3sYUgz1LK8I34U/as\nSl6k+HttHMaCHYBrER4L3OjTW5dCWV8/ZsWT6ynWy1tNYkU2y5Ruxj1CAUsbpb+s\nZZoQglAywncQC5KGiINX29lkMhCX09KI1VLkwGxtLhdAYazZ/wSLfIAlROaeWjwf\nhQaE8xqc5MXC+F3j1U1cDYZuxvD4b9LW9pR9XSFgDKR+fWdWJcC6augUcszBeMiV\neG+Sv6CW89MYpNOpKP47it1Sox3wpSgbjZMWbqcEWJCeizi8EPMiAnKzjE2CEWzo\n8PqLtnTJAgMBAAECggEAdZbw6LsSljhZaalehicRC+V/pY6CRE7B3yvas0ipes6n\nvysZrYxENwLq0oRZ6nY4U2D7MpWaK3knCSDEeEherXITYfLAhyrTnKSGHzDsbVBN\ndlZjQv5lCuWvI44a/Fr+dCleXK3XQy3q7V9/aLcIgIXTvS2WSXyoM89XPsYFhMIk\nrQAEqxNTln0/793sJ5XuAyHCtUWOhaKqUkgdsRnJHMqFXnePXho0/gXcG/61zec0\nExPOLCRwawQ+cCapAsux1ZkY69th3Z5Tt5CtptIozkVEJuTlxcJHCiK647YvjFJU\nIKoD3oaYxVriCvCAEq0maXmq7TiTt/Ca/9rd/h1VsQKBgQDn/wfhOIZiHXnlPGZj\nYtB38iYbWvLxb2/3iygc7EoTs/ET8YnfPftu6SFQhhz9MK9DTcBUfCMJSxYPV27Y\nm7EZpHM+OseYAxdR+N1qegwmu1dnlrrJ+SGzuv4e+Nhj1KF+wlfrCGjsjeZXXimg\nf6sxStnc9H2C1zHIw7b6lYd3rQKBgQDiMh6nx509/T/xLqu4A1Gsx6SfEmcfcPHa\nl2Iir1wPB5jH72Z866oBQdBvH184ZQP+lr43/01xTzMbwid+FUEimY2VNhPJdFqv\niEpVar4BgxoLQxjVnI2fKujJSJ44CFKC8YPMNYcGBvAC4uXal60bC96jhcRIkDEt\ndEOtVFsFDQKBgQDFoIcB4Lj5U8rG8JD4EPEtfGXh37Qc36Ut5qkhGlhwOFUhfBzK\nw24wqP/sLJL9TD/AwbcZQTZHcGM2ZnDSrK5M/b3+QOxOHjP7bFiRn65CQEzQvaIY\n89U12hEoKSuMv1FjPgLPALcA7FBQFLK5OoiG0RCOHOfeUZrjP3XcOQzRcQKBgQDM\nGYtduxlgOOZ8eq9Jr/z/mXkqa9GPJjulERnk0DSR/znVlmf06jSRQ9COpFEoMsYC\n8AQdxQkc5+jm8C7wbr9COCnv7Ea4bXvyjVj9b/6YoLJcXSPIg6WqbG52SUcyqhfB\nvak+F0KJprLk99WNg3UYRYKULHxrOWiWaiUy/j3O9QKBgCoLX7E346IFbu0cTWUb\nfnhdTp6CcdgNIZGo1qluYif+XDte6J9/jsRbMq1zFx739APNWS+pR0f9dtkXx4iX\nH7ldp2sUFd09hEzGpX4zhuCv/xqZRRktirFHN7HxJaYpXWa140HMs7tlml6Oem7M\n7/jnzceX+qb6JyoDW9GcYNj5\n-----END PRIVATE KEY-----\n",
  "client_email": "977639562052-npb9jlbgvt7fbbbar1akla1tie65ti08@developer.gserviceaccount.com",
  "client_id": "977639562052-npb9jlbgvt7fbbbar1akla1tie65ti08.apps.googleusercontent.com",
  "type": "service_account"
}`

type queuePubSub struct {
	svc    *pubsub.Service
	closed bool
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
	config, err := google.JWTConfigFromJSON([]byte(serviceCredentials), pubsub.PubsubScope)
	if err != nil {
		return nil, err
	}
	client := config.Client(oauth2.NoContext)
	svc, err := pubsub.New(client)
	if err != nil {
		return nil, err
	}
	names := []string{conf.Options.G.ConfName, conf.Options.G.MessageName, conf.Options.G.WorkName}
	for _, n := range names {
		// Register the topics while ignoring already exists errors
		_, err := svc.Projects.Topics.Create(fullTopicName(conf.Options.G.Project, n), &pubsub.Topic{}).Do()
		if err != nil && !strings.Contains(err.Error(), "409") {
			return nil, err
		}
		sub := &pubsub.Subscription{Topic: fullTopicName(conf.Options.G.Project, n)}
		_, err = svc.Projects.Subscriptions.Create(fullSubName(conf.Options.G.Project, n), sub).Do()
		if err != nil && !strings.Contains(err.Error(), "409") {
			return nil, err
		}
	}
	return &queuePubSub{svc: svc}, nil
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
	pullResponse, err := q.svc.Projects.Subscriptions.Pull(subName, pullRequest).Do()
	if err != nil {
		return err
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

func (q *queuePubSub) PushConf(u *domain.User, c *domain.Configuration) error {
	confMessage := ConfigurationMessage{User: *u, Configuration: *c}
	return q.push(conf.Options.G.ConfName, &confMessage)
}

func (q *queuePubSub) PopConf(timeout time.Duration) (*domain.User, *domain.Configuration, error) {
	var msg ConfigurationMessage
	err := q.pop(conf.Options.G.ConfName, timeout, &msg)
	if err != nil {
		return nil, nil, err
	}
	return &msg.User, &msg.Configuration, nil
}

func (q *queuePubSub) PushMessage(data *slack.Message) error {
	return q.push(conf.Options.G.MessageName, data)
}

func (q *queuePubSub) PopMessage(timeout time.Duration) (*slack.Message, error) {
	msg := &slack.Message{}
	err := q.pop(conf.Options.G.MessageName, timeout, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (q *queuePubSub) PushWork(data *slack.Message) error {
	return q.push(conf.Options.G.WorkName, data)
}

func (q *queuePubSub) PopWork(timeout time.Duration) (*slack.Message, error) {
	msg := &slack.Message{}
	err := q.pop(conf.Options.G.WorkName, timeout, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (q *queuePubSub) Close() error {
	q.closed = true
	return nil
}
