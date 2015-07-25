package main

import (
	"flag"
	"net/http"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/dedup"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/web"
)

func main() {
	confFile := flag.String("conf", "conf.json", "Path to configuration file in JSON format")
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

	// Let's use all the logical CPUs
	runtime.GOMAXPROCS(runtime.NumCPU())

	// If we are on DEV, let's use embedded DB. On test and prod we will use MySQL
	var r repo.Repo
	if conf.IsDev() || conf.Options.DB.Username == "" {
		r, err = repo.New()
	} else {
		r, err = repo.NewMySQL()
	}
	if err != nil {
		logrus.Fatal(err)
	}
	defer r.Close()

	// Create the queue for the various message exchanges
	q := queue.New()
	defer q.Close()

	b, err := bot.New(r, q)
	if err != nil {
		logrus.Fatal(err)
	}
	err = b.Start()
	if err != nil {
		logrus.Fatal(err)
	}
	defer b.Stop()
	// If we are on dev environment, start the dedup and work process
	if conf.IsDev() {
		dd := dedup.New(q)
		go dd.Start()
		worker, err := bot.NewWorker(r, q)
		if err != nil {
			logrus.Fatal(err)
		}
		go worker.Start()
	}
	appC := web.NewContext(r, q)
	router := web.New(appC)
	if conf.IsDev() {
		logrus.Fatal(http.ListenAndServe(":7070", router))
	} else {
		logrus.Fatal(http.ListenAndServe(":7070", router))
	}
}
