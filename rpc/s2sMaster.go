package rpc

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"baseService"

	"../config"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
)

const (
	statusNormal int = 0
	statusBroken int = -1
)

// --------------- 请求 ------------------

// 节点描述
type RegisterPackage struct {
	Name   string `json:"name"` // 服务名
	URL    string `json:"url"`
	S2sKey string `json:"s2sKey"`
	Tag    string `json:"tag"`
}

// 验证节点信息
func (r *RegisterPackage) Check() error {
	if r.URL == "" || r.Name == "" {
		return errors.New("empty url or name")
	}
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(config.ServerConfig.S2SSecret + r.Name + r.URL))
	md5Key := hex.EncodeToString(md5Ctx.Sum(nil))
	if md5Key != r.S2sKey {
		logs.Info("s2s key not right")
		logs.Debug("expect key=%s", md5Key)
		return errors.New("s2s key not right")
	}
	logs.Info("check s2s key pass")
	return nil
}

// -------------- 节点 --------------

// s2sMember代表一个提供服务的节点
type s2sMember struct {
	URL           string
	Tag           string
	Status        int
	Counter       int64 // 提供服务的总次数
	Failed        int64 // 服务失败的次数
	RegTimestamp  int64 // 注册时间
	LastTimestamp int64 // 上次提供服务的时间
	mux           *sync.Mutex
}

// 测试节点是否正常提供服务
func (m *s2sMember) Test() bool {
	if m == nil {
		logs.Error("calling Test() of a nil pointer....")
		return false
	}
	var transport thrift.TTransport
	var err error
	transport, err = thrift.NewTSocket(m.URL)
	if err != nil {
		logs.Error("test fail, can not create socket: error=%v url=%s", err, m.URL)
		return false
	}
	transport, err = transportFactory.GetTransport(transport)
	if err != nil {
		logs.Error("test fail, can not create transport: error=%v", err)
		return false
	}
	defer transport.Close()
	if err = transport.Open(); err != nil {
		logs.Error("test fail, transport can not open: error=%v", err)
		return false
	}
	client := baseService.NewBaseServiceClient(
		thrift.NewTStandardClient(
			protocolFactory.GetProtocol(transport),
			protocolFactory.GetProtocol(transport)))
	res, err := client.Ping(context.Background(), fmt.Sprint(m.LastTimestamp))
	if err != nil {
		logs.Error("test fail, can not called ping: error=%v", err)
		return false
	}
	if res == "" {
		logs.Error("Ping() return empty result")
		return false
	}
	return true
}

// 更新统计数据
func (m *s2sMember) UpdateCounter(success bool) {
	if m == nil {
		logs.Error("calling UpdateCounter() of a nil pointer....")
		return
	}
	go func() {
		m.mux.Lock()
		defer m.mux.Unlock()
		m.LastTimestamp = time.Now().Unix()
		m.Counter++
		if !success {
			m.Failed++
		}
	}()
}

// 更新节点状态
func (m *s2sMember) UpdateStatus(status int) {
	if m == nil {
		logs.Error("calling UpdateStatus() of a nil pointer....")
		return
	}
	go func() {
		m.mux.Lock()
		defer m.mux.Unlock()
		m.Status = status
	}()
}

// 测试可能存在问题的节点并更新状态
func (m *s2sMember) GoReport() {
	if m == nil {
		logs.Error("calling Report() of a nil pointer....")
		return
	}
	if m.Status != mStatusNormal {
		return
	}
	m.mux.Lock()
	defer m.mux.Unlock()
	m.Status = mStatusTesting

	logs.Info("start checking a s2sMember: node=%+v", *m)
	testTimes := 5
	for i := 1; i <= testTimes; i++ {
		if m.Test() {
			logs.Info("test success after %d try...", i)
			m.Status = mStatusNormal
			return
		}
		logs.Info("test failed × %d", i)
		time.Sleep(2 * time.Second)
	}
	// 测试失败
	// TODO: 触发告警
	m.Status = mStatusDead
	logs.Warning("it s2sMember might destroy: node=%+v", *m)
	return
}

//-------------- 服务 ---------------

// S2sServices代表一个服务
type S2sServices struct {
	addMemberMux *sync.Mutex
	Name         string       // 服务名称
	member       []*s2sMember // 节点列表
	counter      int64        // 请求次数
}

// 创建一个新服务
func NewS2sService(name string) *S2sServices {
	if name == "" {
		panic("receive empty name when make new s2s services")
	}
	return &S2sServices{
		addMemberMux: new(sync.Mutex),
		Name:         name,
		counter:      0,
		member:       make([]*s2sMember, 0),
	}
}

