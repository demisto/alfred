package util

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/demisto/alfred/conf"

	"github.com/Sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var maxLineSize int
var output *lumberjack.Logger

type captainHook struct {
}

func (*captainHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (*captainHook) Fire(entry *logrus.Entry) error {
	skip := 6
	ok := true
	var file string
	var line int
	for ok {
		_, file, line, ok = runtime.Caller(skip)
		if strings.Contains(file, "logrus") {
			skip++
		} else {
			entry.Data["source"] = fmt.Sprintf("%s:%d", file, line)
			return nil
		}
	}
	return nil
}

type simpleFormatter struct {
}

func (*simpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	buff := new(bytes.Buffer)
	buff.Grow(512)
	buff.WriteString(entry.Time.Format("2006-01-02 15:04:05.9999 "))
	msg := entry.Message
	if len(msg) > maxLineSize {
		msg = fmt.Sprintf("%s...\nNOTE, too much data to log, message was truncated.", msg[:maxLineSize])
	}
	fmt.Fprintf(buff, "%s %s ", entry.Level, msg)
	for name, field := range entry.Data {
		fmt.Fprintf(buff, "(%s: %v)", name, field)
	}
	buff.WriteString(" \n")
	if runtime.GOOS == "windows" {
		buff.WriteString("\r")
	}
	return buff.Bytes(), nil
}

// InitLog - init the log for server
func InitLog(fileloc string, logLevel string, stdout bool) {
	// limit the size of the message to log
	maxLineSize = 100000

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		fmt.Printf("Invalid log level value provided; %s, using Info", logLevel)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.AddHook(new(captainHook))
	logrus.SetFormatter(new(simpleFormatter))

	if stdout {
		logrus.SetOutput(os.Stderr)
	} else {
		SetOutput(fileloc)
	}

	conf.LogWriter = logrus.StandardLogger().Writer()
}

//SetOutput ...
func SetOutput(fileloc string) {
	if output == nil {
		output = &lumberjack.Logger{}
		logrus.SetOutput(output)
	}
	output.Filename = fileloc
	output.MaxSize = 10
	output.MaxBackups = 3
	output.MaxAge = 0
}
