package handler

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"

	"../toolbox"

	"../config"

	"github.com/astaxie/beego/logs"
)

// boss管理后台前端
func BossFontEndHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.TrimLeft(fmt.Sprintf("%s", r.URL.Path), "/boss")
	logs.Debug("boss font_end: url=%s", url)
	if url == "" {
		url = "index.html"
	}
	targetPath := config.ServerConfig.BossPath + url
	http.ServeFile(w, r, targetPath)
	return
}

// boss管理后台的请求全部经过这里
func BossAPIHandler(w http.ResponseWriter, r *http.Request) {
	url := strings.Trim(fmt.Sprintf("%s", r.URL.Path), "/")
	logs.Debug("boss url=%v", url)
	switch url {
	case "bsapi/msg/reqDetail":
		requestDetailHandler(w, r)
	case "bsapi/tool/netdish/fileslist":
		netDishListFilesHandler(w, r)
	case "bsapi/tool/netdish/fileOpe":
		netDishFileOpeHandler(w, r)
	case "bsapi/tool/netdish/upload":
		netDishFileUploadHandler(w, r)
	case "bsapi/manage/ipWhiteList/list":
		IPWhitelistHandler(w, r)
	case "bsapi/manage/ipWhiteList/ope":
		IPWhitelistOpeHandler(w, r)
	default:
		DefaultHandler(w, r)
	}
}

type respStruct struct {
	Status  int         `json:"status"`
	Msg     string      `json:"msg"`
	PayLoad interface{} `json:"payLoad"`
}

// 服务端工具-请求详情：查看后端收到请求的详细信息
func requestDetailHandler(w http.ResponseWriter, r *http.Request) {
	type payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var res []payload
	tmpIp, tmpPort := toolbox.GetIpAndPort(r)
	res = append(res, payload{Key: "IP", Value: tmpIp})
	res = append(res, payload{Key: "IpTag", Value: IpMonitor.QueryIpTag(r)})
	res = append(res, payload{Key: "Port", Value: tmpPort})
	res = append(res, payload{Key: "Host", Value: r.Host})
	res = append(res, payload{Key: "Method", Value: r.Method})
	for k, v := range r.Header {
		res = append(res, payload{Key: k, Value: v[0]})
	}
	resp := respStruct{
		Status:  0,
		Msg:     "",
		PayLoad: res,
	}
	responseJson(&w, resp)
}

//  服务端工具-个人网盘：获取列表
func netDishListFilesHandler(w http.ResponseWriter, r *http.Request) {
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		type fileInfo struct {
			Name      string `json:"fileName"`
			Size      int64  `json:"size"`
			Timestamp int64  `json:"timestamp"`
		}
		var payLoad []fileInfo
		var filesInfos []os.FileInfo
		filesInfos, err = ioutil.ReadDir(config.ServerConfig.StaticPath)
		if err != nil {
			logs.Error("Read dir fail: path=%s error=%v", config.ServerConfig.StaticPath, err)
			break
		}
		for _, info := range filesInfos {
			if info.IsDir() {
				continue
			}
			payLoad = append(payLoad, fileInfo{
				Name:      info.Name(),
				Size:      info.Size(),
				Timestamp: info.ModTime().Unix(),
			})
		}
		resp.PayLoad = payLoad
	}
	if err != nil {
		resp.Msg = fmt.Sprint(err)
		resp.Status = -1
	}
	responseJson(&w, resp)
}

