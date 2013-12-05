package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type MoeConfig struct {
	ServerRoot string
	LogFile string
	Version string

	UserDbInfo DbInfo
	LogDbInfo DbInfo

	ImageDir string

	BlockedIPs map[string]bool
}

type DbInfo struct {
	DBUrl string
	DBName string
	TableName string
}

var config MoeConfig

func parseConf() {
	confStr, err := ioutil.ReadFile("/etc/moechatconf.json")
	if err != nil {
		fmt.Println("Error reading config file:", err)
		panic(err)
	}

	err = json.Unmarshal(confStr, &config)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		panic(err)
	}
}
