package rpc

import (
	"context"

	"codeRunner"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
)

// 创建一个codeRunner服务端
func NewCodeRunner(ctx context.Context) (client *codeRunner.CodeRunnerClient, err error) {
	var transport thrift.TTransport
	var node *s2sMember
	node, err = s2sMaster.GetNodeByServiceName(codeRunnerS2SName)
	if err != nil {
		logs.Info("get service node failed: err=%s", err)
		return
	}
	if node == nil {
		logs.Error("node is nil?...")
		return
	}
	for loop := true; loop; loop = false {
		transport, err = thrift.NewTSocket(node.URL)
		if err != nil {
			logs.Error("get socket failed: error=%v", err)
			break
		}
		transport, err = transportFactory.GetTransport(transport)
		if err != nil {
			logs.Error("get transport failed: error=%v", err)
			break
		}
		if err = transport.Open(); err != nil {
			logs.Error("open transport failed: error=%v", err)
			transport.Close()
			break
		}
	}
	// 上报问题节点
	if err != nil {
		logs.Info("a node can't be used: node=%+v", *node)
		node.UpdateCounter(false)
		go node.GoReport()
		return nil, err
	}

	go func() { // 延迟关闭连接
		select {
		case <-ctx.Done():
			transport.Close()
			logs.Info("transport Close...")
			node.UpdateCounter(true)
		}
		if ctx.Err() == context.DeadlineExceeded { // 说明可能代码有误
			node.UpdateCounter(false)
			logs.Error("a connection over time!")
		}
	}()
	client = codeRunner.NewCodeRunnerClient(
		thrift.NewTStandardClient(
			protocolFactory.GetProtocol(transport),
			protocolFactory.GetProtocol(transport)))
	return
}
