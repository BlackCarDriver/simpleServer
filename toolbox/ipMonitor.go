package toolbox

import (
	"fmt"
	"net/http"

	"github.com/astaxie/beego/logs"
)

// IPMonitor 监控和记录、统计请求的IP数据

type IPMonitor struct {
	ipTag     map[string]string // ip标记
	ipHistory map[string]int    // ip访问次数
}

func MakeIpMonitor() *IPMonitor {
	return &IPMonitor{
		ipTag:     make(map[string]string),
		ipHistory: make(map[string]int),
	}
}

// ------------- Update ----------------------------

// 更新某个ip的标记
func (m *IPMonitor) UpdateIpTag(ip string, tag string) {
	m.ipTag[ip] = tag
}

// 更新所有标记
func (m *IPMonitor) UpdateAllIpTag(newIpTag map[string]string) {
	if newIpTag == nil {
		logs.Error("newIpTag is nil")
		return
	}
	m.ipTag = newIpTag
}

// 删除IP标记
func (m *IPMonitor) DeleteIpTag(ip string) {
	delete(m.ipTag, ip)
}

// 删减ipHistory的记录，次数低于n的清除,返回清除的数量
func (m *IPMonitor) ClearipHistoryN(n int) int {
	count := 0
	for k, v := range m.ipHistory {
		if v < n {
			delete(m.ipHistory, k)
			count++
		}
	}
	return count
}

// 获取IP访问次数并自增1
func (m *IPMonitor) GetAndAddIpVisitTimes(ip string) int {
	m.ipHistory[ip]++
	return m.ipHistory[ip]
}

// 获取IP访问的次数
func (m *IPMonitor) GetIpVisitTImes(ip string) int {
	return m.ipHistory[ip]
}

// -------------  Query --------------------

// 判断一个请求的IP是否白名单(有标记则为白名单)
func (m *IPMonitor) IsInWhiteList(r *http.Request) bool {
	ip, _ := GetIpAndPort(r)
	return len(m.ipTag[ip]) > 0
}

// 获取访问IP的标记
func (m *IPMonitor) QueryIpTag(r *http.Request) string {
	ip, _ := GetIpAndPort(r)
	return m.ipTag[ip]
}

// 获取m.ipTag
func (m *IPMonitor) GetIpTag() map[string]string {
	return m.ipTag
}

// ------------ Report ---------------

// 查询ip标记汇总信息
func (m *IPMonitor) GetBlackWhiteList() string {
	res := "=========== IP Tag ============\n"
	for ip, tag := range m.ipTag {
		res += fmt.Sprintf("%s -- %s \n", ip, tag)
	}
	return res
}

// 获取IP访问次数汇总信息
func (m *IPMonitor) GetStatic() string {
	record := ""
	for ip, times := range m.ipHistory {
		record += fmt.Sprintf("%s  %d \n", ip, times)
	}
	return record
}
