package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"../toolbox"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
)

var s2sMaster *ServiceMaster

var protocolFactory thrift.TProtocolFactory
var transportFactory thrift.TTransportFactory

const (
	codeRunnerS2SName = "codeRunner"
)

const (
	mStatusNormal  = 0
	mStatusTesting = 99
	mStatusDead    = -99
)

func init() {
	s2sMaster = NewServiceMaster()
	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory = thrift.NewTBufferedTransportFactory(8192)
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
	resp.Msg = "OK"
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	jsResp, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s", string(jsResp))
}

// 返回一个10分钟自动关闭的context
func GetDefaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Minute*10)
}

//----------------- 统计数据结构 --------------------

// 节点相关统计数据
type overViewMember struct {
	Tag      string `json:"tag"`
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Counter  int64  `json:"counter"`
	Failed   int64  `json:"failed"`
	RegTime  int64  `json:"regTime"`
	LastTime int64  `json:"lastTime"`
}

// 服务相关统计数据
type overViewService struct {
	Name    string           `json:"name"`
	Members []overViewMember `json:"members"`
	Counter int64            `json:"counter"`
}

// 获取rpc服务统计数据
func GetRpcOverview() []overViewService {
	var list []overViewService = make([]overViewService, 0)
	for _, service := range s2sMaster.services {
		tmpMembers := make([]overViewMember, 0)
		var totalCounter int64
		for _, member := range service.member {
			tmpMembers = append(tmpMembers, overViewMember{
				Tag:      member.Tag,
				URL:      member.URL,
				Status:   member.Status,
				Counter:  member.Counter,
				RegTime:  member.RegTimestamp,
				LastTime: member.LastTimestamp,
				Failed:   member.Failed,
			})
			totalCounter += member.Counter
		}
		list = append(list, overViewService{
			Name:    service.Name,
			Counter: totalCounter,
			Members: tmpMembers,
		})
	}
	return list
}
