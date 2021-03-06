package handler

import (
	"../assets"
	// "fmt"
	"github.com/astaxie/beego/logs"
	// "io"
	// "io/ioutil"
	// "bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

func assetsHandler(w http.ResponseWriter, URI string) {
	bs, err := assets.Asset(URI)
	if err != nil {
		logs.Error("assets error: error=%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctype := mime.TypeByExtension(filepath.Ext(URI))
	w.Header().Set("Content-Type", ctype)

	w.Write(bs)
}

func CreateAssetsHandler(assetsRoot string, rmPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		URI := r.RequestURI
		logs.Debug("URI=%s", URI)
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
		logs.Debug("targetPath=%s", targetPath)
		assetsHandler(w, targetPath)
	}
}
