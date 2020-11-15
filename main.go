package main

import (
	"fmt"
	"net/http"
	"strings"

	"./handler"
	"github.com/astaxie/beego/logs"
)

func main() {
	http.HandleFunc("/", mainRouter)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		logs.Error("Run server fail: %v", err)
	}
}

// 第一层路由,所有的handler都经过这里
func mainRouter(w http.ResponseWriter, r *http.Request) {
	url := strings.Trim(fmt.Sprint(r.URL), "/")
	logs.Info(url)
	switch url {
	case "favicon.ico":
		http.ServeFile(w, r, "./static/facicon.ico")
	case "reqMsg":
		handler.GetRequestDetail(w, r)
	case "reqLog":
		handler.GetReqLogs(w, r)
	case "BlackCarDriver":
		handler.AddIpToWhiteList(w, r)
	default:
		handler.DefaultHandler(w, r)
	}
}
