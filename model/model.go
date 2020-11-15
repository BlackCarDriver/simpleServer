package model

import (
	"errors"

	"../config"
	"github.com/astaxie/beego/logs"
	"gopkg.in/mgo.v2"
)

// mongoDB 相关常量
const (
	mongodbName       = "simpleServer"
	CollectUploadFile = "upload_file"
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
