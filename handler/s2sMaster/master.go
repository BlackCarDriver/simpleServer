package s2sMaster

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

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
	resp, err := http.Get(m.URL + "/ping")
	if err != nil {
		logs.Error("test member fail: error=%v member=%+v", *m)
		return false
	}
	if resp.StatusCode != 200 {
		logs.Warn("test member fail: resp=%+v", resp)
		return false
	}
	return true
}

//--------------------------------

// S2sServices代表一个服务
type S2sServices struct {
	mux     *sync.Mutex
	Name    string                // 服务名称
	counter map[string]int        // 用于统计每个节点提供的服务次数和成功率
	member  map[string]*s2sMember // url到节点的映射
}

// 创建一个新服务
func NewS2sService(name string) *S2sServices {
	if name == "" {
		panic("receive empty name when make new s2s services")
	}
	return &S2sServices{
		mux:     new(sync.Mutex),
		Name:    name,
		counter: make(map[string]int),
		member:  make(map[string]*s2sMember),
	}
}

// 新增一个成员
func (s *S2sServices) AddMenber(url, tag string) error {
	s.mux.Lock()
	defer s.mux.Unlock()

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
	oldMember, isExist := s.member[url]
	if isExist { // 此前注册过该节点
		if oldMember.Status == statusNormal {
			logs.Warn("a old normal member already exist: oldmember=%+v", oldMember)
			return nil
		}
		s.member[url].Status = statusNormal
		logs.Info("change a oldMember status, url=%s tag=%s", url, tag)
	} else { // 注册新节点
		s.member[url] = newMember
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
	secret   string
	counter  map[string]int //服务访问次数计数器
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
		md5Ctx := md5.New()
		md5Ctx.Write([]byte(m.secret + req.Name))
		md5Key := hex.EncodeToString(md5Ctx.Sum(nil))
		if md5Key != req.S2sKey {
			err = fmt.Errorf("s2sKey not right")
			logs.Debug("expect key=%s", md5Key)
			break
		}
		if service, isExist := m.services[req.Name]; !isExist { // 第一个节点
			newServices := NewS2sService(req.Name)
			if err = newServices.AddMenber(req.URL, req.Tag); err == nil {
				m.services[req.Name] = newServices
			}
		} else {
			err = service.AddMenber(req.URL, req.Tag)
		}
	}
	if err != nil {
		logs.Warn("register services failed: error=%v req=%+v", err, req)
		return err
	}
	logs.Info("register services success: req=%+v", req)
	return nil
}
