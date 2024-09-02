package logger

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	logf, err := rotatelogs.New(
		"logs/%Y%m%d.log",
		rotatelogs.WithLinkName("logs/log"),
		rotatelogs.WithMaxAge(-1),
		rotatelogs.WithRotationCount(90),
	)
	if err != nil {
		log.Printf("failed to create rotatelogs: %s", err)
		return
	}

	log.SetOutput(logf)
	log.SetLevel(log.InfoLevel | log.WarnLevel | log.ErrorLevel)
}

func I(func_name string, args ...interface{}) {
	log.WithFields(log.Fields{
		"func": func_name,
	}).Info(args...)
}

func E(func_name string, args ...interface{}) {
	log.WithFields(log.Fields{
		"func": func_name,
	}).Error(args...)
}
