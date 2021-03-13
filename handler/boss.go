package handler

import (
	"baseService"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"os/exec"
	"net/http"
	"os"
	"regexp"
	"strings"

	"../toolbox"

	"errors"

	"../config"
	"../rpc"

	"github.com/astaxie/beego/logs"
)

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
		ipWhitelistHandler(w, r)
	case "bsapi/manage/ipWhiteList/ope":
		ipWhitelistOpeHandler(w, r)
	case "bsapi/manage/systemSetting/ope":
		systemSettingHandler(w, r)
	case "bsapi/manage/systemSetting/status":
		getSystemStting(w, r)
	case "bsapi/monitor/rpc/overview":
		getRpcOverview(w, r)
	case "bsapi/monitor/rpc/ope":
		rpcManager(w, r)
	case "bsapi/monitor/rpc/test":
		testRPCInterface(w, r)
	case "bsapi/monitor/sysStateInfo":
		getSysState(w, r)
	case "bsapi/monitor/getServerLog":
		getServerLog(w, r)
	default:
		NotFoundHandler(w, r)
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
func ipWhitelistHandler(w http.ResponseWriter, r *http.Request) {
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
func ipWhitelistOpeHandler(w http.ResponseWriter, r *http.Request) {
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
		logs.Warn("handle ipwhitelist ope failed: error=%v url=%v", err, r.URL)
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 服务端配置-开关设置: 获取当前配置
func getSystemStting(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CallDriverEmail bool `json:"callDriverEmail"`
		AlertEmail bool `json:"alertEmail"`
	}

	payload.CallDriverEmail = sendCallDriverEmail
	payload.AlertEmail = sendAlertEmail

	var resp = respStruct{
		Status: 0,
		Msg: "",
		PayLoad: payload,
	}
	responseJson(&w, resp)
}

// 服务端配置-开关设置: 开关控制请求
func systemSettingHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tag string `json:"tag"`
		Params string `json:"params"`
	}
	var resp respStruct
	var err error
	for loop:=true; loop; loop=false {
		err = toolbox.MustQueryFromRequest(r, &req)
		if err!=nil {
			logs.Error("parse request failed: error=%v", err)
			break
		}
		logs.Info("params=%+v", req)
		switch req.Tag {
		case "systemUpdate": // 更新和重启系统
			cmd := exec.Command("bash", config.ServerConfig.RestartBashPath)
			err = cmd.Run()
			if err != nil {
				logs.Error("Execute Command failed:" + err.Error())
				return
			}else{
				logs.Info("exec success...")
			}
		case "callDriverEmail": // 更新配置参数sendCallDriverEmail
			if req.Params == "true" {
				sendCallDriverEmail = true
			}else if req.Params == "false" {
				sendCallDriverEmail = false
			}else{
				err = fmt.Errorf("unexpect params: params=%q", req.Params)
			}
		case "alertEmail":
			if req.Params == "true" { // 更新配置参数sendAlertEmail
				sendAlertEmail = true
			}else if req.Params == "false" {
				sendAlertEmail = false
			}else{
				err = fmt.Errorf("unexpect params: params=%q", req.Params)
			}
		default:
			err = fmt.Errorf("unexpect params: req=%+v", req)
		}
	}
	if err != nil {
		logs.Error("handle system setting request faield: error=%v url=%+v", err, r.URL)
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 服务端监控-RPC服务状况：查看状况
func getRpcOverview(w http.ResponseWriter, r *http.Request) {
	var resp respStruct
	resp.PayLoad = rpc.GetRpcOverview()
	responseJson(&w, resp)
}

// 服务端监控-RPC服务状况：设置节点状态
func rpcManager(w http.ResponseWriter, r *http.Request) {
	var req struct {
		S2SName string `json:"s2sName"`
		Addr    string `json:"addr"`
		Ope     string `json:"ope"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &req)
		if err != nil {
			logs.Warn("parse request fail: error=%+v", err)
			break
		}
		logs.Info("params=%+v", req)
		err = rpc.SetNodeStatus(req.S2SName, req.Addr, req.Ope)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		resp.Msg = "OK"
	}
	responseJson(&w, resp)
}

// 服务端监控-系统状态：查看最近一小时或一周的系统状况
// get请求,参数:type=[long\short]
func getSysState(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	params := r.FormValue("type")
	var resp respStruct
	if params == "short" {
		resp.PayLoad = toolbox.SysStateInfoShort.Report()
	} else if params == "long" {
		resp.PayLoad = toolbox.SysStateInfoLong.Report()
	} else if params == "realTime" {
		resp.PayLoad = toolbox.GetState()
	} else {
		resp.Status = -1
		resp.Msg = "unexpect params"
		w.WriteHeader(http.StatusBadRequest)
		logs.Warn("unexpect params: params=%s", params)
	}
	responseJson(&w, resp)
}

// 服务端监控-服务端日志: 日志搜索
func getServerLog(w http.ResponseWriter, r *http.Request) {
	var err error
	var file *os.File
	var resp respStruct
	for loop := true; loop; loop = false {
		r.ParseForm()
		target := r.FormValue("target")
		logs.Info("target=%s", target)
		file, err = os.Open(config.ServerConfig.LogPath)
		if err != nil {
			logs.Error("Open logfile fall: error=%v", err)
			break
		}
		defer file.Close()
		buf := bufio.NewReader(file)
		var bytes []byte
		bytes, err = ioutil.ReadAll(buf)
		if err != nil {
			logs.Error("read file failed: error=%v", err)
			break
		}

		if target == "" {
			resp.PayLoad = strings.Split(string(bytes), "\n")
			break
		}
		var reg *regexp.Regexp
		regStr := fmt.Sprintf(`\B.*%s.*\n`, target)
		logs.Debug("regstr=%s", regStr)
		reg, err = regexp.Compile(regStr)
		if err != nil {
			logs.Warn("Compile regexp fail: regStr=%s error=%v", regStr, err)
			break
		}
		res := reg.FindAllString(string(bytes), -1)
		resp.PayLoad = res
		logs.Debug("payload length=%d", len(res))
	}

	if err != nil {
		resp.PayLoad = fmt.Sprint(err)
		resp.Status = -1
		w.WriteHeader(http.StatusInternalServerError)
	}
	responseJson(&w, resp)
}

// 服务端监控-RPC接口测试
func testRPCInterface(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp respStruct
	rpcResp := &baseService.CommomResp{}
	var req struct {
		Tag  string `json:"tag"`
		Str1 string `json:"str1"`
		Str2 string `json:"str2"`
		Str3 string `json:"str3"`
		Num1 int64  `json:"num1"`
		Num2 int64  `json:"num2"`
		Num3 int64  `json:"num3"`
	}
	var runResult struct {
		StdErr string `json:"stdErr"`
		StdOut string `json:"stdOut"`
	}
	for loop := true; loop; loop = false {
		if r.Method == http.MethodGet {
			err = fmt.Errorf("unexpect method: %s", r.Method)
			break
		}
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&req)
		if err != nil {
			break
		}
		switch req.Tag {
		case "codeRunner_buildGo":
			rpcResp, err = rpc.BuildGo(req.Str1, req.Str2)
		case "codeRunner_buildCpp":
			rpcResp, err = rpc.BuildCpp(req.Str1, req.Str2)
		case "codeRunner_buildC":
			rpcResp, err = rpc.BuildC(req.Str1, req.Str2)
		case "codeRunner_run":
			rpcResp, err = rpc.Run(req.Str1, req.Str2, req.Str3)
		default:
			err = fmt.Errorf("unexpect tag")
			break
		}
		if err != nil {
			break
		}
		if rpcResp.Status != 0 {
			err = fmt.Errorf("%s return error: error=%v", rpcResp.Msg, err)
			break
		}
		if rpcResp.Payload == nil {
			err = errors.New("rpc response payload is null")
			break
		}
		err = json.Unmarshal(rpcResp.Payload, &runResult)
		if err != nil {
			logs.Error("json unmarshal failed: error=%v", err)
			break
		}
		logs.Info("runResult=%+v", runResult)
		resp.Msg = rpcResp.Msg // codeHash
		resp.PayLoad = runResult
	}
	if err != nil {
		logs.Warn("test interface failed: tag=%s params=%+v error=%v", req.Tag, req, err)
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}
