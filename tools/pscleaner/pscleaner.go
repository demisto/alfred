package main

import (
	"encoding/json"

	"github.com/demisto/alfred/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/pubsub/v1"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	jsonCreds, err := json.Marshal(conf.Options.G.Credentials)
	check(err)
	config, err := google.JWTConfigFromJSON(jsonCreds, pubsub.PubsubScope)
	check(err)
	client := config.Client(oauth2.NoContext)
	svc, err := pubsub.New(client)
	check(err)
	list, err := svc.Projects.Subscriptions.List(conf.Options.G.Project).Do()
	check(err)
	for i := range list.Subscriptions {
		svc.Projects.Subscriptions.Delete(list.Subscriptions[i].Name)
	}
	topics, err := svc.Projects.Topics.List(conf.Options.G.Project).Do()
	check(err)
	for i := range topics.Topics {
		svc.Projects.Topics.Delete(topics.Topics[i].Name)
	}
}
