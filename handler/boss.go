package handler

import (
	"fmt"
	"net/http"
	"strings"

	"../toolbox"

	"../config"

	"github.com/astaxie/beego/logs"
)

// boss管理后台前端
func BossFontEndHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.TrimLeft(fmt.Sprintf("%s", r.URL.Path), "/boss")
	logs.Debug("boss font_end: url=%s", url)
	if url == "" {
		url = "index.html"
	}
	targetPath := fmt.Sprintf("%s/%s", config.ServerConfig.BossPath, url)
	http.ServeFile(w, r, targetPath)
	return
}

// boss管理后台的请求全部经过这里
func BossAPIHandler(w http.ResponseWriter, r *http.Request) {
	logs.Debug("boss url=%v", r.URL)
	url := strings.Trim(fmt.Sprintf("%s", r.URL.Path), "/")
	switch url {
	case "bsapi/msg/reqDetail":
		requestDetailHandler(w, r)
	default:
		DefaultHandler(w, r)
	}
}

type respBody struct {
	Status  int         `json:"status"`
	Msg     string      `json:"msg"`
	PayLoad interface{} `json:"payLoad"`
}

// 查看后端收到请求的详细信息
func requestDetailHandler(w http.ResponseWriter, r *http.Request) {
	type payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var res []payload
	tmpIp, tmpPort := toolbox.GetIpAndPort(r)
	res = append(res, payload{Key: "IP", Value: tmpIp})
	res = append(res, payload{Key: "Port", Value: tmpPort})
	res = append(res, payload{Key: "Host", Value: r.Host})
	res = append(res, payload{Key: "Method", Value: r.Method})
	for k, v := range r.Header {
		res = append(res, payload{Key: k, Value: v[0]})
	}
	resp := respBody{
		Status:  0,
		Msg:     "",
		PayLoad: res,
	}
	responseJson(&w, resp)
}
