package toolbox

import (
	"fmt"
	"sync"

	"github.com/astaxie/beego/logs"
)

// 简单缓存
type SimpleCache struct {
	container map[string][]byte
	caption   int64
	size      int64
	rwmux     *sync.Mutex
}

// 创建一个简单缓存，cap为最大容量，单位为kb
func CreateSimpeCache(cap int64) (*SimpleCache, error) {
	if cap <= 0 || cap > 100<<10 { //最大容量为100M
		return nil, fmt.Errorf("unexpect cap: cap=%dkb", cap)
	}
	return &SimpleCache{
		container: make(map[string][]byte),
		caption:   cap << 10,
		size:      0,
		rwmux:     new(sync.Mutex),
	}, nil
}

func (c *SimpleCache) Save(key string, value []byte) error {
	if key == "" || value == nil {
		return fmt.Errorf("unexpect params: key=%s value=%v", key, value)
	}
	c.rwmux.Lock()
	defer c.rwmux.Unlock()
	oldValue, isExist := c.container[key]
	if isExist {
		logs.Debug("already exist, replace, key=%s len=%d", key, len(oldValue))
		c.size -= int64(len(oldValue))
		delete(c.container, key)
	}
	if c.size+int64(len(value)) > c.caption {
		return fmt.Errorf("Cache space is used up: caption=%d size=%d", c.caption, c.size)
	}
	c.size += int64(len(value))
	c.container[key] = value
	logs.Debug("save data to cache: key=%s len=%d size=%d", key, len(value), c.size)
	return nil
}

func (c *SimpleCache) Get(key string) []byte {
	if key == "" {
		return nil
	}
	c.rwmux.Lock()
	defer c.rwmux.Unlock()
	value, isExist := c.container[key]
	if !isExist {
		return nil
	}
	logs.Debug("get data from cache: key=%s len=%d", key, len(value))
	return value
}