// 新增一个节点
func (s *S2sServices) AddMenber(req RegisterPackage) error {
	s.addMemberMux.Lock()
	defer s.addMemberMux.Unlock()
	var err error
	if err = req.Check(); err != nil {
		return err
	}
	newMember := &s2sMember{
		URL:           req.URL,
		Tag:           req.Tag,
		Status:        statusNormal,
		Counter:       0,
		Failed:        0,
		LastTimestamp: 0,
		RegTimestamp:  time.Now().Unix(),
		mux:           new(sync.Mutex),
	}
	if !newMember.Test() {
		logs.Warn("add member failed: test not pass")
		return fmt.Errorf("test not pass")
	}
	var oldMember *s2sMember
	isExist := false
	index := 0
	for index := 0; index < len(s.member); index++ {
		if s.member[index].URL == req.URL {
			oldMember = s.member[index]
			isExist = true
			break
		}
	}
	if isExist { // 此前注册过该节点
		if oldMember.Status == statusNormal {
			logs.Warn("a old normal member already exist: oldmember=%+v", oldMember)
			return nil
		}
		s.member[index].Status = statusNormal
		logs.Info("change a oldMember status: req=%+v", req)
	} else { // 注册新节点
		s.member = append(s.member, newMember)
		logs.Info("add a new member: req=%+v", req)
	}
	return nil
}

// 移除一个节点
func (s *S2sServices) RemoveMember(req RegisterPackage) bool {
	s.addMemberMux.Lock()
	defer s.addMemberMux.Unlock()
	if err := req.Check; err != nil {
		logs.Warning("skip remove for check s2sKey failed")
		return false
	}
	index := -1
	for i := 0; i < len(s.member); i++ {
		if s.member[i].URL == req.URL {
			index = i
			break
		}
	}
	if index < 0 {
		logs.Info("no member found to delete: request=%+v", req)
		return false
	}
	s.member = append(s.member[0:index], s.member[index+1:]...)
	return true
}

//--------------- 管理员 -----------

// 服务管理员
type ServiceMaster struct {
	mux      *sync.Mutex
	secret   string // secret用于计算正确的s2sKey
	services map[string]*S2sServices
}

// 创建一个服务管理员
func NewServiceMaster() *ServiceMaster {
	return &ServiceMaster{
		mux:      new(sync.Mutex),
		services: make(map[string]*S2sServices), //服务名到服务的映射
	}
}

// 处理一个服务提供者的注册服务请求
func (m *ServiceMaster) Register(req RegisterPackage) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	var err error
	for loop := true; loop; loop = false {
		if err = req.Check(); err != nil {
			break
		}
		// 服务注册
		if service, isExist := m.services[req.Name]; !isExist { // 第一个节点,新建服务
			newServices := NewS2sService(req.Name)
			if err = newServices.AddMenber(req); err == nil {
				m.services[req.Name] = newServices
			}
		} else {
			err = service.AddMenber(req) // 非首个节点,加入到服务队列
		}
	}
	if err != nil {
		logs.Warn("register services failed: error=%v req=%+v", err, req)
		return err
	}
	logs.Info("register services success: req=%+v", req)
	return nil
}

// 处理一个服务下线请求 TODO:新增http接口调用
func (m *ServiceMaster) UnRegister(req RegisterPackage) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	var err error
	if err = req.Check(); err != nil {
		logs.Warning("check not pass: error=%v", err)
		return nil
	}
	service, exist := m.services[req.Name]
	if !exist {
		logs.Warning("service not exist: request=%+v", req)
		return nil
	}
	result := service.RemoveMember(req)
	logs.Info("remove member result: request=%+v result=%b", req, result)
	return nil
}

// 根据服务的名称获取一个提供服务的节点
func (m *ServiceMaster) GetNodeByServiceName(name string) (node *s2sMember, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	service, isExist := m.services[name]
	if !isExist {
		return nil, fmt.Errorf("service not found: name=%s", name)
	}
	if len(service.member) == 0 {
		return nil, fmt.Errorf("no member in it service: service=%+v", service)
	}
	targetIndex := service.counter % int64(len(service.member))
	service.counter++
	// 若分配的节点处于灰度状态,则返回下一个正常的节点
	if service.member[targetIndex].Status != mStatusNormal {
		logs.Info("turn to a unnormal node: member=%+v", *service)
		var normalIndex int64 = -1
		for i := targetIndex; i <= targetIndex+int64(len(m.services)); i++ {
			next := i % int64(len(service.member))
			if service.member[next].Status == mStatusNormal {
				normalIndex = int64(next)
				break
			}
		}
		if normalIndex < 0 {
			logs.Warning("no member is normal: totalNumber=%d s2sName=%s", len(service.member), name)
			return nil, fmt.Errorf("no member in normal stuats: s2sName=%+v", name)
		}
		targetIndex = normalIndex
	}
	targetMember := service.member[targetIndex]
	return targetMember, nil
}
