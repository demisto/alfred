package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
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
}
