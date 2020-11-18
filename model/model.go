package model

import (
	"errors"

	"../config"
	"github.com/astaxie/beego/logs"
	"gopkg.in/mgo.v2"
)

// mongoDB 相关常量
const (
	mongodbName          = "simpleServer"
	CollectUploadFile    = "upload_file"
	CollectCallDriverMsg = "call_driver_msg" //callDriver应用的聊条记录
)

var (
	ErrorNoRecord error = errors.New("No record")
)

//全局对象
var (
	session  *mgo.Session  = nil
	database *mgo.Database = nil
)

func init() {
	// 链接mongo
	var err error
	session, err = mgo.Dial(config.DataBaseConfig.MongoURL)
	if err != nil {
		logs.Error("Dial mongoDB fial: url=%s  err=%v", config.DataBaseConfig.MongoURL, err)
		panic(err)
	}
	database = session.DB(mongodbName)
	if database == nil {
		logs.Error("Connect to database fail: dbName=%s", mongodbName)
	}
	logs.Info("mongoDB init success...")
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
	Status    int    `bson:"status"`
}
