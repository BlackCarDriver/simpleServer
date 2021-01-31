package rpc

import (
	"context"
	"time"

	"codeRunner"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
)

// 返回一个10分钟自动关闭的context
func GetDefaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Minute*10)
}

// 创建一个codeRunner服务端
func NewCodeRunner(ctx context.Context) (client *codeRunner.CodeRunnerClient, err error) {
	var transport thrift.TTransport
	var addr string
	addr, err = s2sMaster.GetUrlByServiceName(codeRunnerS2SName)
	if err != nil {
		logs.Error("get service addre failed: err=%s", err)
		return
	}
	transport, err = thrift.NewTSocket(addr)
	if err != nil {
		logs.Error("get socket failed: error=%v addr=%s", err, addr)
		return
	}
	transport, err = transportFactory.GetTransport(transport)
	if err != nil {
		logs.Error("get transport failed: error=%v", err)
		return
	}
	if err = transport.Open(); err != nil {
		logs.Error("open transport failed: error=%v", err)
		transport.Close()
		return
	}
	go func() { // 延迟关闭连接
		select {
		case <-ctx.Done():
			transport.Close()
			logs.Info("transport Close...")
		}
		if ctx.Err() == context.DeadlineExceeded { // 说明可能代码有误
			logs.Error("a connection over time!")
		}
	}()
	client = codeRunner.NewCodeRunnerClient(
		thrift.NewTStandardClient(
			protocolFactory.GetProtocol(transport),
			protocolFactory.GetProtocol(transport)))
	return
}