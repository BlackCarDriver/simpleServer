package toolbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"time"

	"github.com/astaxie/beego/logs"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	go initMailSender()
	go initSysMonitor()
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
func ServerFile(w http.ResponseWriter, filePath string, fileName string, size int64) error {
	if fileName == "" {
		return fmt.Errorf("fileName can't be null")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprint(size))
	w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	_, err = io.Copy(w, file)
	if err != nil {
		return err
	}
	return nil
}

// ServerFile重置版本
func ServerFile2(w *http.ResponseWriter, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	fileState, err := file.Stat()
	if err != nil {
		return err
	}
	if fileState.IsDir() {
		return fmt.Errorf("it is floder")
	}
	(*w).Header().Set("Content-Type", "application/octet-stream")
	(*w).Header().Set("Content-Length", fmt.Sprint(fileState.Size()))
	(*w).Header().Set("content-disposition", fmt.Sprintf("attachment; filename=%s", fileState.Name()))
	_, err = io.Copy(*w, file)
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

// 检查文件是否存在
func CheckFileExist(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			logs.Error("Error unnormal: %v", err)
		}
		return false
	}
	return true
}

// 将数据保存到指定路径的文件中，若文件不存在会自动创建，否则跳过
func WriteToFile(filePath string, data []byte) error {
	var err error
	// 若文件已存在则跳过
	if _, err = os.Stat(filePath); err == nil {
		logs.Warn("skip save file because already exist: filePath=%s", filePath)
		return nil
	}
	if err != nil && os.IsNotExist(err) == false {
		logs.Error("save to file false: error=%v path=%s data=%s", err, filePath, string(data))
		return err
	}
	// 若路径不存在则创建
	dir, _ := path.Split(filePath)
	_, err = os.Stat(dir)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0666)
		if err != nil {
			logs.Error("create directory fail: path=%s error=%v", dir, err)
			return err
		}
		logs.Info("create a directory: dir=%q", dir)
	}
	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		logs.Error("write file fail: error=%v", err)
	}
	return err
}

// 检查一个目录是否存在
func CheckDirExist(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			logs.Error("error with file exist: %v", err)
		}
		return false
	}
	return info.IsDir()
}

// 读取一个文件的文本数据,以json格式解析到某个结构体中, ptrToTarget必须是指针
func ParseJsonFromFile(filePath string, ptrToTarget interface{}) (err error) {
	defer func() {
		msg, ok := recover().(interface{})
		if ok {
			var errptr = &err
			*errptr = fmt.Errorf("catch panic: err=%v", msg)
		}
	}()
	if !CheckFileExist(filePath) {
		return fmt.Errorf("file not exist: path=%s", filePath)
	}
	// 确认是指针
	rType := reflect.TypeOf(ptrToTarget)
	if rType.Kind() != reflect.Ptr {
		return fmt.Errorf("target not a pointer: type=%v", rType.Kind())
	}
	var text string
	text, err = ParseFile(filePath)
	err = json.Unmarshal([]byte(text), &ptrToTarget)
	return err
}

// 将Get请求或Post请求中传输的参数赋值到结构体里面的字段
// ptrToTargetb必须为一个指向结构体的指针, 且通过'json标签'来指定从表单中获取数据的关键字
// 注意事项：只有全部字段都在表单中找到并且转换成功才返回nil; 即使返回err不为空,目标结构体可能被改变;
// 前端使用post请求时加上'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
func MustQueryFromRequest(req *http.Request, ptrToTarget interface{}) (err error) {
	defer func() {
		msg, ok := recover().(interface{})
		if ok {
			var errptr = &err
			*errptr = fmt.Errorf("catch panic: err=%v", msg)
		}
	}()

	if req == nil {
		return fmt.Errorf("params request is null")
	}
	req.ParseForm()
	if len(req.Form) == 0 {
		return fmt.Errorf("request Form is empty")
	}
	// 确认是指针
	rType := reflect.TypeOf(ptrToTarget)
	rValue := reflect.ValueOf(ptrToTarget)
	if rType.Kind() != reflect.Ptr {
		return fmt.Errorf("target not a pointer: type=%v", rType.Kind())
	}
	// 确认指向结构体
	rType = rType.Elem()
	rValue = rValue.Elem()
	if rType.Kind() != reflect.Struct {
		return fmt.Errorf("target is not a pointer to struct: kind=%v", rType.Kind())
	}
	// 确认能被修改
	if !rValue.CanSet() {
		return fmt.Errorf("target can't be changed")
	}

	// 遍历结构体中的字段并从表单中获取相应值
	for i := 0; i < rValue.NumField(); i++ {
		tmpVal := rValue.Field(i)
		vname := rType.Field(i).Name

		if !tmpVal.CanSet() {
			return fmt.Errorf("field can't be set, index=%d name=%s", i, vname)
		}

		jsTag, found := rType.Field(i).Tag.Lookup("json")
		if !found {
			return fmt.Errorf("json tag not found, index=%d name=%v tag=%v", i, vname, rType.Field(i).Tag)
		}

		// 从请求表单中获取相应值
		rawStr := req.Form.Get(jsTag)
		if rawStr == "" && tmpVal.Kind() != reflect.String {
			return fmt.Errorf("field not found in query form: index=%d name=%s jsTag=%s form=%v", i, vname, jsTag, req.Form)
		}

		switch tmpVal.Kind() {
		case reflect.String:
			tmpVal.SetString(rawStr)
		case reflect.Int, reflect.Int32, reflect.Int64:
			tmpInt, err := strconv.ParseInt(rawStr, 10, 64)
			if err != nil {
				return fmt.Errorf("parse Int fail: index=%d name=%s rawStr=%s err=%v", i, vname, rawStr, err)
			}
			tmpVal.SetInt(tmpInt)

		case reflect.Float32, reflect.Float64:
			tmpFloat, err := strconv.ParseFloat(rawStr, 64)
			if err != nil {
				return fmt.Errorf("parse Float fail: index=%d name=%s rawStr=%s err=%v", i, vname, rawStr, err)
			}
			tmpVal.SetFloat(tmpFloat)

		case reflect.Bool:
			tmpBool, err := strconv.ParseBool(rawStr)
			if err != nil {
				return fmt.Errorf("Parse Bool fail: index=%d name=%s rawStr=%s err=%v", i, vname, rawStr, err)
			}
			tmpVal.SetBool(tmpBool)

		default:
			return fmt.Errorf("unsupport kind of field, index=%d name=%v kind=%v", i, vname, tmpVal.Kind())
		}
	}
	return nil
}
