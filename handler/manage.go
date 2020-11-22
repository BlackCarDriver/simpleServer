package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"../config"
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
)

// 管理相关路由全部经过这里
func ManageHandler(w http.ResponseWriter, r *http.Request) {
	if !config.ServerConfig.IsTest && !tb.IsInWhiteList(r) {
		logs.Warn("block a visit for manage")
		RecordRequest(r, "🚯")
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	url := strings.Trim(fmt.Sprintf("%s", r.URL.Path), "/")
	logs.Debug("Manage url=%v", url)
	switch url {
	case "manage":
		manageHtml(w, r)
	case "manage/upload":
		uploadFile(w, r)
	default:
		DefaultHandler(w, r)
	}
}

// 管理页面
func manageHtml(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./source/mange.html")
}

// 接收文件并保存，名字不变
func uploadFile(w http.ResponseWriter, r *http.Request) {
	var err error
	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		logs.Error("Parse form fail: err=%v req=%+v", err, r)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.ParseForm()
	logs.Info("number of upload file: %d", len(r.MultipartForm.File))
	for _, files := range r.MultipartForm.File {
		for _, v := range files {
			logs.Info("save file request: name=%v  size=%v", v.Filename, v.Size)
			file, err := v.Open()
			if err != nil {
				logs.Error("open file fial:　err=%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer file.Close()
			filePath := fmt.Sprintf(config.ServerConfig.StaticPathTP, v.Filename)
			cur, err := os.Create(filePath)
			if err != nil {
				logs.Error("create file fial:　err=%v path=%s", err, filePath)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer cur.Close()
			size, err := io.Copy(cur, file)
			if err != nil {
				logs.Error("save file fail: err=%v path=%s size=%d", err, filePath, size)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			logs.Info("Save file success: size=%d name=%d", size, v.Filename)
			fmt.Fprintf(w, "/static/%s", v.Filename)
		}
	}
}
