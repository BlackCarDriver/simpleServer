package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"../config"
	"../model"
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
)

type paramsType struct {
	Nick string `json:"nick"`
	Msg  string `json:"msg"`
}

type responseType struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

type CmdType struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

const myName = "BlackCarDriver"

var sendEmail = true // 控制是否发送邮件的开关之一

// CallDriver应用请求全部经过这里
func CallDriverHandler(w http.ResponseWriter, r *http.Request) {
	logs.Debug("callDriver url=%v", r.URL)
	url := strings.Trim(fmt.Sprintf("%s", r.URL.Path), "/")
	switch url {
	case "callDriver":
		CallDriverHtml(w, r)
	case "callDriver/sendMessage":
		callDriverReceiveMsg(w, r)
	case "callDriver/getHistory":
		callDriverGetChatHistroy(w, r)
	case "callDriver/boss":
		callDriverBossHtml(w, r)
	case "callDriver/boss/getAll":
		callDriverGetAllChat(w, r)
	case "callDriver/boss/reply":
		callDriverBossReply(w, r)
	case "callDriver/boss/control":
		callDriverSetMail(w, r)
	default:
		DefaultHandler(w, r)
	}
}

func CallDriverHtml(w http.ResponseWriter, r *http.Request) {
	RecordRequest(r, "🚓")
	http.ServeFile(w, r, "./source/callDriverIndex.html")
}

// 接受来自callDriver应用的消息，保存到数据库并发送通知邮件
func callDriverReceiveMsg(w http.ResponseWriter, r *http.Request) {
	var req paramsType
	var resp responseType
	var err error
	ip, _ := tb.GetIpAndPort(r)
	for loop := true; loop; loop = false {
		// 解析参数
		err = tb.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Error("Parse request fail: err=%v request=%v", r)
			break
		}
		req.Nick = strings.TrimSpace(req.Nick)
		req.Msg = strings.TrimSpace(req.Msg)
		logs.Info("Receive a message: %v", req)

		// 参数检查
		if len(req.Nick) < 2 || len(req.Msg) < 1 {
			resp.Status = -1
			resp.Msg = "nick or message is too short"
			break
		}
		if req.Nick == myName {
			resp.Status = -1
			resp.Msg = "Sorry, you can't use it nick..."
			break
		}

		// 保存聊天记录
		err = model.InsertCallDriverMessage(req.Nick, myName, req.Msg, ip)
		if err != nil {
			logs.Error("Save to history fail: %v", err)
			break
		}
		// 发送邮箱通知
		if config.ServerConfig.IsTest || !sendEmail {
			logs.Info("Skip send email: isTest=%s  sendEmail=%v",
				config.ServerConfig.IsTest, sendEmail)
			break
		}
		err = tb.SendToMySelf(req.Nick, req.Msg)
		if err != nil {
			logs.Error("Send mail fail: err=%v req=%+v", err, req)
			break
		}
		logs.Info("Send mail success")
	}

	logs.Info("receive message result: req=%v err=%v", req, err)

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	} else {
		resp.Status = 0
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", bytes)
}

// 查询聊天记录
func callDriverGetChatHistroy(w http.ResponseWriter, r *http.Request) {
	type reqType struct {
		Nick string `json:"nick"`
	}
	type respType struct {
		Status   int    `json:"status"` // 当有新消息时,status=1
		Msg      string `json:"msg"`
		ChatHtml string `json:"chatHtml"`
	}
	var req reqType
	var resp respType
	var history []model.CallDriverChat
	var err error

	for loop := true; loop; loop = false {
		// 解析参数
		err = tb.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Error("Parse request fail: err=%v request=%v", r)
			break
		}
		req.Nick = strings.TrimSpace(req.Nick)
		logs.Info("get history request: %v", req)

		// 参数检查
		if len(req.Nick) < 2 {
			resp.Status = -1
			resp.Msg = "nick or message is too short"
			break
		}
		if req.Nick == myName {
			resp.Status = -1
			resp.Msg = "Sorry, you can't use it nick..."
			break
		}

		// 查询记录
		history, err = model.FindCallDriverMessage(req.Nick, 10)
		if err != nil || history == nil {
			logs.Error("get history fail: err=%v len=%v", err, len(history))
			break
		}
		// 生成聊条记录html代码
		divHtml := ""
		resp.Status = 0
		for idx := len(history) - 1; idx >= 0; idx-- {
			v := history[idx]
			if v.Status == 0 && v.From != req.Nick {
				logs.Info("new message remark")
				resp.Status = 1
			}
			tstr := time.Unix(v.TimeStamp, 0).Format("01-02 15:04")
			className := "msgNick1"
			if v.From != req.Nick {
				className = "msgNick2"
			}
			divHtml += fmt.Sprintf(`<p class="msgTime">--------------------[  %s ]--------------------</p>
			<p class="%s">%s: <span class="msgText">%s</span></p>
			`, tstr, className, v.From, v.Message)
		}
		resp.Msg = divHtml
	}

	logs.Info("get histroy result: req=%v err=%v", req, err)

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", bytes)
}

