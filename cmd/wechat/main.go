package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ops-itop/k8s-ep-healthcheck/pkg/notify/wechat"
)

func main() {
	touser := flag.String("t", "@all", "接受消息的用户名")
	agentid := flag.Int("i", 0, "指定agentid")
	content := flag.String("c", "Hello Wechat", "发送的内容")
	corpid := flag.String("p", "", "corpid")
	corpsecret := flag.String("s", "", "corpsecret")

	flag.Parse()

	if *corpid == "" || *corpsecret == "" {
		flag.Usage()
		return
	}

	token, err := wechat.GetToken(*corpid, *corpsecret)
	if err != nil {
		fmt.Println(err)
		return
	}

	msg := wechat.WechatMsg{Touser: *touser, Msgtype: "text", Agentid: *agentid, Text: map[string]string{"content": *content}}

	buf, err := json.Marshal(msg)
	if err != nil {
		return
	}
	err = wechat.SendMsg(token.Access_token, buf)
	if err != nil {
		fmt.Println(err)
		return
	}
}
