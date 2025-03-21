package wechat

// ref. https://studygolang.com/articles/8401

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	sendUrl  = "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="
	tokenUrl = "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid="
)

var requestError = errors.New("request error,check url or network")

type AccessToken struct {
	Access_token string `json:"access_token"`
	Expires_in   int64  `json:"expires_in"`
	Next_due     int64
}

type WechatMsg struct {
	Touser  string            `json:"touser"`
	Msgtype string            `json:"msgtype"`
	Agentid int               `json:"agentid"`
	Text    map[string]string `json:"text"`
}

type send_msg_error struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

func SendMsg(Access_token string, msgbody []byte) error {
	body := bytes.NewBuffer(msgbody)
	resp, err := http.Post(sendUrl+Access_token, "application/json", body)
	if resp.StatusCode != 200 {
		return requestError
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var e send_msg_error

	err = json.Unmarshal(buf, &e)

	if err != nil {
		return err
	}

	if e.Errcode != 0 && e.Errmsg != "ok" {
		return errors.New(string(buf))
	}
	return nil
}

func GetToken(corpid, corpsecret string) (at AccessToken, err error) {
	resp, err := http.Get(tokenUrl + corpid + "&corpsecret=" + corpsecret)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = requestError
		return
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(buf, &at)
	if at.Access_token == "" {
		err = errors.New("corpid or corpsecret error.")
	}

	at.Next_due = time.Now().Unix() + at.Expires_in
	return
}

func UpdateToken(token *AccessToken, corpid string, corpsecret string) (err error) {
	if token.Access_token == "" {
		*token, err = GetToken(corpid, corpsecret)
		if err != nil {
			return err
		}
	}

	if token.Next_due <= time.Now().Unix() {
		*token, err = GetToken(corpid, corpsecret)
		if err != nil {
			return err
		}
	}
	return nil
}
