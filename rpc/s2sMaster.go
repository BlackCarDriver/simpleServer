package rpc

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"baseService"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
)

const (
	statusNormal int = 0
	statusBroken int = -1
)

// s2sMember代表一个提供服务的节点
type s2sMember struct {
	URL           string
	Tag           string
	Status        int
	RegTimestamp  int64 // 注册时间
	LastTimestamp int64 // 上次提供服务的时间
}

// 测试节点是否正常提供服务
func (m *s2sMember) Test() bool {
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
	_, err = client.Ping(context.Background())
	if err != nil {
		logs.Error("test fail, can not called ping: error=%v", err)
		return false
	}
	return true
}

//--------------------------------

// S2sServices代表一个服务
type S2sServices struct {
	addMemberMux *sync.Mutex
	Name         string       // 服务名称
	counter      int          // 请求次数
	member       []*s2sMember // 节点列表
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

// 新增一个成员
func (s *S2sServices) AddMenber(url, tag string) error {
	s.addMemberMux.Lock()
	defer s.addMemberMux.Unlock()

	if url == "" || tag == "" {
		panic("receive empty url or tag when add new member")
	}
	newMember := &s2sMember{
		URL:          url,
		Tag:          tag,
		Status:       statusNormal,
		RegTimestamp: time.Now().Unix(),
	}
	if !newMember.Test() {
		logs.Warn("add member failed: test not pass")
		return fmt.Errorf("test not pass")
	}
	var oldMember *s2sMember
	isExist := false
	index := 0
	for index := 0; index < len(s.member); index++ {
		if s.member[index].URL == url {
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
		logs.Info("change a oldMember status, url=%s tag=%s", url, tag)
	} else { // 注册新节点
		s.member = append(s.member, newMember)
		logs.Info("add a new member, url=%s tag=%s", url, tag)
	}
	return nil
}

//----------------------------------------------------

// 注册服务时的请求包
type RegisterPackage struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	S2sKey string `json:"s2sKey"`
	Tag    string `json:"tag"`
}

// 服务管理员
type ServiceMaster struct {
	mux      *sync.Mutex
	secret   string         // 用于防治恶意注册服务
	counter  map[string]int // 服务访问次数计数器
	services map[string]*S2sServices
}

// 创建一个服务管理员
func NewServiceMaster(secret string) *ServiceMaster {
	return &ServiceMaster{
		mux:      new(sync.Mutex),
		secret:   secret,
		counter:  make(map[string]int),          // 统计每个service提供服务的次数
		services: make(map[string]*S2sServices), //服务名到服务的映射
	}
}

// 处理一个服务提供者的注册服务请求
func (m *ServiceMaster) Register(req RegisterPackage) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	var err error
	for loop := true; loop; loop = false {
		if req.Name == "" || req.S2sKey == "" {
			err = fmt.Errorf("empty name or s2skey")
			break
		}
		// s2sKey验证
		md5Ctx := md5.New()
		md5Ctx.Write([]byte(m.secret + req.Name))
		md5Key := hex.EncodeToString(md5Ctx.Sum(nil))
		if md5Key != req.S2sKey {
			err = fmt.Errorf("s2sKey not right")
			logs.Debug("expect key=%s", md5Key)
			break
		}
		// 服务注册
		if service, isExist := m.services[req.Name]; !isExist { // 第一个节点,新建服务
			newServices := NewS2sService(req.Name)
			if err = newServices.AddMenber(req.URL, req.Tag); err == nil {
				m.services[req.Name] = newServices
			}
		} else {
			err = service.AddMenber(req.URL, req.Tag) // 非首个节点,加入到服务队列
		}
	}
	if err != nil {
		logs.Warn("register services failed: error=%v req=%+v", err, req)
		return err
	}
	logs.Info("register services success: req=%+v", req)
	return nil
}

// 根据服务的名称获取一个节点的地址
func (m *ServiceMaster) GetUrlByServiceName(name string) (url string, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	service, isExist := m.services[name]
	if !isExist {
		return "", fmt.Errorf("service not found: name=%s", name)
	}
	if len(service.member) == 0 {
		return "", fmt.Errorf("no member in it service: service=%+v", service)
	}
	targetIndex := service.counter % len(service.member)
	service.counter++
	return service.member[targetIndex].URL, nil
}

//----------------- Temp TestCode --------------------

func (m *ServiceMaster) RegisterTest(req RegisterPackage) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	var err error
	for loop := true; loop; loop = false {
		if req.Name == "" || req.S2sKey == "" {
			err = fmt.Errorf("empty name or s2skey")
			break
		}
		// 服务注册
		if service, isExist := m.services[req.Name]; !isExist { // 第一个节点,新建服务
			newServices := NewS2sService(req.Name)
			if err = newServices.AddMenber(req.URL, req.Tag); err == nil {
				m.services[req.Name] = newServices
			}
		} else {
			err = service.AddMenber(req.URL, req.Tag) // 非首个节点,加入到服务队列
		}
	}
	if err != nil {
		logs.Warn("register services failed: error=%v req=%+v", err, req)
		return err
	}
	logs.Info("register services success: req=%+v", req)
	return nil
}

func test() {
	req1 := RegisterPackage{
		Name:   "test",
		URL:    "url1",
		S2sKey: "...",
		Tag:    "sss",
	}
	req2 := RegisterPackage{
		Name:   "test",
		URL:    "url2",
		S2sKey: "...",
		Tag:    "sss",
	}
	req3 := RegisterPackage{
		Name:   "test",
		URL:    "url3",
		S2sKey: "...",
		Tag:    "sss",
	}
	err := s2sMaster.RegisterTest(req1)
	err = s2sMaster.RegisterTest(req2)
	err = s2sMaster.RegisterTest(req3)
	logs.Info("error=%v", err)

	url, err := s2sMaster.GetUrlByServiceName("test")
	logs.Info("url=%s error=%v", url, err)

	url, err = s2sMaster.GetUrlByServiceName("test")
	logs.Info("url=%s error=%v", url, err)
	url, err = s2sMaster.GetUrlByServiceName("test")
	logs.Info("url=%s error=%v", url, err)
	url, err = s2sMaster.GetUrlByServiceName("test")
	logs.Info("url=%s error=%v", url, err)
	url, err = s2sMaster.GetUrlByServiceName("test")
	logs.Info("url=%s error=%v", url, err)
	url, err = s2sMaster.GetUrlByServiceName("test")
	logs.Info("url=%s error=%v", url, err)
	url, err = s2sMaster.GetUrlByServiceName("test2")
	logs.Info("url=%s error=%v", url, err)
	url, err = s2sMaster.GetUrlByServiceName("test2")
	logs.Info("url=%s error=%v", url, err)
}
