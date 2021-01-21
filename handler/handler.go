package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"../config"
	tb "../toolbox"

	"github.com/astaxie/beego/logs"
)

func init() {
	// initCloner()
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
	visitStr := tb.GetStatic()
	logStr, err := tb.ParseFile(config.ServerConfig.LogPath)
	if err != nil {
		logs.Error("read logs file fail: %v", err)
	}
	fmt.Fprintf(w, "%s\n%s", visitStr, logStr)
}

// 将ip地址加入到白名单
func AddIpToWhiteList(w http.ResponseWriter, r *http.Request) {
	ip, _ := tb.GetIpAndPort(r)
	tb.AddWhiteList(ip)
	RecordRequest(r, "✅")
	fmt.Fprint(w, "👌 OK!")
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
	visitTimes := tb.GetAndAddIpVisitTimes(ip)
	log := fmt.Sprintf("%s%d  %d  %s  %s  %s  %s  %s  %s",
		preFix,
		tb.RequestCounter,
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
