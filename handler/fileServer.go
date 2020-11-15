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
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
)

// 接受一个post请求，将主体中的文件保存下来，返回一个下载链接,每次仅支持上传单个文件
// Example： curl -F 'file=@default.conf' http://localhost:80/upload
func UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logs.Warn("UploadFile() receive bad request: %v", r)
		w.WriteHeader(http.StatusMethodNotAllowed)
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

	//获取表单文件, 此时 v 为 []*FileHeader, FileHeader包含字段：FileName 和  Size ...
	for _, files := range r.MultipartForm.File {
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

		randName := tb.GetRandomString(8)
		//先创建一个文件，然后使用io.Copy来保存表单中的文件
		var cur *os.File
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
		downloadUrl := fmt.Sprintf(config.ServerConfig.DownloadUrlTp, randName)
		fmt.Fprintf(w, "\nSave file success: size=%d name=%s \n download_url=%s", size, header.Filename, downloadUrl)
	}
	return
}

// 返回文件供下载
// Example: curl http://localhost:80/download/abcdefgddd
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	logs.Info(r)
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
	fileName := url[9:]
	logs.Info("sourceName=%s", fileName)

	filePath := fmt.Sprintf(config.ServerConfig.SourcePathTp, fileName)
	err := tb.ServerFile(w, filePath, "fileName")
	if err != nil {
		logs.Error("ServerFile fail: %v", err)
		fmt.Fprintf(w, "Error happen: %v", err)
		return
	}
	logs.Info("Server file success")
}
