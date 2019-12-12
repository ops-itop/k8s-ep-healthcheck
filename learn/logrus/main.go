package main

import (
	blog "github.com/astaxie/beego/logs"
	log "github.com/sirupsen/logrus"
	"os"
	//"reflect"
)

var level = os.Getenv("LOGLEVEL")

func init() {
	log.SetOutput(os.Stdout)
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Panic("Log Level not illegal.You should use trace,debug,info,warn,warning,error,fatal,panic")
	}
	log.SetLevel(logLevel)
}

func main() {
	log.Info("Log Level is " + level)
	log.WithFields(log.Fields{
		"animal": "walrus",
		"size":   10,
	}).Info("A group of walrus emerges from the ocean")

	log.WithFields(log.Fields{
		"omg":    true,
		"number": 122,
	}).Warn("The group's number increased tremendously!")

	log.WithFields(log.Fields{
		"omg":    true,
		"number": 100,
	}).Error("The ice breaks!")

	// A common pattern is to re-use fields between logging statements by re-using
	// the logrus.Entry returned from WithFields()
	contextLogger := log.WithFields(log.Fields{
		"common": "this is a common field",
		"other":  "I also should be logged always",
	})

	contextLogger.Info("I'll be logged with common and other field")
	contextLogger.Info("Me too")

	// test beego logs
	beegolog := blog.NewLogger(10000)
	beegolog.SetLogger("console", `{"level":4}`)
	beegolog.Trace("trace %s %s", "param1", "param2")
	beegolog.Debug("debug")
	beegolog.Info("info")
	beegolog.Warn("warning")
	beegolog.Error("error")
	beegolog.Critical("critical")
}
