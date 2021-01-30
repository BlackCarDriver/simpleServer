package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"../toolbox"
	"github.com/astaxie/beego/logs"
)

var s2sMaster *ServiceMaster

func init() {
	s2sMaster = NewServiceMaster("secret")
	initCodeRunner()
	logs.Info("rpc init...")
}

// 注册RPC服务的http接口
func RegisterServiceHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterPackage
	var err error
	var resp struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &req)
		if err != nil {
			break
		}
		err = s2sMaster.Register(req)
		if err != nil {
			break
		}
		logs.Info("register service success: req=%+v", req)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	resp.Msg = "OK"
	jsResp, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", string(jsResp))
}
