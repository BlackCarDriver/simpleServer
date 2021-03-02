package handler

import (
	"fmt"
	"net/http"
	"strings"

	"../assets"
	"../config"
	"github.com/astaxie/beego/logs"
)

// codeMaster前端
func CodeMasterFontEndHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.TrimPrefix(r.URL.Path, "/codeMaster")
	logs.Debug("codeMaster font_end: url=%s", url)
	if url == "" {
		url = "index.html"
	}
	targetPath := config.ServerConfig.CodeMasterPath + url
	http.ServeFile(w, r, targetPath)
	return
}

// 测试返回assets
func FackBlogHandler(w http.ResponseWriter, r *http.Request) {
	URI := r.RequestURI
	URI = strings.TrimLeft(URI, "/blog/")
	if URI == "" {
		logs.Warn("auto use index to save response")
		URI = "index.html"
	}
	logs.Info("uri=%s", URI)
	bs, err := assets.Asset("res/fackBlog/" + URI)
	if err != nil {
		logs.Error("assets error: error=%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Fprint(w, string(bs))
}
