package config

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"strings"

	"github.com/astaxie/beego/logs"
)

type mailConfig struct {
	MailUser string `xml:"mail_user"` // 发出邮件的地址
	MailPort int    `xml:"mail_port"`
	MailPass string `xml:"mail_pass"`
	MailHost string `xml:"mail_host"` // 代理服务器地址
	MailTo   string `xml:"mail_to"`   // 接收邮件的地址
}

// 备注：目录路径配置,约定目录路径以/结尾
type serverConfig struct {
	AuthorityKey  string `xml:"authority_key"`   // 获取权限的访问路由
	IsTest        bool   `xml:"is_test"`         // 是否测试环境
	S2SSecret     string `xml:"s2s_secret"`      // s2s密钥
	ServerURL     string `xml:"serverUrl"`       // 访问本服务的url(结尾没斜杠)
	StaticPath    string `xml:"statis_path"`     // 存储静态文件的路径(斜杠结尾)
	LogPath       string `xml:"log_path"`        // 日志存储的位置(斜杠结尾)
	CloneBlogPath string `xml:"clone_blog_path"` // 克隆网站页面路径(斜杠结尾)
	BossPath      string `xml:"boss_path"`       // 管理后台页面路径(斜杠结尾)
}

type databaseConfig struct {
	UseMongo    bool   `xml:"useMongo"`    // 是否链接mongo数据库
	MongoURL    string `xml:"mongoUrl"`    // 链接mongoDB的URI
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

	// 一些检查和修正
	ServerConfig.StaticPath = strings.TrimRight(ServerConfig.StaticPath, "/") + "/"
	ServerConfig.CloneBlogPath = strings.TrimRight(ServerConfig.CloneBlogPath, "/") + "/"
	ServerConfig.BossPath = strings.TrimRight(ServerConfig.BossPath, "/") + "/"
	ServerConfig.ServerURL = strings.TrimRight(ServerConfig.ServerURL, "/")

	logs.Info("MailConfig: %+v", MailConfig)
	logs.Info("ServerConfig: %+v", ServerConfig)
	logs.Info("DataBaseConfig: %+v", DataBaseConfig)
	logs.Info("config init success...")
}
