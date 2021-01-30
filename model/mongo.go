package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"../config"
	tb "../toolbox"
	"github.com/astaxie/beego/logs"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 统一检查mongo数据查询的请求
func mongoBlocker() error {
	if !config.DataBaseConfig.UseMongo {
		return errors.New("mongo are not going to used, pleace check the config")
	}
	if !isMongoInit { // 延迟初始化
		deleyInitMongo()
	}
	return nil
}

// 延迟初始化mongo
// 若已初始化完成,则直接返回，否则加锁，进行初始化
func deleyInitMongo() {
	mongoInitMux.Lock()
	defer mongoInitMux.Unlock()
	if isMongoInit {
		return
	}
	var err error
	session, err = mgo.Dial(config.DataBaseConfig.MongoURL)
	if err != nil {
		logs.Error("Dial mongoDB fial: url=%s  err=%v", config.DataBaseConfig.MongoURL, err)
		panic(err)
	}
	database = session.DB(config.DataBaseConfig.MongodbName)
	if database == nil {
		logs.Error("Connect to database fail: dbName=%s", config.DataBaseConfig.MongodbName)
	}
	isMongoInit = true
	logs.Info("mongoDB delay init success...")
	return
}

// ================ IpMonitor =======================

// 设置或更新util集合的数据项
func UpdateUtilData(key string, value interface{}) error {
	var err error
	var jsonData []byte
	if err = mongoBlocker(); err != nil {
		logs.Error("%v", err)
		return err
	}
	for loop := true; loop; loop = false {
		collection := database.C(CollectUtil)
		if collection == nil {
			err = fmt.Errorf("connect to collection fail: collection=%s", CollectUtil)
			break
		}
		jsonData, err = json.Marshal(value)
		if err != nil {
			break
		}
		var newValue = UtilStruct{
			Key:       key,
			Value:     string(jsonData),
			Timestamp: time.Now().Unix(),
		}
		_, err = collection.RemoveAll(bson.M{"key": key})
		if err != nil {
			logs.Error("remove oldData failed: error=%v key=%s", err, key)
			break
		}
		err = collection.Insert(newValue)
		if err != nil {
			logs.Error("insert failed: err=%v key=%s", err, key)
			break
		}
	}
	if err != nil {
		logs.Error("update util data failed: error=%v key=%s value=%+v", err, key, value)
	} else {
		logs.Info("update util data success, key=%s", key)
	}
	return nil
}

// 根据key获取util集合的某项数据, value必须为可被修改的类型,如结构体的指针或map
func GetUtilData(key string, value interface{}) error {
	var err error
	if err = mongoBlocker(); err != nil {
		logs.Error("%v", err)
		return err
	}
	for loop := true; loop; loop = false {
		collection := database.C(CollectUtil)
		if collection == nil {
			err = fmt.Errorf("connect to collection fail: collection=%s", CollectUtil)
			break
		}
		query := collection.Find(bson.M{"key": key})
		if query == nil {
			logs.Error("find result is null: key=%s", key)
			return fmt.Errorf("no record found in database")
		}
		var result UtilStruct
		err = query.One(&result)
		if err != nil {
			logs.Error("query result failed: error=%v", err)
			return err
		}
		err = json.Unmarshal([]byte(result.Value), value)
		logs.Debug("key=%s latestValue=%+v", key, value)
	}
	return nil
}

// ================ StaticHandler ====================

// 记录文件上传信息
func InsertUploadRecord(fileName string, code string, size int64) error {
	var err error
	if err = mongoBlocker(); err != nil {
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

	if err = mongoBlocker(); err != nil {
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
	if err = mongoBlocker(); err != nil {
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
	if err = mongoBlocker(); err != nil {
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
	if err = mongoBlocker(); err != nil {
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
	if err = mongoBlocker(); err != nil {
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
