package handler

// 提供文件上传和下载功能，可通过curl和wget命令实现文件传送
import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"../config"
	"../model"
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
)

// 静态文件存储服务,可用于在服务器之间通过命令行传送文件，收到的文件在存储时会隐藏文件名等信息
func StaticHandler(w http.ResponseWriter, r *http.Request) {
	logs.Debug("static url=%v", r.URL)
	url := strings.Trim(fmt.Sprintf("%s", r.URL.Path), "/")
	if url == "static/upload" {
		staticUploadHandler(w, r)
	} else if strings.HasPrefix(url, "static/download/") {
		staticDownloadHandler(w, r)
	} else if strings.HasPrefix(url, "static/preview/") {
		staticPreViewHandler(w, r)
	} else {
		logs.Warn("skip unexpect static request: url=%s", url)
		w.WriteHeader(http.StatusForbidden)
	}
	return
}

// 接受一个post请求，将主体中的文件保存下来，返回一个下载链接,每次仅支持上传单个文件
// Example：curl -F 'file=@default.conf' http://localhost:80/static/upload
func staticUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logs.Warn("UploadFile() receive bad request: %v", r)
		fmt.Fprint(w, "demo: curl -F 'file=@default.conf' http://localhost:80/upload")
		return
	}

	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(w, "Error happen: %v", err)
		}
	}()

	r.ParseForm()
	logs.Info("%+v", r)
	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		logs.Error(err)
		return
	}

	// 拒绝上传多个文件的请求
	if size := len(r.MultipartForm.File); size != 1 {
		err = fmt.Errorf("Reject because size of multipartForm is %d", size)
		logs.Warn(err)
		return
	}

	for _, files := range r.MultipartForm.File {
		// 解析表单
		if size := len(files); size != 1 {
			err = fmt.Errorf("Reject because size of fileHeader is %d", len(files))
			return
		}
		header := files[0]
		logs.Info("name=%s  size=%d", header.Filename, header.Size)
		var file multipart.File
		file, err = header.Open()
		if err != nil {
			logs.Error("open upload file fail: %v", err)
			return
		}
		defer file.Close()

		// 保存文件到本地，名字名字为随机，长度为8
		var cur *os.File
		randName := tb.GetRandomString(8)
		filePath := fmt.Sprintf("%s%s.tmp", config.ServerConfig.StaticPath, randName)
		cur, err = os.Create(filePath)
		if err != nil {
			logs.Error("create file fail: %v", err)
			return
		}
		defer cur.Close()
		var size int64
		size, err = io.Copy(cur, file)
		if err != nil {
			return
		}
		// 记录上传记录到mongo
		err = model.InsertUploadRecord(header.Filename, randName, size)
		if err != nil {
			logs.Error("save upload file record fail: err=%v", err)
			break
		}
		downloadUrl := fmt.Sprintf("%s/static/download/%s", config.ServerConfig.ServerURL, randName)
		previewUrl := fmt.Sprintf("%s/static/preview/%s.tmp", config.ServerConfig.ServerURL, randName)
		fmt.Fprintf(w,
			"\n Save file success: size=%d name=%s\n browser_download_url:  %s\n browser_preview_url: %s\n command_download_url:  wget --no-check-certificate --content-disposition %s \n",
			size, header.Filename, downloadUrl, previewUrl, downloadUrl)
	}
	return
}

// 以弹窗下载方式返回staticUploadHandler上传的文件
// Example: wget --no-check-certificate http://localhost:80/download/abcdefgddd
func staticDownloadHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		logs.Warn("DownloadFile reeceive a bad request: %v", r)
		return
	}
	url := strings.Trim(fmt.Sprint(r.URL), "/") // 取件码必须正好为10个小写字母
	if !regexp.MustCompile("^static/download/[a-z]{8}$").MatchString(url) {
		w.WriteHeader(http.StatusBadRequest)
		logs.Warn("unexpect url: %s", r.URL)
		return
	}
	code := url[16:] // 取件码
	filePath := fmt.Sprintf("%s%s.tmp", config.ServerConfig.StaticPath, code)
	if !tb.CheckFileExist(filePath) {
		logs.Info("file not exist: path=%s", filePath)
		fmt.Fprintf(w, "file not exist")
		return
	}

	var record model.FileUpload
	record, err = model.GetUploadRecord(code)
	if err != nil {
		logs.Info("record not found: path=%s  err=%v", filePath, err)
		fmt.Fprintf(w, "file record not found: %v", err)
	}

	err = tb.ServerFile(w, filePath, record.FileName, record.Size)
	if err != nil {
		logs.Error("ServerFile fail: %v", err)
		fmt.Fprintf(w, "Error happen: %v", err)
		return
	}
	logs.Info("Server file success: %+v", record)
}

// 以预览方式返回StaticPath目录下的图片等资源
// url format: /static/preview/${fileName}
func staticPreViewHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		logs.Warn("DownloadFile reeceive a bad request: %v", r)
		return
	}
	uri := strings.Trim(fmt.Sprint(r.URL), "/")
	uri, _ = url.QueryUnescape(uri)
	fileName := strings.TrimPrefix(uri, "static/preview/")
	filePath := config.ServerConfig.StaticPath + fileName
	logs.Info("get static path: %s", filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logs.Info("stat file fail: error=%v filePath=%s", err, filePath)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if fileInfo.IsDir() {
		logs.Warn("filePath is a floder: path=%s", filePath)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filePath)
	return
}
