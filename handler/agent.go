package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"../toolbox"
	"github.com/astaxie/beego/logs"
)

var (
	saveRoot    = "D:/WorkPlace/Git WorPlace/CloneWebSite/blog" // 克隆网站时响应体保存位置的根路径
	targetUrlTP = "https://biaochenxuying.cn/%s"                // 修改这里改变需要克隆的目标网站
	headerMap   = make(map[string]http.Header)
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

// 作用:通过正向代理浏览网站，将所有请求得到的响应数据保存到$saveRoot,用于克隆目标网站
// 使用方法：http.HandleFunc("/", CloneAgent), 浏览完目标网站后访问/exit，生成响应头数据
func CloneAgent(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/exit" { // 保存响应头
		headData, _ := json.Marshal(headerMap)
		targetPath := fmt.Sprintf("%s/header.json", saveRoot)
		err := toolbox.WriteToFile(targetPath, headData)
		fmt.Fprintf(w, "save header result: error=%v  goodbey", err)
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}

	newURI := fmt.Sprintf(targetUrlTP, strings.Trim(r.RequestURI, "/"))
	req, _ := http.NewRequest(r.Method, newURI, r.Body)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logs.Error("Request fail: error=%v method=%v URI=%v", err, r.Body, r.RequestURI)
		return
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	tmpUri := saveResponse(r.RequestURI, body)
	headerMap[tmpUri] = res.Header

	for k, v := range res.Header {
		w.Header().Set(k, v[0])
	}
	w.Write(body)
}

// 创建一个反向代理处理器,(专门用于返回cloneAgent缓存下来的请求响应体), 使用http.ServeFile()方法返回
// fileRoot：缓存文件保存的根目录, rmPrefix: 根据访问URI寻找文件路径时去除的URL前缀,无需斜杠
func CreateAgentHandler(fileRoot, rmPrefix string) http.HandlerFunc {
	if !toolbox.CheckDirExist(fileRoot) {
		logs.Emergency("fileRoot not exist: fileRoot=%s", fileRoot)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		URI := r.RequestURI
		URI = strings.TrimLeft(URI, "/")
		if rmPrefix != "" {
			URI = strings.TrimLeft(URI, rmPrefix)
			URI = strings.TrimLeft(URI, "/")
		}
		if URI == "" {
			logs.Warn("auto use index to save response")
			URI = "index.html"
		}
		targetPath := fileRoot + URI
		// 若为api请求，则修改文件名
		if strings.Contains(URI, "?") {
			URI = remakeURI(URI)
			targetPath = fileRoot + URI
		}
		logs.Debug("targetPath=%s", targetPath)
		http.ServeFile(w, r, targetPath)
	}
}

// 创建一个反向代理处理器, 使用assets方法返回经过gzip压缩静态文件
// 最终访问的路径为 ${assetsRoot}${去除rmPrefix前缀的URI},assetsRoot应以斜杠结尾
func CreateAgentHandler2(assetsRoot string, rmPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		URI := r.RequestURI
		URI = strings.TrimLeft(URI, "/")
		if rmPrefix != "" {
			URI = strings.TrimLeft(URI, rmPrefix)
			URI = strings.TrimLeft(URI, "/")
		}
		if URI == "" {
			logs.Info("auto use index to save response")
			URI = "index.html"
		}
		targetPath := assetsRoot + URI
		// 若为api请求，则修改文件名
		if strings.Contains(URI, "?") {
			targetPath = assetsRoot + remakeURI(URI)
		}
		logs.Debug("targetPath=%s", targetPath)
		assetsHandler(w, targetPath)
	}
}

// ========================================================

// 请求目标网站的数据并保存响应主体,若目标文件已存在则跳过, 返回一个可识别保存路径的URI
func saveResponse(URI string, data []byte) string {
	URI = strings.TrimLeft(URI, "/")
	if URI == "" {
		logs.Warn("auto use index to save response")
		URI = "index.html"
	}
	targetPath := fmt.Sprintf("%s/%s", saveRoot, URI)
	// 对api请求的响应进行进行特殊处理
	if strings.Contains(URI, "?") {
		URI = remakeURI(URI)
		targetPath = fmt.Sprintf("%s/%s", saveRoot, URI)
	}
	err := toolbox.WriteToFile(targetPath, data)
	logs.Debug("save response result: targetPath=%s error=%v", targetPath, err)
	return URI
}

// 用于保存响应体到本地时，重构文件名，将非法字符用下划线代替
func remakeURI(URI string) string {
	index := strings.Index(URI, "?")
	if index < 0 {
		return URI
	}
	path := URI[0:index]
	path = strings.TrimLeft(path, "/")
	path = strings.TrimLeft(path, "api/")
	params := strings.TrimLeft(URI[index:], "?")
	params = regexp.MustCompile(`[\/:*?"<>|]`).ReplaceAllString(params, "_")
	return fmt.Sprintf("api/%s/%s.json", path, params)
}

/*
// 用于返回保存在本地的数据时设置响应头header
func setDefaultHeader(w *http.ResponseWriter, URI string) {
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
*/
