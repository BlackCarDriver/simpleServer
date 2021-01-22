package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"./config"
	"./handler"
	tb "./toolbox"
	"github.com/astaxie/beego/logs"
)

// 一些特定功能的处理器
var (
	blogHandler http.HandlerFunc
)

func initMain() {
	logs.SetLogFuncCall(true) // 文件名和行号HandleTest
	logs.SetLogFuncCallDepth(3)
	logs.EnableFuncCallDepth(true)
	if config.ServerConfig.IsTest {
		logs.SetLogger("console")
	} else {
		logs.SetLogger("file", fmt.Sprintf(`{"filename":"%s", "daily ": "false"}`, config.ServerConfig.LogPath))
		// logs.SetLevel(logs.LevelInformational) // 不打印debug级别日志
	}
	blogHandler = handler.CreateHandler(config.ServerConfig.CloneBlogPath, "bolg")
}

func test() {
	res, err := url.QueryUnescape("/static/preview/%E5%8A%A8%E7%94%BB%E7%89%87.mp4")
	logs.Debug("res=%q  error=%v", res, err)
	os.Exit(0)
}

func main() {
	// test()
	initMain()
	muxer := http.NewServeMux()
	muxer.Handle("/", MakeHandler(defaultHandler))
	muxer.Handle("/blog/", blogHandler)                                  // 空壳博客
	muxer.Handle("/boss/", MakeHandler(handler.BossFontEndHandler))      // 管理后台
	muxer.Handle("/bsapi/", MakeHandler(handler.BossAPIHandler))         // 管理后台api
	muxer.Handle("/callDriver/", MakeHandler(handler.CallDriverHandler)) // callDriver应用
	muxer.Handle("/static/", MakeHandler(handler.StaticHandler))         // 静态文件存储服务
	muxer.Handle("/manage/", MakeHandler(handler.ManageHandler))

	err := http.ListenAndServe(":80", muxer)
	if err != nil {
		logs.Emergency("listen http fail: error=%v", err)
	}
}

// 处理其他请求
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.Trim(fmt.Sprint(r.URL), "/")
	if url == "" { // 空路由转跳到空壳博客
		http.Redirect(w, r, "/blog", http.StatusPermanentRedirect)
		return
	}
	logs.Debug("default handler: url=%s", url)
	switch url {
	case "favicon.ico": // 返回浏览器标签显示的图标
		http.ServeFile(w, r, "./source/favicon.ico")
	case "reqMsg": // 查看请求的详细信息
		wrapper(handler.GetRequestDetail, w, r, true, true)
	case "reqLog": // 查看请求日志
		wrapper(handler.GetReqLogs, w, r, true, true)
	case config.ServerConfig.AuthorityKey: // 将ip地址加入权限列表
		handler.AddIpToWhiteList(w, r)
	default:
		handler.DefaultHandler(w, r)
	}
}

// 在handler外包装一层, 控制是否进行ip拦截和请求详情记录
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

// ========= Handler Maker ===========
type myHandler struct {
	handler http.HandlerFunc
}

func (h myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.handler == nil {
		logs.Emergency("calling a empty a handler")
		return
	}
	h.handler(w, r)
}
func MakeHandler(fv http.HandlerFunc) myHandler {
	return myHandler{
		handler: fv,
	}
}

// ===================================
