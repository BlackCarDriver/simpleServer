package main

import (
	"fmt"
	"net/http"
	"strings"

	"./config"
	"./handler"
	"./rpc"
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
		logs.SetLogger("file", fmt.Sprintf(`{"filename":"%s", "daily": true, "maxlines": 20000}`, config.ServerConfig.LogPath))
		// logs.SetLevel(logs.LevelInformational) // 不打印debug级别日志
	}
	blogHandler = handler.CreateHandler(config.ServerConfig.CloneBlogPath, "bolg")
}

func main() {
	// go test()
	initMain()
	muxer := http.NewServeMux()
	muxer.HandleFunc("/", defaultHandler)
	muxer.HandleFunc("/favicon.ico", handler.FaviconHandler)
	muxer.HandleFunc("/registerS2S", rpc.RegisterServiceHandler) // 注册RPC服务
	// muxer.HandleFunc("/blog/", blogHandler)                                  // 空壳博客
	muxer.HandleFunc("/blog/", handler.FackBlogHandler)                      // 空壳博客
	muxer.HandleFunc("/boss/", handler.BossFontEndHandler)                   // 管理后台前端
	muxer.HandleFunc("/codeMaster/", handler.CodeMasterFontEndHandler)       // codeMaster前端
	muxer.Handle("/bsapi/", MakeSafeHandler(handler.BossAPIHandler))         // 管理后台api
	muxer.Handle("/callDriver/", MakeSafeHandler(handler.CallDriverHandler)) // callDriver应用
	muxer.Handle("/static/", MakeSafeHandler(handler.StaticHandler))         // 静态文件存储服务
	muxer.Handle("/manage/", MakeSafeHandler(handler.ManageHandler))

	err := http.ListenAndServe(":80", muxer)
	if err != nil {
		logs.Emergency("listen http fail: error=%v", err)
	}
}

// 处理其他请求
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.Trim(fmt.Sprint(r.URL.Path), "/")
	if url == "" { // 空路由转跳到空壳博客
		http.Redirect(w, r, "/blog", http.StatusPermanentRedirect)
		return
	}
	logs.Debug("default handler: url=%s", url)
	switch url {
	case "reqMsg": // 查看请求的详细信息
		wrapper(handler.GetRequestDetail, w, r, true, true)
	case "reqLog": // 查看请求日志
		wrapper(handler.GetReqLogs, w, r, true, true)
	case config.ServerConfig.AuthorityKey: // 将ip地址加入权限列表
		handler.AddIpToWhiteList(w, r)
	default:
		handler.NotFoundHandler(w, r)
	}
}

// 在handlerFunc外包装一层, 控制是否进行ip拦截和请求详情记录
func wrapper(defHandler http.HandlerFunc, w http.ResponseWriter, r *http.Request, auth bool, record bool) {
	if auth && !config.ServerConfig.IsTest { // 正式环境下校检特定请求的ip
		if !handler.IpMonitor.IsInWhiteList(r) {
			handler.NotFoundHandler(w, r)
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
	auth    bool // 是否对ip进行拦截
}

func (h myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.auth && !config.ServerConfig.IsTest && !handler.IpMonitor.IsInWhiteList(r) {
		w.WriteHeader(http.StatusNonAuthoritativeInfo)
		fmt.Fprint(w, "sorry, Permission denied...")
		return
	}
	h.handler(w, r)
}

// 生成默认的handler
func MakeDefaultHandler(fv http.HandlerFunc) myHandler {
	return myHandler{
		handler: fv,
	}
}

// 生成对ip进行拦截的handler
func MakeSafeHandler(fv http.HandlerFunc) myHandler {
	return myHandler{
		handler: fv,
		auth:    true,
	}
}

// ===================================
