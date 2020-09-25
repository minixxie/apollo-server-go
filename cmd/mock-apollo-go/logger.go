package main

import (
	"github.com/lalamove/nui/nlogger"
	"github.com/sirupsen/logrus"
)

type defaultLogger struct {
	l *logrus.Logger
}

// Debug will print the message in debug level
func (dl *defaultLogger) Debug(msg string) {
	dl.l.Debug(msg)
}

// Info will print the message in info level
func (dl *defaultLogger) Info(msg string) {
	dl.l.Info(msg)
}

// Warn will print the message in warning level
func (dl *defaultLogger) Warn(msg string) {
	dl.l.Warn(msg)
}

// Error will print the message in error level
func (dl *defaultLogger) Error(msg string) {
	dl.l.Error(msg)
}

// Fatal will print the message in fatal level and kill the main process
func (dl *defaultLogger) Fatal(msg string) {
	dl.l.Fatal(msg)
}

func newLogger(level logrus.Level) nlogger.Structured {
	var log = logrus.New()
	log.SetLevel(level)
	return nlogger.ToStructured(&defaultLogger{log})
}
