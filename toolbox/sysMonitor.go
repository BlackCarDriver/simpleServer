package toolbox

import (
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

var SysStateInfoShort *cycleList // 记录最近1小时的系统负载,每10秒采集一次
var SysStateInfoLong *cycleList  // 记录最近一周的系统负载，每30分钟采集一次

func initSysMonitor() {
	// if config.ServerConfig.IsTest {
	// 	return
	// }
	SysStateInfoShort = NewCycleList(360)
	SysStateInfoLong = NewCycleList(336)
	go func() {
		for range time.Tick(10 * time.Second) {
			SysStateInfoShort.Record(GetState())
		}
	}()
	go func() {
		for range time.Tick(30 * time.Minute) {
			var lastHourStat sysState
			lastHourRecord := SysStateInfoShort.Report()
			if len(lastHourRecord) == 0 {
				SysStateInfoLong.Record(GetState())
				continue
			}
			for _, v := range lastHourRecord {
				lastHourStat.AvgLoad += v.(*sysState).AvgLoad
				lastHourStat.CpuPercent += v.(*sysState).CpuPercent
				lastHourStat.DishPercent += v.(*sysState).DishPercent
				lastHourStat.VMUsedPercent += v.(*sysState).VMUsedPercent
				lastHourStat.ProcsTotal += v.(*sysState).ProcsTotal
			}
			lastHourStat.AvgLoad /= float64(len(lastHourRecord))
			lastHourStat.CpuPercent /= float64(len(lastHourRecord))
			lastHourStat.DishPercent /= float64(len(lastHourRecord))
			lastHourStat.VMUsedPercent /= float64(len(lastHourRecord))
			lastHourStat.ProcsTotal /= len(lastHourRecord)
			lastHourStat.Timestamp = time.Now().Unix()
			logs.Info("record hour state: result=%+v", lastHourStat)
			SysStateInfoLong.Record(&lastHourStat)
		}
	}()
	logs.Info("sysMonitor init success...")
}

// --------------- 负载信息 --------------------

// 系统负载信息
type sysState struct {
	Timestamp     int64   `json:"timestamp"`
	CpuPercent    float64 `json:"cpuPercent"`    // cpu平均负载
	DishPercent   float64 `json:"dishPercent"`   // 磁盘占用量
	AvgLoad       float64 `json:"avgLoad"`       // 最近1分钟的平均负载
	ProcsTotal    int     `json:"procsTotal"`    // 进程总数
	VMUsedPercent float64 `json:"vmUsedPercent"` // 虚拟内存使用量
}

func GetState() *sysState {
	var state sysState
	state.Timestamp = time.Now().Unix()
	CPUPercent, err := cpu.Percent(time.Second*0, false)
	if err != nil || len(CPUPercent) == 0 {
		logs.Warn("get cpu state error: %v", err)
	}
	state.CpuPercent = CPUPercent[0]
	usageStat, err := disk.Usage("/")
	if err != nil {
		logs.Warn("get dish state error: %v", err)
	}
	state.DishPercent = usageStat.UsedPercent
	avg, err := load.Avg()
	if err != nil {
		logs.Warn("get avg state error: %v", err)
	}
	state.AvgLoad = avg.Load1
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		logs.Warn("get mem state error: %v", err)
	}
	var pids []int32
	pids, err = process.Pids()
	if err != nil {
		logs.Warn("get pids error: %v", err)
	}
	state.ProcsTotal = len(pids)
	state.VMUsedPercent = virtualMemory.UsedPercent
	return &state
}

// --------------- 循环队列 -----------------

type cycleList struct {
	data  []interface{}
	ptr   int
	size  int
	vaild int
	mux   *sync.Mutex
}

func NewCycleList(size int) *cycleList {
	if size < 0 {
		size = 0
	}
	return &cycleList{
		ptr:   0,
		size:  size,
		vaild: 0,
		data:  make([]interface{}, size),
		mux:   new(sync.Mutex),
	}
}

func (c *cycleList) Record(value interface{}) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.vaild < len(c.data) {
		c.vaild++
	}
	if c.ptr+1 > len(c.data) {
		c.ptr = c.ptr % len(c.data)
	}
	c.data[c.ptr] = value
	c.ptr++
	return
}

func (c *cycleList) Report() []interface{} {
	c.mux.Lock()
	defer c.mux.Unlock()
	var result []interface{}
	if c.vaild <= len(c.data) {
		for i := 0; i < c.vaild; i++ {
			result = append(result, c.data[i%len(c.data)])
		}
	} else {
		for i := c.ptr; i < c.ptr+len(c.data); i++ {
			result = append(result, c.data[i%len(c.data)])
		}
	}
	return result
}
