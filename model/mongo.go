package model

import (
	"errors"
	"fmt"
	"time"

	tb "../toolbox"
	"github.com/astaxie/beego/logs"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 统一拦截mongo数据查询请求
func commonBlocker() error {
	if !isInitMongo {
		return errors.New("mongo is not init")
	}
	return nil
}

// =============================================

// 记录文件上传信息
func InsertUploadRecord(fileName string, code string, size int64) error {
	var err error
	if err = commonBlocker(); err != nil {
		logs.Error("%v", err)
		return err
	}

	for loop := true; loop; loop = false {
		if fileName == "" || code == "" {
			err = fmt.Errorf("unexpcet params: fileName=%s code=%s", fileName, code)
			break
		}
		collection := database.C(CollectUploadFile)
		if collection == nil {
			err = fmt.Errorf("connect to collection fail: collection=%s", CollectUploadFile)
			break
		}
		data := FileUpload{
			FileName:  fileName,
			Code:      code,
			TimeStamp: time.Now().Unix(),
			Size:      size,
		}
		err = collection.Insert(data)
		if err != nil {
			break
		}
	}
	if err != nil {
		logs.Error("mongoDB insert Error: %v", err)
	}
	return err
}

// 获取文件保存信息
func GetUploadRecord(code string) (FileUpload, error) {
	var record FileUpload
	var err error
	var query *mgo.Query

	if err = commonBlocker(); err != nil {
		logs.Error("%v", err)
		return record, err
	}

	collection := database.C(CollectUploadFile)
	query = collection.Find(bson.M{"code": code})
	if query == nil {
		logs.Warning("find result is null: code=%s", code)
		return record, ErrorNoRecord
	}
	err = query.One(&record)
	if err != nil {
		logs.Error(err)
	}
	return record, err
}

// =============== CallDriver ==================
// 保存callDriver应用中收到的来自其他用户的消息
func InsertCallDriverMessage(from, to, msg, ip string) error {
	var err error
	if err = commonBlocker(); err != nil {
		logs.Error("%v", err)
		return err
	}

	ts := time.Now().Unix()
	record := CallDriverChat{
		ID:        fmt.Sprintf("%d%s", ts, tb.GetRandomString(3)),
		From:      from,
		To:        to,
		Message:   msg,
		TimeStamp: ts,
		IP:        ip,
		Status:    0,
	}
	for loop := true; loop; loop = false {
		if from == "" || to == "" || msg == "" {
			err = fmt.Errorf("unexpect params: from=%s to=%s msg=%s", from, to, msg)
			break
		}
		collection := database.C(CollectCallDriverMsg)
		if collection == nil {
			err = fmt.Errorf("connect to collection fail: collection=%s", CollectCallDriverMsg)
			break
		}
		err = collection.Insert(record)
		if err != nil {
			break
		}
	}
	logs.Debug("insert result: collection=%s err=%v record=%v", CollectCallDriverMsg, err, msg)
	return err
}

// 查询callDriver应用的聊天记录
func FindCallDriverMessage(nick string, num int) (history []CallDriverChat, err error) {
	history = make([]CallDriverChat, 0)
	if err = commonBlocker(); err != nil {
		logs.Error("%v", err)
		return history, err
	}
	for loop := true; loop; loop = false {
		if nick == "" || num <= 0 {
			err = fmt.Errorf("unexpect params: nick=%s num=%d", nick, num)
			break
		}
		collection := database.C(CollectCallDriverMsg)
		if collection == nil {
			err = fmt.Errorf("connect to collection fail: collection=%s", CollectCallDriverMsg)
			break
		}
		var query *mgo.Query
		query = collection.Find(bson.M{"$or": []bson.M{bson.M{"from": nick}, bson.M{"to": nick}}}).Sort("-timeStamp").Limit(num)
		if query == nil {
			err = fmt.Errorf("query return null")
			break
		}
		query.All(&history)
		// 更新消息状态
		if len(history) > 0 {
			ids := make([]string, 0)
			for _, t := range history {
				ids = append(ids, t.ID)
			}
			go UpdateCallDriverMessage(ids)
		}
	}
	logs.Debug("find result: collection=%s err=%v nick=%s history.len=%d",
		CollectCallDriverMsg, err, nick, len(history))
	return history, err
}

// 记录聊天记录已读,status自增1
func UpdateCallDriverMessage(ids []string) {
	var err error
	if err = commonBlocker(); err != nil {
		logs.Error("%v", err)
		return
	}
	if ids == nil || len(ids) == 0 {
		logs.Warning("unexpect params: ids=%v", ids)
		return
	}
	collection := database.C(CollectCallDriverMsg)
	if collection == nil {
		logs.Error("connect to collection fail: collection=%s", CollectCallDriverMsg)
		return
	}
	info, err := collection.UpdateAll(bson.M{"_id": bson.M{"$in": ids}}, bson.M{"$inc": bson.M{"status": 1}})
	if err != nil {
		logs.Error("update callDriver chat fail: err=%v ids=%v", err, ids)
		return
	}
	logs.Debug("update message status success: total=%d update=%d", len(ids), info.Updated)
}

// 查询所有聊天记录
func FindAllCallDriverMessage() (history []CallDriverChat, err error) {
	history = make([]CallDriverChat, 0)
	if err = commonBlocker(); err != nil {
		logs.Error("%v", err)
		return history, err
	}
	for loop := true; loop; loop = false {
		collection := database.C(CollectCallDriverMsg)
		if collection == nil {
			err = fmt.Errorf("connect to collection fail: collection=%s", CollectCallDriverMsg)
			break
		}
		var query *mgo.Query
		myName := "BlackCarDriver"
		query = collection.Find(bson.M{"$or": []bson.M{bson.M{"from": myName}, bson.M{"to": myName}}}).Sort("-timeStamp").Limit(50)
		if query == nil {
			err = fmt.Errorf("query return null")
			break
		}
		query.All(&history)
	}
	logs.Debug("find result: collection=%s err=%v history.len=%d", CollectCallDriverMsg, err, len(history))
	return history, err
}
