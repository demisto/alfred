package main

import (
	"encoding/json"
	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/pubsub/v1"
)

var (
	confFile = flag.String("conf", "conf.json", "Path to configuration file in JSON format")
	logLevel = flag.String("loglevel", "info", "Specify the log level for output (debug/info/warn/error/fatal/panic) - default is info")
	logFile  = flag.String("logfile", "", "The log file location")
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	err := conf.Load(*confFile, true)
	if err != nil {
		logrus.Fatal(err)
	}
	jsonCreds, err := json.Marshal(conf.Options.G.Credentials)
	check(err)
	config, err := google.JWTConfigFromJSON(jsonCreds, pubsub.PubsubScope)
	check(err)
	client := config.Client(oauth2.NoContext)
	svc, err := pubsub.New(client)
	check(err)
	list, err := svc.Projects.Subscriptions.List("projects/" + conf.Options.G.Project).Do()
	check(err)
	for i := range list.Subscriptions {
		logrus.Infof("Deleting subscription - %s", list.Subscriptions[i].Name)
		_, err := svc.Projects.Subscriptions.Delete(list.Subscriptions[i].Name).Do()
		check(err)
	}
	topics, err := svc.Projects.Topics.List("projects/" + conf.Options.G.Project).Do()
	check(err)
	for i := range topics.Topics {
		logrus.Infof("Deleting topic - %s", topics.Topics[i].Name)
		_, err := svc.Projects.Topics.Delete(topics.Topics[i].Name).Do()
		check(err)
	}
}
