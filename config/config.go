package config

import (
	"fmt"

	"github.com/tkanos/gonfig"
)

type Configuration struct {
	HttpServer struct {
		Host string
		Port string
	}
	HttpsServer struct {
		Host string
		Port string
	}
	Token   string
	Storage struct {
		Atlas            bool
		User             string
		Pass             string
		Host             string
		Port             string
		Database         string
		ConnectionString string
	}
	WebSocket struct {
		Token string
		Url   string
	}
	UsePermissionAuth bool
	SaiAuth           struct {
		Host string
		Port string
	}
	Duplication         bool   // if true, modification requests (save,update,upsert,remove)
	DuplicationURL      string // url, where duplicating request will be sent
	DuplicationInterval int    // interval to send duplicated requests
	DuplicateTimeout    int    // timeout to send duplicated request
	DuplicateMethod     string // method for saiService handler
}

func Load() Configuration {
	var config Configuration
	err := gonfig.GetConf("config.json", &config)

	if err != nil {
		fmt.Println("Configuration problem:", err)
		panic(err)
	}

	return config
}
