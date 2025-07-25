package main

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"

	"github.com/dumacp/go-logs/pkg/logs"
)

func newLog(logger *logs.Logger, prefix string, flags int, priority int) error {

	logg, err := syslog.NewLogger(syslog.Priority(priority), flags)
	if err != nil {
		return err
	}
	logger.SetLogError(logg)
	return nil
}

func newMultiLog(logger *logs.Logger, prefix string, flags int, priority int) error {

	logg, err := syslog.NewLogger(syslog.Priority(priority), flags)
	if err != nil {
		return err
	}

	logStd := log.New(os.Stderr, fmt.Sprintf("[ %s ]", prefix), flags)

	// Logger combinado
	multiWriter := io.MultiWriter(logStd.Writer(), logg.Writer())
	combinedLogger := log.New(multiWriter, "", 0)

	logger.SetLogError(combinedLogger)
	return nil
}

func initLogs(debug, logStd bool) {
	if logStd {
		return
	}
	newMultiLog(logs.LogWarn, "[ warn ] ", 0, 4)
	newLog(logs.LogInfo, "[ info ] ", 0, 6)
	newLog(logs.LogBuild, "[ build ] ", 0, 7)
	newMultiLog(logs.LogError, "[ error ] ", 0, 3)
	if !debug {
		logs.LogBuild.Disable()
	}
}
