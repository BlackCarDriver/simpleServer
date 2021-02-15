package rpc

import (
	"context"

	"baseService"
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
			break
		}
		client = codeRunner.NewCodeRunnerClient(
			thrift.NewTStandardClient(protocolFactory.GetProtocol(transport), protocolFactory.GetProtocol(transport)))
		_, err = client.Ping(ctx, "")
		if err != nil {
			logs.Error("can't not ping")
			break
		}
	}
	// 上报问题节点
	if err != nil {
		transport.Close()
		logs.Info("a node can't be used: error=%v node=%+v", err, *node)
		node.UpdateCounter(false)
		go node.GoReport()
		return nil, err
	}

	go func() { // 延迟关闭连接
		select {
		case <-ctx.Done():
			logs.Info("transport Close...")
			node.UpdateCounter(true)
		}
		if ctx.Err() == context.DeadlineExceeded { // 说明可能代码有误
			node.UpdateCounter(false)
			logs.Error("a connection time expired!")
		}
		transport.Close()
	}()

	return
}

// ----------------------------

func BuildGo(code, input string) (*baseService.CommomResp, error) {
	ctx, cancel := GetDefaultContext()
	defer cancel()
	client, err := NewCodeRunner(ctx)
	if err != nil {
		logs.Error("new client fail: error=%v", err)
		return nil, err
	}
	res, err := client.BuildGo(ctx, code, input)
	logs.Info("resp=%+v error=%v", res, err)
	return res, err
}

func BuildCpp(code, input string) (*baseService.CommomResp, error) {
	ctx, cancel := GetDefaultContext()
	defer cancel()
	client, err := NewCodeRunner(ctx)
	if err != nil {
		logs.Error("new client fail: error=%v", err)
		return nil, err
	}
	res, err := client.BuildCpp(ctx, code, input)
	logs.Info("resp=%+v error=%v", res, err)
	return res, err
}

func Run(codeType, hash, input string) (*baseService.CommomResp, error) {
	ctx, cancel := GetDefaultContext()
	defer cancel()
	client, err := NewCodeRunner(ctx)
	if err != nil {
		logs.Error("new client fail: error=%v", err)
		return nil, err
	}
	res, err := client.Run(ctx, codeType, hash, input)
	logs.Info("resp=%+v error=%v", res, err)
	return res, err
}
