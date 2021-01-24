package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"../config"
	"../model"
	tb "../toolbox"
	sm "./s2sMaster"

	"github.com/astaxie/beego/logs"
)

var IpMonitor *tb.IPMonitor
var s2sMaster *sm.ServiceMaster

func init() {
	// 初始化ip监控
	IpMonitor = tb.MakeIpMonitor()
	// 从数据库读取已有的ip标记数据
	oldTags := make(map[string]string)
	err := model.GetUtilData("ipTag", &oldTags)
	if err != nil {
		logs.Error("init ipTag failed: error=%v", err)
	}
	IpMonitor.UpdateAllIpTag(oldTags)
	// 开启一个协程，每隔一段时间备份ip标记到mongo
	go func() {
		for _ = range time.NewTicker(10 * time.Minute).C {
			err := model.UpdateUtilData("ipTag", IpMonitor.GetIpTag())
			logs.Debug("update ipTag result: error=%v", err)
		}
	}()
	// 初始化s2s
	s2sMaster = sm.NewServiceMaster("secret")
}

// 注册一个RPC服务
func RegisterServiceHandler(w http.ResponseWriter, r *http.Request) {
	var req sm.RegisterPackage
	var err error
	var resp struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	for loop := true; loop; loop = false {
		err = tb.MustQueryFromRequest(r, &req)
		if err != nil {
			break
		}
		err = s2sMaster.Register(req)
		if err != nil {
			break
		}
		logs.Info("register service success: req=%+v", req)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	resp.Msg = "OK"
	responseJson(&w, resp)
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
func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	RecordRequest(r, "🚫")
	w.WriteHeader(http.StatusNotFound)
	http.ServeFile(w, r, "./source/hello.html")
}

// =================================================================

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
		return
	}
	fmt.Fprintf(*w, "%s", bytes)
}
