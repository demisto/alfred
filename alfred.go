package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/web"
)

func main() {
	confFile := flag.String("conf", "conf.json", "Path to configuration file in JSON format")
	public := flag.String("public", "site", "Path to the web directory containing the public files")
	logLevel := flag.String("loglevel", "info", "Specify the log level for output (debug/info/warn/error/fatal/panic) - default is info")
	logFile := flag.String("logfile", "", "The log file location")
	flag.Parse()
	err := conf.Load(*confFile, true)
	if err != nil {
		logrus.Fatal(err)
	}
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(level)
	logf := os.Stderr
	if *logFile != "" {
		logf, err = os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			logrus.Fatal(err)
		}
		defer logf.Close()
	}
	logrus.SetOutput(logf)
	conf.LogWriter = logrus.StandardLogger().Writer()
	defer conf.LogWriter.Close()
	r, err := repo.New()
	if err != nil {
		logrus.Fatal(err)
	}
	defer r.Close()
	b, err := bot.New(r)
	if err != nil {
		logrus.Fatal(err)
	}
	err = b.Start()
	if err != nil {
		logrus.Fatal(err)
	}
	defer b.Stop()
	appC := web.NewContext(r, b)
	router := web.New(appC, *public)
	logrus.Fatal(http.ListenAndServe(":7070", router))
}
