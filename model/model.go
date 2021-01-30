package model

import (
	"errors"
	"sync"

	"gopkg.in/mgo.v2"
)

// mongoDB 相关常量
const (
	CollectUploadFile    = "upload_file"     // 文件暂存服务记录的文件信息
	CollectCallDriverMsg = "call_driver_msg" //callDriver应用的聊条记录
	CollectUtil          = "util"            // 杂项信息,约定使用UtilStruct作为数据项结构
)

var (
	ErrorNoRecord error = errors.New("No record")
)

//全局对象
var (
	session      *mgo.Session  = nil
	database     *mgo.Database = nil
	isMongoInit                = false // mongo是否已初始化完成
	mongoInitMux *sync.Mutex
)

func init() {
	mongoInitMux = new(sync.Mutex)
}

// ============== mongoDB 结构体 ========================

// util集合的数据统一使用这个结构体
type UtilStruct struct {
	Key       string `bson:"key"`
	Value     string `bson:"value"`
	Timestamp int64  `bson:"timestamp"`
}

// 上传文件保存后对应的文件名和下载码
type FileUpload struct {
	FileName  string `bson:"fileName"`
	Code      string `bson:"code"`
	TimeStamp int64  `bson:"timeStamp"`
	Size      int64  `bson:"size"`
}

// callDriver 应用聊天记录结构
type CallDriverChat = struct {
	ID        string `bson:"_id"`
	From      string `bson:"from"`
	To        string `bson:"to"`
	Message   string `bson:"message"`
	TimeStamp int64  `bson:"timeStamp"`
	IP        string `bson:"ip"`
	Status    int    `bson:"status"`
}
