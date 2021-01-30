package service

import (
	"context"

	"./gen-go/codeRunner"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
)

func NewCodeRunnerTest() {
	var transport thrift.TTransport
	var err error
	var addr = "127.0.0.1:81"
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
	defer transport.Close()
	if err := transport.Open(); err != nil {
		logs.Error("open transport failed: error=%v", err)
		return
	}
	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)
	codeRunnerClient := codeRunner.NewCodeRunnerClient(thrift.NewTStandardClient(iprot, oprot))
	res, err := codeRunnerClient.Ping(context.Background())
	logs.Info("result=%v  error=%v", res, err)
}
