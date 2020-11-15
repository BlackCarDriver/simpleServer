package model

import (
	"fmt"
	"time"

	"github.com/astaxie/beego/logs"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func InsertUploadRecord(fileName string, code string, size int64) error {
	var err error
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

func GetUploadRecord(code string) (FileUpload, error) {
	var record FileUpload
	var err error
	var query *mgo.Query

	collection := database.C(CollectUploadFile)
	query = collection.Find(bson.M{"code": code})
	if query == nil {
		logs.Info("find result is null: code=%s", code)
		return record, ErrorNoRecord
	}
	err = query.One(&record)
	if err != nil {
		logs.Error(err)
	}
	return record, err
}
