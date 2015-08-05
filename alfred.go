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
	if conf.Options.DB.Username == "" {
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

	if conf.Options.Bot {
		b, err := bot.New(r, q)
		if err != nil {
			logrus.Fatal(err)
		}
		go func() {
			err = b.Start()
			if err != nil {
				logrus.Fatal(err)
			}
		}()
		defer b.Stop()
	}

	if conf.Options.Dedup {
		dd := dedup.New(q)
		go dd.Start()
	}

	if conf.Options.Worker {
		worker, err := bot.NewWorker(r, q)
		if err != nil {
			logrus.Fatal(err)
		}
		go worker.Start()
	}

	if conf.Options.Web {
		appC := web.NewContext(r, q)
		router := web.New(appC)
		if conf.Options.SSL.Cert != "" {
			logrus.Fatal(http.ListenAndServeTLS(conf.Options.Address, conf.Options.SSL.Cert, conf.Options.SSL.Key, router))
		} else {
			logrus.Fatal(http.ListenAndServe(conf.Options.Address, router))
		}
	}
}
