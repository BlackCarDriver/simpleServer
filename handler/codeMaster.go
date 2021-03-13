package handler

import (
	"baseService"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"../rpc"
	"github.com/astaxie/beego/logs"
)

// codeMaster的请求全部经过这里
func CodeMasterAPIHandler(w http.ResponseWriter, r *http.Request) {
	uri := strings.Trim(r.URL.Path, "/")
	logs.Debug("uri=%s", uri)
	switch uri {
	case "cmapi/createCode/debug":
		codeDebug(w, r)
	default:
		logs.Warn("unexpect uri: uri=%s", uri)
	}
}

// 接受测试代码和输入,返回运行结果
func codeDebug(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Code  string `json:"code"`
		Lang  string `json:"lang"`
		Input string `json:"input"`
	}
	var runResult struct {
		StdErr string `json:"stdErr"`
		StdOut string `json:"stdOut"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}
		logs.Debug("params=%+v len=%d,%d", params, len(params.Code), len(params.Input))
		// 检查参数
		if len(params.Code) > 20000 {
			err = fmt.Errorf("code too long: length=%d", len(params.Code))
			break
		}
		if len(params.Input) > 20000 {
			err = fmt.Errorf("input too long: length=%d", len(params.Input))
			break
		}
		// 运行
		var rpcResp *baseService.CommomResp
		if params.Lang == "CPP" {
			rpcResp, err = rpc.BuildCpp(params.Code, params.Input)
		} else if params.Lang == "C" {
			rpcResp, err = rpc.BuildC(params.Code, params.Input)
		} else if params.Lang == "GO" {
			rpcResp, err = rpc.BuildGo(params.Code, params.Input)
		} else {
			err = fmt.Errorf("unexpect params: lang=%q", params.Lang)
			break
		}

		// 返回值检查
		if err != nil {
			logs.Info("run code failed: error=%v params=%+v", err, params)
			break
		}
		if rpcResp.Status != 0 {
			err = fmt.Errorf("run failed: msg=%v", rpcResp.Msg)
			break
		}
		if rpcResp.Payload == nil {
			err = errors.New("payload is nil")
			break
		}
		err = json.Unmarshal(rpcResp.Payload, &runResult)
		if err != nil {
			logs.Error("unmarshal error: error=%v", err)
			break
		}
		resp.PayLoad = runResult
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}
