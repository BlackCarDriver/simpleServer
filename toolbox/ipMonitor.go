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
	IpBlackList    = make(map[string]bool) // 访问黑名单
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
	if IpBlackList[ip] {
		return false
	}
	return IpWhiteList[ip]
}

// 判断一个请求的IP地址是否在黑名单内
func IsInBlackList(r *http.Request) bool {
	ip, _ := GetIpAndPort(r)
	return IpBlackList[ip]
}

// 将IP加入到白名单
func AddWhiteList(ip string) {
	IpBlackList[ip] = false
	IpWhiteList[ip] = true
}

// 将IP加入到黑名单
func AddBlackList(ip string) {
	IpWhiteList[ip] = false
	IpBlackList[ip] = true
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

// 删减IpHistory的记录，次数低于n的清除,返回清楚的数量
func ClearIpHistoryN(n int) int {
	count := 0
	for k, v := range IpHistory {
		if v < n {
			delete(IpHistory, k)
			count++
		}
	}
	return count
}

// 查询黑白名单列表
func GetBlackWhiteList() string {
	res := "=========== White List ============\n"
	for ip, exist := range IpWhiteList {
		if exist {
			res += ip + "\n"
		}
	}
	res += "=========== Black List ============\n"
	for ip, exist := range IpBlackList {
		if exist {
			res += ip + "\n"
		}
	}
	return res
}
