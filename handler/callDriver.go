package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"../model"
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
)

// CallDriver应用请求全部经过这里
func CallDriverHandler(w http.ResponseWriter, r *http.Request) {
	logs.Debug("callDriver url=%v", r.URL)
	url := fmt.Sprintf("%s", r.URL.Path)
	switch url {
	case "/callDriver/sendMessage":
		callDriverReceiveMsg(w, r)
	case "/callDriver/getHistory":
		callDriverGetChatHistroy(w, r)
	default:
		DefaultHandler(w, r)
	}
}

// 接受来自callDriver应用的消息，保存到数据库并发送通知邮件
func callDriverReceiveMsg(w http.ResponseWriter, r *http.Request) {
	type paramsType struct {
		Nick string `json:"nick"`
		Msg  string `json:"msg"`
	}
	type responseType struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	var req paramsType
	var resp responseType
	var err error

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
		// 保存聊天记录
		err = model.InsertCallDriverMessage(req.Nick, "BlackCarDriver", req.Msg)
		if err != nil {
			logs.Error("Save to history fail: %v", err)
			break
		}
		// 发送邮箱通知
		logs.Info("TODO:邮箱已发出！！！")
	}

	logs.Info("receive message result: req=%v err=%v", req, err)

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	} else {
		resp.Status = 0
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprint(w, bytes)
}

// 查询聊天记录
func callDriverGetChatHistroy(w http.ResponseWriter, r *http.Request) {
	type paramsType struct {
		Nick string `json:"nick"`
	}
	type responseType struct {
		Status   int    `json:"status"`
		Msg      string `json:"msg"`
		ChatHtml string `json:"chatHtml"`
	}
	var req paramsType
	var resp responseType
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
		// 查询记录
		history, err = model.FindCallDriverMessage(req.Nick, 10)
		if err != nil || history == nil {
			logs.Error("get history fail: err=%v history=%v", err, history)
			break
		}
		// 生成聊条记录html代码
		logs.Info("history=%v", history)
		logs.Info("TODO: 生成聊天记录代码")
	}

	logs.Info("get histroy result: req=%v err=%v resp=%v", req, err, resp)

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	} else {
		resp.Status = 0
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprint(w, bytes)
}
