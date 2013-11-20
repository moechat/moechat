package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"os"
)

var LogFile = "/var/log/moechat.log"

var mongoURL = "localhost"
var messageLogDB = "message_log_db"

type messageLog struct {
	Id bson.ObjectId
	Msg *interface{}
	Room *ChatRoom
}

var msgLogChan = make(chan messageLog)

func messageLogRun(msgLogDB *mgo.Database, logger chan messageLog) {
	for {
		select {
		case _ = <-logger:
		default: log.Fatal("Failed to read from logger!")
		}
	}
}

func initLog() {
	logfile, err := os.OpenFile(LogFile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error opening logfile: %v", err)
	}

	log.SetOutput(logfile)

	mongoSession, err := mgo.Dial(mongoURL)
	if err != nil {
		log.Fatal("Error opening mongo DB: %v", err)
	}

	msgLogDB := mongoSession.DB(messageLogDB)
	go messageLogRun(msgLogDB, msgLogChan)
}
