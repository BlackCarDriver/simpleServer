package service

import (
	"context"

	"github.com/apache/thrift/lib/go/thrift"
)

var protocolFactory thrift.TProtocolFactory
var transportFactory thrift.TTransportFactory
var defaultCtx context.Context

func init() {
	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory = thrift.NewTBufferedTransportFactory(8192)
	defaultCtx = context.Background()
}
