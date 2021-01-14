package config

import (
	"encoding/xml"
	"io/ioutil"
	"os"

	"github.com/astaxie/beego/logs"
)

type mailConfig struct {
	MailUser string `xml:"mail_user"` // 发出邮件的地址
	MailPort int    `xml:"mail_port"`
	MailPass string `xml:"mail_pass"`
	MailHost string `xml:"mail_host"` // 代理服务器袋子
	MailTo   string `xml:"mail_to"`   // 接收邮件的地址
}

type serverConfig struct {
	AuthorityKey  string `xml:"authority_key"`   // 将ip加入到白名单的路由地址
	IsTest        bool   `xml:"is_test"`         // 是否测试环境
	DownloadUrlTp string `xml:"download_url"`    // 请求下载文件的url模板
	SourcePathTp  string `xml:"source_path"`     // 文件上传和下载的文件路径模板
	StaticPathTP  string `xml:"statis_path"`     // 存储静态文件的位置
	CloneBlogPath string `xml:"clone_blog_path"` // 克隆网站存储的位置
}

type databaseConfig struct {
	UseMongo    bool   `xml:"useMongo"`    // 是否链接mongo数据库
	MongoURL    string `xml:"mongoUrl"`    // 链接mongoDB的链接
	MongodbName string `xml:"mongodbName"` // 使用的mongoDB数据库名称
}

var MailConfig mailConfig
var ServerConfig serverConfig
var DataBaseConfig databaseConfig

func init() {
	xmlFile, err := os.Open("./config/config.xml")
	if err != nil {
		logs.Critical("Error opening config file: %v", err)
		os.Exit(1)
		return
	}
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)
	xml.Unmarshal(b, &MailConfig)
	xml.Unmarshal(b, &ServerConfig)
	xml.Unmarshal(b, &DataBaseConfig)

	logs.Info("MailConfig: %+v", MailConfig)
	logs.Info("ServerConfig: %+v", ServerConfig)
	logs.Info("DataBaseConfig: %+v", DataBaseConfig)
	logs.Info("config init success...")
}
