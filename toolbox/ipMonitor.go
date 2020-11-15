package toolbox

import (
	"fmt"
	"net/http"
	"sync"
)

// 监控和记录、统计请求的IP数据

var (
	RequestCounter = 0                     // 访问次数计数器
	IpWhiteList    = make(map[string]bool) // 访问白名单
	IpHistory      = make(map[string]int)  // ip访问记录和次数
)

// 获取统计数据
func GetStatic() string {
	record := ""
	for ip, times := range IpHistory {
		record += fmt.Sprintf("%s   %d \n", ip, times)
	}
	return record
}

// 判断一个请求的IP是否在白名单内
func IsInWhiteList(r *http.Request) bool {
	ip, _ := GetIpAndPort(r)
	_, exist := IpHistory[ip]
	return exist
}

// 将IP加入到白名单
func AddWhiteList(ip string) {
	IpWhiteList[ip] = true
}

var readMux *sync.Mutex = new(sync.Mutex)

// 获取IP访问次数并自增1
func GetAndAddIpVisitTimes(ip string) int {
	times := IpHistory[ip]
	readMux.Lock()
	IpHistory[ip]++
	readMux.Unlock()
	return times
}
