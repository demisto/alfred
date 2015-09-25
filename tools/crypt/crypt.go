package main

import (
	"flag"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/util"
)

var (
	confFile = flag.String("conf", "conf.json", "Path to configuration file in JSON format")
	logLevel = flag.String("loglevel", "info", "Specify the log level for output (debug/info/warn/error/fatal/panic) - default is info")
	logFile  = flag.String("logfile", "", "The log file location")
	action   = flag.String("action", "decrypt", "Action to perform on the data - encrypt/decrypt")
	data     = flag.String("data", "", "The data to perform the action on")
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	err := conf.Load(*confFile, false)
	if err != nil {
		logrus.Fatal(err)
	}
	if *data == "" {
		logrus.Fatal("Please specify the data")
	}
	if *action != "decrypt" && *action != "encrypt" {
		logrus.Fatal("Invalid action specified")
	}
	if *action == "decrypt" {
		clear, err := util.Decrypt(*data, conf.Options.Security.DBKey)
		check(err)
		fmt.Println(clear)
	} else {
		cipher, err := util.Encrypt(*data, conf.Options.Security.DBKey)
		check(err)
		fmt.Println(cipher)
	}
}
