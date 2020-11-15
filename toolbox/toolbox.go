package toolbox

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/astaxie/beego/logs"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// 从请求中获取IP和端口
func GetIpAndPort(r *http.Request) (remoteAddr, port string) {
	remoteAddr = r.RemoteAddr
	port = "?"
	XForwardedFor := "X-Forwarded-For"
	XRealIP := "X-Real-IP"
	if ip := r.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = r.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, port, _ = net.SplitHostPort(remoteAddr)
	}
	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}
	return
}

// 读取一个文件的内容到字符串
func ParseFile(path string) (text string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("Open %s fall: %v", path, err)
	}
	defer file.Close()
	buf := bufio.NewReader(file)
	bytes, err := ioutil.ReadAll(buf)
	if err != nil {
		return "", fmt.Errorf("ioutil.ReadAll fall : %v", err)
	}
	return string(bytes), nil
}

// 清空一个文件的内容
func ClearFile(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		logs.Error(err)
		return err
	}
	defer file.Close()
	if _, err = file.WriteString(""); err != nil {
		logs.Error(err)
		return err
	}
	return nil
}

// 文件服务，提供弹出下载弹框的响应
func ServerFile(w http.ResponseWriter, filePath string, fileName string) error {
	if fileName == "" {
		return fmt.Errorf("fileName can't be null")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	_, err = io.Copy(w, file)
	if err != nil {
		return err
	}
	return nil
}

// 生成一个随机字符串
func GetRandomString(l int) string {
	str := "abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	for i := 0; i < l; i++ {
		result = append(result, bytes[rand.Int()%len(bytes)])
	}
	return string(result)
}
