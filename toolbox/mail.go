package toolbox

import (
	"fmt"

	"../config"
	"github.com/astaxie/beego/logs"
	"gopkg.in/gomail.v2"
)

// 相关配置
var (
	agentHost string
	agentPort int
	agentUser string
	agentPass string
)

func initMailSender() {
	logs.Info("mail sender init...")
	agentHost = config.MailConfig.MailHost
	agentPort = config.MailConfig.MailPort
	agentUser = config.MailConfig.MailUser
	agentPass = config.MailConfig.MailPass
}

// 发送邮箱接口
func sendMail(mailTo []string, subject string, body string) error {
	d := gomail.NewDialer(
		agentHost,
		agentPort,
		agentUser,
		agentPass,
	)
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(agentUser, "CallDriver")) //添加别名
	m.SetHeader("To", mailTo...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	err := d.DialAndSend(m)
	if err != nil {
		logs.Debug("send mail fial: %+v", d)
	}
	return err
}

// 发送一条消息到自己的邮箱
func SendToMySelf(name, body string) error {
	mailTo := []string{config.MailConfig.MailTo}
	subject := fmt.Sprintf("来自 %s 的消息", name)
	err := sendMail(mailTo, subject, body)
	if err != nil {
		logs.Error("Send email fail: %v", err)
		return err
	}
	logs.Info("send success")
	return nil
}
