package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

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
	var r repo.Repo
	var err error
	if conf.Options.DB.Username == "" {
		r, err = repo.New()
	} else {
		r, err = repo.NewMySQL()
	}
	if err != nil {
		logrus.Fatal(err)
	}
	closers = append(closers, r)

	// Create the queue for the various message exchanges
	q, err := queue.New()
	if err != nil {
		logrus.Fatal(err)
	}
	closers = append(closers, q)

	serviceChannel := make(chan bool)
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
			serviceChannel <- true
		}()
		closers = append(closers, &botCloser{b})
	}

	if conf.Options.Dedup {
		dd := bot.NewDedup(q)
		go func() {
			dd.Start()
			serviceChannel <- true
		}()
	}

	if conf.Options.Worker {
		worker, err := bot.NewWorker(r, q)
		if err != nil {
			logrus.Fatal(err)
		}
		go func() {
			worker.Start()
			serviceChannel <- true
		}()
	}

	if conf.Options.Web {
		appC := web.NewContext(r, q)
		router := web.New(appC)
		go func() {
			router.Serve()
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
	// Handle OS signals to gracefully shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	logrus.Infoln("Listening to OS signals")

	run(signalCh)
	logrus.Infoln("Server shutdown completed")
}
