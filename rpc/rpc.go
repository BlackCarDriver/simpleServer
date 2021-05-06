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
	mStatusTesting = 1   // 检查中
	mStatusHangUp  = 2   // 手动暂停
	mStatusDown    = -1  // 主动下线
	mStatusDead    = -99 // 异常
)

func init() {
	s2sMaster = NewServiceMaster()
	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory = thrift.NewTBufferedTransportFactory(8192)

	logs.Info("rpc init...")
}

// 注册或注销RPC服务的http接口
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
		if req.Ope == "register" {
			err = s2sMaster.Register(req)
		} else if req.Ope == "unregister" {
			err = s2sMaster.UnRegister(req)
		} else {
			err = fmt.Errorf("unexpect params: ope=%q", req.Ope)
		}
		if err != nil {
			break
		}
		logs.Info("%s service success: req=%+v", req.Ope, req)
	}

	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	} else {
		resp.Msg = "OK"
		w.WriteHeader(207) // 约定以297作为注册成功的标志
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

// 设置特定节点的状态
// status:[remove|restore|hang]
func SetNodeStatus(s2sName string, addr string, status string) error {
	s2sMaster.mux.Lock()
	defer s2sMaster.mux.Unlock()
	var err error
	for loop := true; loop; loop = false {
		if s2sName == "" || addr == "" || status == "" {
			err = fmt.Errorf("unexpect params")
			break
		}
		service, isExist := s2sMaster.services[s2sName]
		if !isExist {
			err = fmt.Errorf("service not found: s2sName=%s", s2sName)
			break
		}
		if len(service.member) == 0 {
			err = fmt.Errorf("no member in it service: s2sName=%s", s2sName)
			break
		}
		var member *s2sMember
		member, err = service.GetMemberByAddr(addr)
		if err != nil {
			break
		}
		switch status {
		case "hang":
			member.UpdateStatus(mStatusHangUp)
		case "restore":
			member.UpdateStatus(mStatusNormal)
		case "remove":
			err = service.RemoveMemberByAddr(addr)
		default:
			err = fmt.Errorf("unknow params: status=%s", status)
		}
	}
	if err != nil {
		logs.Error("update member stauts failed: error=%v", err)
		return err
	}
	logs.Info("update member status success: s2sName=%s addr=%s", s2sName, addr)
	return nil
}

// 生成所有节点的节点描述，用于持久化节点信息
func GetAllNodeMsg() []RegisterPackage {
	res := make([]RegisterPackage, 0)
	overViewMsg := GetRpcOverview()
	if len(overViewMsg) == 0 {
		return res
	}
	for _, service := range overViewMsg {
		for _, member := range service.Members {
			tmpDesc := RegisterPackage{
				Name:   service.Name,
				URL:    member.URL,
				Tag:    member.Tag,
				Ope:    "register",
				S2sKey: getS2sKey(service.Name, member.URL),
			}
			res = append(res, tmpDesc)
		}
	}
	logs.Info("res=%+v", res)
	return res
}

// 根据之前保存的节点信息还原节点状态
func RestoreAllNode(nodes []RegisterPackage) {
	logs.Info("RestoreAllNode called: nodes numbers=%d", len(nodes))
	for _, node := range nodes {
		err := s2sMaster.Register(node)
		if err != nil {
			logs.Error("restore node failed: error=%v node=%+v", err, node)
		} else {
			logs.Info("restore node success: node=%+v", node)
		}
	}
}
