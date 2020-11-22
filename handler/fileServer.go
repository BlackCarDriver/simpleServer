package handler

// 提供文件上传和下载功能，可通过curl和wget命令实现文件传送
import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"

	"../config"
	"../model"
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
)

// 接受一个post请求，将主体中的文件保存下来，返回一个下载链接,每次仅支持上传单个文件
// Example：curl -F 'file=@default.conf' http://localhost:80/upload
func UploadFile(w http.ResponseWriter, r *http.Request) {
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
		filePath := fmt.Sprintf(config.ServerConfig.SourcePathTp, randName)
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
		}
		downloadUrl := fmt.Sprintf(config.ServerConfig.DownloadUrlTp, randName)
		fmt.Fprintf(w, `\nSave file success: size=%d name=%s \n
		browser_download_url:  %s \n
		command_download_url:  wget --no-check-certificate --content-disposition %s \n
		`, size, header.Filename, downloadUrl, downloadUrl)
	}
	return
}

// 返回文件供下载
// Example: wget --no-check-certificate http://localhost:80/download/abcdefgddd
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	var err error
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		logs.Warn("DownloadFile reeceive a bad request: %v", r)
		return
	}
	url := strings.Trim(fmt.Sprint(r.URL), "/") // 取件码必须正好为10个小写字母
	if !regexp.MustCompile("^download/[a-z]{8}$").MatchString(url) {
		w.WriteHeader(http.StatusBadRequest)
		logs.Warn("unexpect url: %s", r.URL)
		return
	}
	code := url[9:] // 取件码
	filePath := fmt.Sprintf(config.ServerConfig.SourcePathTp, code)
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

// 返回图片等资源
func StatisHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		logs.Warn("DownloadFile reeceive a bad request: %v", r)
		return
	}
	url := strings.Trim(fmt.Sprint(r.URL), "/")
	if !regexp.MustCompile("^static/\\w{2,40}\\.\\w{2,5}$").MatchString(url) {
		w.WriteHeader(http.StatusBadRequest)
		logs.Warn("unexpect url: %s", r.URL)
		return
	}
	fileName := strings.TrimPrefix(url, "static/")
	filePath := fmt.Sprintf(config.ServerConfig.StaticPathTP, fileName)
	logs.Info("get static path: %s", filePath)
	if !tb.CheckFileExist(filePath) {
		logs.Info("files not exist: %s", filePath)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		logs.Error("open file fail: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	size, err := io.Copy(w, file)
	if err != nil {
		logs.Error("return statis fail: err=%e", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	logs.Info("server statis file: size=%d  name=%s", size, fileName)
	return
}
