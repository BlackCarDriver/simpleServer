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
	AuthorityKey  string `xml:"authority_key"` // 将ip加入到白名单的路由地址
	IsTest        bool   `xml:"is_test"`
	DownloadUrlTp string `xml:"download_url"` // 请求下载文件的url模板
	SourcePathTp  string `xml:"source_path"`  // 文件上传和下载的文件路径模板
}

var MailConfig mailConfig
var ServerConfig serverConfig

func init() {
	xmlFile, err := os.Open("./config/server.xml")
	if err != nil {
		logs.Critical("Error opening config file: %v", err)
		os.Exit(1)
		return
	}
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)
	xml.Unmarshal(b, &MailConfig)
	xml.Unmarshal(b, &ServerConfig)

	logs.Info("MailConfig: %+v", MailConfig)
	logs.Info("ServerConfig: %+v", ServerConfig)
}
