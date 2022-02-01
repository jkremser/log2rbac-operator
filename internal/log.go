package internal

import (
	"github.com/fatih/color"
	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
)

type OwnLog struct {
	logr.Logger
}

const (
	err   = 3
	warn  = 2
	info  = 1
	debug = 0
)

func Wrap(log logr.Logger) OwnLog {
	return OwnLog{
		log,
	}
}

func Errorf(format string, args ...interface{}) {
	msgf(err, format, args...)
}

func msgf(severity int, format string, args ...interface{}) {
	switch severity {
	case info:
		log.Infof(format, args...)
	case debug:
		log.Debugf(format, args...)
	case warn:
		log.Warnf(format, args...)
	case err:
		color.Red("Prints text in cyan.")
		log.Errorf(format, args...)
	default:
		msgf(info, format, args...)
	}
}
