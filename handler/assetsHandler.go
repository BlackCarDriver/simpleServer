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
