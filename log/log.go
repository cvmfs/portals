package log

import (
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.TraceLevel)
}

func Log() *log.Entry {
	return log.WithFields(log.Fields{})
}

func LogE(err error) *log.Entry {
	return log.WithFields(log.Fields{"error": err})
}

func Decorate(fields map[string]string) func(*log.Entry) *log.Entry {
	return func(l *log.Entry) *log.Entry {
		for key, value := range fields {
			l = l.WithField(key, value)
		}
		return l
	}
}