//================ Boss专用，需要IP白名单权限 =========================

// Boss页面
func callDriverBossHtml(w http.ResponseWriter, r *http.Request) {
	if !config.ServerConfig.IsTest && !tb.IsInWhiteList(r) {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	http.ServeFile(w, r, "./source/callDriverBoss.html")
}

// Boss回复消息
func callDriverBossReply(w http.ResponseWriter, r *http.Request) {
	if !config.ServerConfig.IsTest && !tb.IsInWhiteList(r) {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	var req paramsType
	var resp responseType
	var err error
	ip, _ := tb.GetIpAndPort(r)
	for loop := true; loop; loop = false {
		// 解析参数
		err = tb.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Error("Parse request fail: err=%v request=%v", r)
			break
		}
		req.Nick = strings.TrimSpace(req.Nick)
		req.Msg = strings.TrimSpace(req.Msg)

		// 参数检查
		if len(req.Nick) < 2 || len(req.Msg) < 1 {
			resp.Status = -1
			resp.Msg = "nick or message is too short"
		}

		// 保存聊天记录
		err = model.InsertCallDriverMessage(myName, req.Nick, req.Msg, ip)
		if err != nil {
			logs.Error("Save to history fail: %v", err)
			break
		}
		logs.Info("reply success")
	}

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	} else {
		resp.Status = 0
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", bytes)
}

// Boss查看消息
func callDriverGetAllChat(w http.ResponseWriter, r *http.Request) {
	if !config.ServerConfig.IsTest && !tb.IsInWhiteList(r) {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	type respType struct {
		Status   int    `json:"status"`
		Msg      string `json:"msg"`
		ChatHtml string `json:"chatHtml"`
	}
	var resp respType
	var history []model.CallDriverChat
	var err error

	for loop := true; loop; loop = false {
		// 查询记录
		history, err = model.FindAllCallDriverMessage()
		if err != nil || history == nil {
			logs.Error("get history fail: err=%v len=%v", err, len(history))
			break
		}
		// 生成聊条记录html代码
		divHtml := ""
		resp.Status = 0
		for idx := len(history) - 1; idx >= 0; idx-- {
			v := history[idx]
			tstr := time.Unix(v.TimeStamp, 0).Format("01-02 15:04")
			className := "msgNick1"
			if v.From != myName {
				className = "msgNick2"
			}
			divHtml += fmt.Sprintf(`<p class="msgTime">--------------------[  %s ]--------------------</p>
			<p class="%s">%s: <span class="msgText">%s</span></p>
			`, tstr, className, v.From, v.Message)
		}
		resp.Msg = divHtml
	}

	logs.Info("get histroy result: err=%v", err)

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", bytes)
}

// 其他相关控制
func callDriverSetMail(w http.ResponseWriter, r *http.Request) {
	if !config.ServerConfig.IsTest && !tb.IsInWhiteList(r) {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	var req CmdType
	var err error
	var resp responseType
	for loop := true; loop; loop = false {
		// 解析参数
		err = tb.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Error("Parse request fail: err=%v request=%v", r)
			break
		}
		switch req.Key {
		case "sendMail":
			var sendOrNot bool
			sendOrNot, err = strconv.ParseBool(req.Value)
			if err == nil {
				logs.Info("Email config change: sendEmail=%v", sendOrNot)
				sendEmail = sendOrNot
			}

		default:
			err = fmt.Errorf("unexpect key: %s", req.Key)
		}
	}
	logs.Info("contral result: req=%v err=%v", req, err)

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	} else {
		resp.Status = 0
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", bytes)
}
