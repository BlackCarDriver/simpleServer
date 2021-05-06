package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"../config"
	"../model"
	"../rpc"
	tb "../toolbox"

	"github.com/astaxie/beego/logs"
)

var IpMonitor *tb.IPMonitor

// 一些影响系统行为的配置变量
var (
	sendCallDriverEmail = true  // 是否接收callDriver应用的邮件
	sendAlertEmail      = false // 是否发送告警通知 (暂时未用)
)

// 一些信息
var (
	serverStartTime int64 // 程序启动时间
)

func init() {
	serverStartTime = time.Now().Unix()

	// 初始化ip监控
	IpMonitor = tb.NewIpMonitor()

	if !config.ServerConfig.IsTest {
		// 从mongo中读取旧的标记记录，同时开启协程来定期持久化ip标记数据
		oldTags := make(map[string]string)
		err := model.GetUtilData("ipTag", &oldTags)
		if err != nil {
			logs.Error("init ipTag failed: error=%v", err)
		} else {
			IpMonitor.UpdateAllIpTag(oldTags)
		}

		// 从mongo读取RPC服务节点记录，还原上次记录的状态
		rpcNodes := make([]rpc.RegisterPackage, 0)
		err = model.GetUtilData("rpcNodes", &rpcNodes)
		if err != nil {
			logs.Error("init rpcNode failed: error=%v", err)
		} else {
			go rpc.RestoreAllNode(rpcNodes)
		}

		// 定期更新ip标记数据和RPC服务状态
		go func() {
			for range time.NewTicker(10 * time.Minute).C {
				err := model.UpdateUtilData("ipTag", IpMonitor.GetIpTag())
				logs.Debug("update ipTag result: error=%v", err)
				err = model.UpdateUtilData("rpcNodes", rpc.GetAllNodeMsg())
				logs.Debug("update rpcNodes result: error=%v", err)
			}
		}()
	}

	logs.Info("handler init success...")
}

// 查看并返回请求详情
func GetRequestDetail(w http.ResponseWriter, r *http.Request) {
	RecordRequest(r, "")
	ip, port := tb.GetIpAndPort(r)
	content := ""
	content += `<!DOCTYPE html><html lang="zh-CN"><head></head><body>`
	content += fmt.Sprintf("【time】     %s<br><br>", time.Now().Format("2006-01-02 15:04:05"))

	for k, v := range r.Header {
		content += "【" + k + "】     "
		for _, v := range v {
			content += v + "  "
		}
		content += "<br><br>"
	}

	content += "【method】   " + r.Method + "<br><br>"
	content += "【 IP 】     " + ip + "<br><br>"
	content += "【 Port 】     " + port + "<br><br>"
	content += "【URL】     " + fmt.Sprint(r.URL) + "<br><br>"
	content += "【HOST】     " + r.Host + "<br><br>"
	if r.Method != "GET" {
		r.ParseForm()
		content += "【Form】     " + fmt.Sprint(r.Form) + "<br><br>"
	}

	content += `</body></html>`
	w.Write([]byte(content))
}

// 获取访问日志
func GetReqLogs(w http.ResponseWriter, r *http.Request) {
	visitStr := IpMonitor.GetStatic()
	logStr, err := tb.ParseFile(config.ServerConfig.LogPath)
	if err != nil {
		logs.Error("read logs file fail: %v", err)
	}
	fmt.Fprintf(w, "%s\n%s", visitStr, logStr)
}

// 将ip地址加入到白名单
// 可以在url中增加tag参数来指定ip的标记
func AddIpToWhiteList(w http.ResponseWriter, r *http.Request) {
	ip, _ := tb.GetIpAndPort(r)
	tag := "Guest"
	r.ParseForm()
	if r.Form.Get("tag") != "" {
		tag = r.Form.Get("tag")
	}
	IpMonitor.UpdateIpTag(ip, tag)
	RecordRequest(r, "✅")
	fmt.Fprintf(w, "IP=%s \n Tag=%s \n ✅", ip, tag)
}

// 处理没有找到正确路由的请求
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	RecordRequest(r, "🚫")
	w.WriteHeader(http.StatusNotFound)
	assetsHandler(w, "res/html/hello.html")
}

// 返回浏览器标签图标
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	ref := r.Header.Get("Referer")
	if ref == "" {
		logs.Warn("unexpect icon for url: url=%s", r.URL)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	logs.Debug("ref=%s", ref)
	URL, err := url.Parse(ref)
	if err != nil {
		logs.Warn("parse ref failed: ref=%s", ref)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	app := strings.Trim(URL.Path, "/")
	switch app {
	case "codeMaster":
		assetsHandler(w, "res/icon/codeMaster.ico")
	case "boss":
		assetsHandler(w, "res/icon/boss.ico")
	default:
		logs.Warn("unexpect icon for app: app=%s", app)
	}
	return
}

// ====================== commom =================================

// 记录访问日志
func RecordRequest(req *http.Request, preFix string) {
	ip, port := tb.GetIpAndPort(req)
	visitTimes := IpMonitor.GetAndAddIpVisitTimes(ip)
	log := fmt.Sprintf("%s  %d  %s  %s  %s  %s  %s  %s",
		preFix,
		visitTimes,
		ip,
		port,
		req.Method,
		req.URL,
		req.Header["User-Agent"],
		req.Header["Accept-Language"],
	)
	if visitTimes == 0 { // 对于某个第一次访问的Ip做特殊处理
		log = "🔥" + log
	}
	logs.Info(log)
}

func responseJson(w *http.ResponseWriter, payload interface{}) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		logs.Error("json marshal error: payload=%+v error=%v", payload, err)
		(*w).WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(*w, "sorry, something bad happen...")
		return
	}
	(*w).Header().Add("content-type", "application/json")
	fmt.Fprintf(*w, "%s", bytes)
}