// 服务端工具-个人网盘-文件下载：文件删除/文件预览
func netDishFileOpeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OpeType  string `json:"opeType"`
		FileName string `json:"fileName"`
	}
	var err error
	var resp respStruct
	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Error("parse params fail: url=%s error=%v", r.URL, err)
			break
		}
		if req.OpeType != "delete" && req.OpeType != "download" {
			err = fmt.Errorf("unexpect opeType: req=%+v", req)
			break
		}
		targetPath := config.ServerConfig.StaticPath + req.FileName
		var info os.FileInfo
		info, err = os.Stat(targetPath)
		if err != nil {
			break
		}
		if info.IsDir() {
			err = fmt.Errorf("can't delete a floder: req=%+v", req)
			break
		}
		// 提供文件下载
		if req.OpeType == "download" {
			logs.Info("netdish download file: path=%s", targetPath)
			toolbox.ServerFile2(&w, targetPath)
			return
		}
		// 删除指定文件
		err = os.Remove(targetPath)
		logs.Info("netdish remove file: path=%s info=%+v error=%v", targetPath, info, err)
	}
	if err != nil {
		logs.Error("fail to handle: error=%v req=%+v", err, req)
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 服务端工具-个人网盘：文件上传
func netDishFileUploadHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp respStruct
	defer func() {
		if err != nil {
			logs.Error("handle upload fail: error=%v", err)
			resp.Status = -1
			resp.Msg = fmt.Sprint(err)
		}
		responseJson(&w, resp)
	}()

	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		logs.Error("Parse form fail: err=%v req=%+v", err, r)
		return
	}
	r.ParseForm()
	logs.Info("number of upload file: %d", len(r.MultipartForm.File))
	for _, files := range r.MultipartForm.File {
		for _, v := range files {
			var file multipart.File
			file, err = v.Open()
			if err != nil {
				logs.Error("open file fial:　err=%v", err)
				return
			}
			defer file.Close()
			filePath := config.ServerConfig.StaticPath + v.Filename
			_, err = os.Stat(filePath)
			if err == nil {
				err = fmt.Errorf("name already exist: %s", v.Filename)
				return
			}
			var cur *os.File
			cur, err = os.Create(filePath)
			if err != nil {
				logs.Error("create file fial:　err=%v path=%s", err, filePath)
				return
			}
			defer cur.Close()
			var size int64
			size, err = io.Copy(cur, file)
			if err != nil {
				logs.Error("save file fail: err=%v path=%s size=%d", err, filePath, size)
				return
			}
			logs.Info("Save file success: size=%d filePath=%s", size, filePath)
		}
		break
	}
	return
}

// 服务端配置-IP白名单配置:获取ip标记列表
func IPWhitelistHandler(w http.ResponseWriter, r *http.Request) {
	type payLoadStruct struct {
		Ip    string `json:"ip"`
		Tag   string `json:"tag"`
		Times int    `json:"times"`
	}
	payload := make([]payLoadStruct, 0)
	var resp respStruct
	for ip, tag := range IpMonitor.GetIpTag() {
		payload = append(payload, payLoadStruct{
			Ip:    ip,
			Tag:   tag,
			Times: IpMonitor.GetIpVisitTImes(ip),
		})
	}
	resp.PayLoad = payload
	responseJson(&w, resp)
}

// 服务端配置-IP白名单配置：新增修改标记\删除标记
func IPWhitelistOpeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OpeType string `json:"opeType"`
		IP      string `json:"ip"`
		Tag     string `json:"tag"`
	}
	var err error
	var resp respStruct

	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Warn("parse request fail: error=%+v", err)
			break
		}
		if req.OpeType != "update" && req.OpeType != "delete" {
			err = fmt.Errorf("unexpect opetype: %q", req.OpeType)
			break
		}
		if net.ParseIP(req.IP) == nil {
			err = fmt.Errorf("ip format not right: ip=%s", req.IP)
			break
		}
		if req.Tag == "" {
			err = fmt.Errorf("tag is null")
			break
		}
		if req.OpeType == "update" { // 新增或修改标签
			IpMonitor.UpdateIpTag(req.IP, req.Tag)
		} else {
			IpMonitor.DeleteIpTag(req.IP)
		}
		logs.Info("ip whitelist updated: req=%+v", req)
	}
	if err != nil {
		logs.Warn("handle ipwhitelist ope failed: error=%v req=%+v", err, req)
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}
