package main

import (
	"bytes"
	"encoding/json"
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

//centOS build:    GOARCH=amd64  GOOS=linux  go build -o  bin/work-wechat
func main() {
	http.HandleFunc("/", indexHandler)
	var conf = loadEnv()
	port := conf.Port
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	//聊天

	chat.Init(conf.ChatConfig)
	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func loadEnv() work.EnvConfig {
	//
	//自己load

	file, err := os.Open("config.yaml") // For cofig access.
	if err != nil {
		log.Panicln("please config config.yaml")
	}
	defer file.Close()

	buffer, err := ioutil.ReadAll(file)
	currentConfig := work.EnvConfig{}
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
	case "/send/template/card":
		var temp = work.TemplateCard{}
		content, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(content, &temp)
		log.Println(temp)
		chat.SendTemplateMsg(temp)
	case "/callback/wechat":
		msg_signature := r.FormValue("msg_signature")
		timestamp := r.FormValue("timestamp")
		nonce := r.FormValue("nonce")
		echostr := r.FormValue("echostr")

		log.Println(msg_signature)
		log.Println(timestamp)
		log.Println(nonce)
		log.Println(echostr)
		//第一次callback 认证绑定 时候使用
		//result, e := chat.VerityCallback(msg_signature, timestamp, nonce, echostr)
		content, _ := ioutil.ReadAll(r.Body)
		callback, err := chat.Callback(msg_signature, timestamp, nonce, content)
		log.Printf("%s\t%s", callback, err)
		//w.Write()

	default:
		http.NotFound(w, r)
		return
	}

}
