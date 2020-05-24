package logrusconfig

import (
	"flag"

	prefixed "github.com/BertoldVdb/logrus-prefixed-formatter"
	"github.com/sirupsen/logrus"
)

var loglevel *int

func InitParam() {
	loglevel = flag.Int("loglevel", int(logrus.DebugLevel), "The loglevel to use. Valid values are from 0 to 6. Higher values output more information")
}

func GetLogger(level logrus.Level) *logrus.Entry {
	logrus.ErrorKey = "$error"
	logger := logrus.New()
	if loglevel == nil {
		logger.SetLevel(level)
	} else {
		logger.SetLevel(logrus.Level(*loglevel))
	}
	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	customFormatter.PrefixPadding = 20
	customFormatter.SpacePadding = 50
	logger.SetFormatter(customFormatter)
	return logrus.NewEntry(logger)
}
