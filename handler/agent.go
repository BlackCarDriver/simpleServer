package handler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"../toolbox"
	"github.com/astaxie/beego/logs"
)

var (
	saveRoot    = "D:/TEMP/Clone"                // 克隆网站时响应体保存位置的根路径
	targetUrlTP = "https://biaochenxuying.cn/%s" // 修改这里改变需要克隆的目标网站
)

func initCloner() {
	logs.Info("cloner init...")
	saveRoot = strings.TrimRight(saveRoot, "/")
	if saveRoot == "" {
		logs.Emergency("config saveRoot can't be null")
	}
	if !toolbox.CheckDirExist(saveRoot) {
		logs.Emergency("cloner save root not exits: saveRoot=%s", saveRoot)
	}
}

// 正向代理浏览网站并保存响应数据到本地
func CloneAgent(w http.ResponseWriter, r *http.Request) {
	req, _ := http.NewRequest(r.Method, newURI(r.RequestURI), r.Body)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logs.Error("Request fail: error=%v method=%v URI=%v", err, r.Body, r.RequestURI)
		return
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	saveResponse(r.RequestURI, body)
	logs.Debug("URI=%s  header=%+v", r.RequestURI, res.Header)
	for k, v := range res.Header {
		w.Header().Set(k, v[0])
	}
	w.Write(body)
}

// 本地浏览缓存下来的页面
func ServerWithBuff(w http.ResponseWriter, r *http.Request) {
	URI := r.RequestURI
	URI = strings.TrimLeft(URI, "/")
	if URI == "" {
		logs.Warn("auto use index to save response")
		URI = "index.html"
	}
	// 若为api请求，则修改文件名
	if strings.Contains(URI, "?") {
		URI = remakeURI(URI)
	}
	targetPath := fmt.Sprintf("%s/%s", saveRoot, URI)
	logs.Debug("targetPath=%s", targetPath)
	setHeader(&w, r.RequestURI)
	http.ServeFile(w, r, targetPath)
}

// 代理中URL的转换
func newURI(oldURI string) string {

	newURI := fmt.Sprintf(targetUrlTP, strings.Trim(oldURI, "/"))
	logs.Debug("%s ---> %s", oldURI, newURI)
	return newURI
}

// 保存响应主体
func saveResponse(URI string, data []byte) {
	URI = strings.TrimLeft(URI, "/")
	if URI == "" {
		logs.Warn("auto use index to save response")
		URI = "index.html"
	}
	// 若为api请求，则修改文件名
	if strings.Contains(URI, "?") {
		URI = remakeURI(URI)
	}
	targetPath := fmt.Sprintf("%s/%s", saveRoot, URI)
	err := toolbox.WriteToFile(targetPath, data)
	logs.Debug("save response result: targetPath=%s error=%v", targetPath, err)
}

// 用于保存响应体到本地时，重构文件名，将非法字符用下划线代替
func remakeURI(URI string) string {
	index := strings.Index(URI, "?")
	if index < 0 {
		return URI
	}
	path := URI[0:index]
	name := strings.TrimLeft(URI[index:], "?")
	name = regexp.MustCompile(`[\/:*?"<>|]`).ReplaceAllString(name, "_")
	return fmt.Sprintf("%s/%s", path, name)
}

// 用于返回保存在本地的数据时设置响应头header
func setHeader(w *http.ResponseWriter, URI string) {
	(*w).Header().Set("Connection", "keep-alive")
	logs.Debug("URI=%s", URI)
	if URI == "/project" {
		(*w).Header().Set("Content-Type:", "text/html")
		return
	}
	if strings.HasSuffix(URI, ".html") || URI == "/project" {
		(*w).Header().Set("Content-Type:", "text/html")
		return
	}
	if strings.Contains(URI, "css") {
		(*w).Header().Set("Content-Type:", "text/css")
	}
	if strings.HasSuffix(URI, ".ico") {
		(*w).Header().Set("Content-Type:", "image/x-icon")
		return
	}
	if strings.HasSuffix(URI, ".js") {
		(*w).Header().Set("Content-Type:", "application/javascript")
		return
	}
	if strings.HasSuffix(URI, ".json") {
		(*w).Header().Set("Content-Type:", "application/json; charset=utf-8")
		return
	}
	if strings.HasSuffix(URI, ".png") {
		(*w).Header().Set("Content-Type:", "image/png")
		return
	}
	if strings.HasSuffix(URI, ".gif") {
		(*w).Header().Set("Content-Type:", "image/gif")
		return
	}
	if strings.HasSuffix(URI, ".svg") {
		(*w).Header().Set("Content-Type:", "image/svg+xml")
		return
	}
	if strings.HasSuffix(URI, ".jpeg") {
		(*w).Header().Set("Content-Type:", "image/jpeg")
		return
	}
}
