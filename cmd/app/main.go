package main

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"work-wechat/pkg/work"
)

var chat = work.WorkWeixin{}

type ChatConfig struct {
	Corpid  string `yaml:"corpid"`
	Secret  string `yaml:"secret"`
	AgentId int    `yaml:"agentId"`
}
type EnvConfig struct {
	Port       string      `yaml:"port"`
	ChatConfig *ChatConfig `yaml:"chat"`
}

//centOS build:    GOARCH=amd64  GOOS=linux  go build -o  bin/work-wechat
func main() {

	conf := loadEnv()

	http.HandleFunc("/", indexHandler)
	port := conf.Port
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	//聊天
	//chat.Init("ww40f46d8d9dc94ed2", "B8WViyiyVmFhH-dq4wU8p1f9GZgt0cb3l7ksNYCpW-o", 1000009)
	chat.Init(conf.ChatConfig.Corpid, conf.ChatConfig.Secret, conf.ChatConfig.AgentId)
	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func loadEnv() EnvConfig {
	//
	//自己load
	file, err := os.Open("config.yaml") // For cofig access.
	if err != nil {
		log.Panicln("please config config.yaml")
	}
	defer file.Close()

	buffer, err := ioutil.ReadAll(file)
	var currentConfig EnvConfig
	err = yaml.Unmarshal(buffer, &currentConfig)
	log.Println("load config：", string(buffer))
	if err != nil {
		log.Panicln("Unmarshal config err", err, string(buffer))
	}
	return currentConfig
}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/sendGroupMsg":
		toUser := strings.Split(r.FormValue("toUser"), ",")
		title := r.FormValue("title")
		content, _ := ioutil.ReadAll(r.Body)
		_, err := w.Write(bytes.NewBufferString(chat.SendGroupText(toUser, title, string(content))).Bytes())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	default:
		http.NotFound(w, r)
		return
	}

}
