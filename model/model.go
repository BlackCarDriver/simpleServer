package model

import (
	"errors"
	"sync"

	"gopkg.in/mgo.v2"
)

// mongoDB 相关常量
const (
	CollectUploadFile      = "upload_file"         // 文件暂存服务记录的文件信息
	CollectCallDriverMsg   = "call_driver_msg"     //callDriver应用的聊条记录
	CollectUtil            = "util"                // 杂项信息,约定使用UtilStruct作为数据项结构
	CollectCodeMasterWorks = "code_master_work"    // codeMaster应用程序作品
	CollectCodeComment     = "code_master_comment" // codeMaster作品评论
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

// codeMaster 程序作品
type CodeMasterWork struct {
	ID          string `json:"id" bson:"_id"`
	Title       string `json:"title"`
	CType       int    `json:"ctype"`    // 作品类型 [0-其他,1-生活问题,2-数据结构,3-程序开发,4-趣味恶搞]
	Language    string `json:"language"` // 编程语言 [C++\C\GO]
	Author      string `json:"author"`
	TagStr      string `json:"tagStr"`
	Desc        string `json:"desc"`      // 简介
	InputDesc   string `json:"inputDesc"` // 输入数据格式描述
	Detail      string `json:"detail"`
	Code        string `json:"code"`
	DemoInput   string `json:"demoInput"`
	DemoOutput  string `json:"demoOuput"`
	CoverURL    string `json:"coverUrl"` // 封面图片
	Timestamp   int64  `json:"timestamp"`
	Score       int    `json:"score"` // 评分，满分为50分
	Status      int    `json:"status"`
	IsRecommend bool   `json:"isRecommend"` // 是否推荐
}

// codeMaster 单条评论
type Comment struct {
	Author    string `json:"author"`
	ImgSrc    string `json:"imgSrc"`
	Desc      string `json:"desc"` // 评论内容
	Timestamp int64  `json:"timestamp"`
}

// codeMaster作品评论列表
type CommendList struct {
	WorkID   string     `json:"workId"` // 作品的id
	Comments []*Comment `json:"comments"`
}
