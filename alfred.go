package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/demisto/alfred/util"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/web"
)

var (
	confFile = flag.String("conf", "conf.json", "Path to configuration file in JSON format")
	logLevel = flag.String("loglevel", "info", "Specify the log level for output (debug/info/warn/error/fatal/panic) - default is info")
	logFile  = flag.String("logfile", "", "The log file location")
)

type closer interface {
	Close() error
}

type botCloser struct {
	*bot.Bot
}

func (b *botCloser) Close() error {
	b.Stop()
	return nil
}

func run(signalCh chan os.Signal) {
	var closers []closer
	// If we are on DEV, let's use embedded DB. On test and prod we will use MySQL
	r, err := repo.NewMySQL()
	if err != nil {
		logrus.Fatal(err)
	}
	closers = append(closers, r)

	// Create the queue for the various message exchanges
	q, err := queue.New(r)
	if err != nil {
		logrus.Fatal(err)
	}
	closers = append(closers, q)

	serviceChannel := make(chan bool)
	if conf.Options.Web {
		b, err := bot.New(r, q)
		if err != nil {
			logrus.Fatal(err)
		}
		go func() {
			err = b.Start()
			if err != nil {
				logrus.Fatal(err)
			}
			serviceChannel <- true
		}()
		closers = append(closers, &botCloser{b})
		appC := web.NewContext(r, q, b)
		router := web.New(appC)
		go func() {
			router.Serve()
			serviceChannel <- true
		}()
	}

	if conf.Options.Worker {
		worker, err := bot.NewWorker(q)
		if err != nil {
			logrus.Fatal(err)
		}
		go func() {
			worker.Start()
			serviceChannel <- true
		}()
	}

	// Block until one of the signals above is received
	select {
	case <-signalCh:
		logrus.Infoln("Signal received, initializing clean shutdown...")
	case <-serviceChannel:
		logrus.Infoln("A service went down, shutting down...")
	}
	closeChannel := make(chan bool)
	go func() {
		for i := range closers {
			closers[i].Close()
		}
		closeChannel <- true
	}()
	// Block again until another signal is received, a shutdown timeout elapses,
	// or the Command is gracefully closed
	logrus.Infoln("Waiting for clean shutdown...")
	select {
	case <-signalCh:
		logrus.Infoln("Second signal received, initializing hard shutdown")
	case <-time.After(time.Second * 30):
		logrus.Infoln("Time limit reached, initializing hard shutdown")
	case <-closeChannel:
	}
}

func main() {
	flag.Parse()
	util.InitLog(*logFile, *logLevel, *logFile == "")
	defer conf.LogWriter.Close()
	err := conf.Load(*confFile, true)
	if err != nil {
		logrus.Fatal(err)
	}

	// Handle OS signals to gracefully shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	logrus.Infoln("Listening to OS signals")

	run(signalCh)
	logrus.Infoln("Server shutdown completed")
}
