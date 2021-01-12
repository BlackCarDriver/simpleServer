package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"./config"
	"./handler"
	tb "./toolbox"
	"github.com/astaxie/beego/logs"
)

func initMain() {
	logs.SetLogFuncCall(true) // 文件名和行号HandleTest
	if config.ServerConfig.IsTest {
		logs.SetLogger("console")
	} else {
		logs.SetLogger("file", `{"filename":"./server.log"}`)
		logs.SetLevel(logs.LevelInformational) // 不打印debug级别日志
	}
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
}

// 一些特定功能的处理器
var (
	blogHandler = handler.CreateHandler("D:/WorkPlace/Git WorPlace/CloneWebSite/blog", "bolg")
)

func main() {
	initMain()
	// http.HandleFunc("/", handler.CloneAgent) // 网站克隆
	http.HandleFunc("/", mainRouter)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		logs.Error("Run server fail: %v", err)
	}
}

// 第一层路由,所有的handler都经过这里
func mainRouter(w http.ResponseWriter, r *http.Request) {
	url := strings.Trim(fmt.Sprint(r.URL), "/")
	if url == "" { // 空路由转跳到空壳博客
		http.Redirect(w, r, "/blog", http.StatusPermanentRedirect)
	}
	logs.Debug(url)
	switch url {
	case "favicon.ico": // 返回显示的图标
		http.ServeFile(w, r, "./source/favicon.ico")

	case "reqMsg": // 查看请求的详细信息
		wrapper(handler.GetRequestDetail, w, r, true, true)

	case "reqLog": // 查看请求日志
		wrapper(handler.GetReqLogs, w, r, true, true)

	case "upload": // 上传文件
		wrapper(handler.UploadFile, w, r, true, true)

	case config.ServerConfig.AuthorityKey: // 将ip地址加入权限列表
		handler.AddIpToWhiteList(w, r)

	default:
		regexHandler(w, r, url)
	}
}

// 处理模糊匹配的路由
func regexHandler(w http.ResponseWriter, r *http.Request, url string) {
	if !config.ServerConfig.IsTest && !tb.IsInWhiteList(r) { // 正式环境下对ip做拦截
		handler.DefaultHandler(w, r)
		return
	}
	if strings.HasPrefix(url, "") { // 空壳博客
		blogHandler(w, r)
		return
	}
	if regexp.MustCompile("^download/[a-z]{5,20}$").MatchString(url) { // 下载文件
		wrapper(handler.DownloadFile, w, r, true, true)
		return
	}
	if strings.HasPrefix(url, "static") { // 静态文件服务
		handler.StatisHandler(w, r)
		return
	}
	if strings.HasPrefix(url, "callDriver") { // callDriver应用
		handler.CallDriverHandler(w, r)
		return
	}
	if strings.HasPrefix(url, "manage") { // 管理页面
		handler.ManageHandler(w, r)
		return
	}
	handler.DefaultHandler(w, r)
}

// 在handler外包装一层控制是权限和请求信息记录
func wrapper(defHandler http.HandlerFunc, w http.ResponseWriter, r *http.Request, auth bool, record bool) {
	if auth && !config.ServerConfig.IsTest { // 正式环境下校检特定请求的ip
		if !tb.IsInWhiteList(r) {
			handler.DefaultHandler(w, r)
			return
		}
	}
	if record {
		handler.RecordRequest(r, "")
	}
	defHandler(w, r)
}
