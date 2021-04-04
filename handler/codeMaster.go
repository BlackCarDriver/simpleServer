package handler

import (
	"baseService"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"../model"
	"../rpc"
	"../toolbox"
	"github.com/astaxie/beego/logs"
)

// codeMaster的请求全部经过这里
func CodeMasterAPIHandler(w http.ResponseWriter, r *http.Request) {
	uri := strings.Trim(r.URL.Path, "/")
	logs.Debug("uri=%s", uri)
	switch uri {
	case "cmapi/createCode/debug":
		codeDebug(w, r)
	case "cmapi/createCode/submit":
		codeSubmitHandler(w, r)
	case "cmapi/home/getAllWorks":
		getAllWorksHandler(w, r)
	case "cmapi/codeDetail/getDetailByID":
		getCodeDetail(w, r)
	default:
		logs.Warn("unexpect uri: uri=%s", uri)
	}
}

// 接受测试代码和输入,返回运行结果
func codeDebug(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Code  string `json:"code"`
		Lang  string `json:"lang"`
		Input string `json:"input"`
	}
	var runResult struct {
		StdErr string `json:"stdErr"`
		StdOut string `json:"stdOut"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}
		logs.Debug("params=%+v len=%d,%d", params, len(params.Code), len(params.Input))
		// 检查参数
		if len(params.Code) > 20000 {
			err = fmt.Errorf("code too long: length=%d", len(params.Code))
			break
		}
		if len(params.Input) > 20000 {
			err = fmt.Errorf("input too long: length=%d", len(params.Input))
			break
		}
		// 运行
		var rpcResp *baseService.CommomResp
		if params.Lang == "CPP" {
			rpcResp, err = rpc.BuildCpp(params.Code, params.Input)
		} else if params.Lang == "C" {
			rpcResp, err = rpc.BuildC(params.Code, params.Input)
		} else if params.Lang == "GO" {
			rpcResp, err = rpc.BuildGo(params.Code, params.Input)
		} else {
			err = fmt.Errorf("unexpect params: lang=%q", params.Lang)
			break
		}

		// 返回值检查
		if err != nil {
			logs.Info("run code failed: error=%v params=%+v", err, params)
			break
		}
		if rpcResp.Status != 0 {
			err = fmt.Errorf("run failed: msg=%v", rpcResp.Msg)
			break
		}
		if rpcResp.Payload == nil {
			err = errors.New("payload is nil")
			break
		}
		err = json.Unmarshal(rpcResp.Payload, &runResult)
		if err != nil {
			logs.Error("unmarshal error: error=%v", err)
			break
		}
		resp.PayLoad = runResult
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 处理作品提交post请求
func codeSubmitHandler(w http.ResponseWriter, r *http.Request) {
	var work model.CodeMasterWork
	var err error
	var resp respStruct
	for loop := true; loop; loop = false {
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&work)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}
		logs.Debug("work=%+v", work)
		// 检查提交字段是否完整和有误
		if work.Language != "CPP" && work.Language != "C" && work.Language != "GO" {
			err = fmt.Errorf("unexpect params: language=%q", work.Language)
			break
		}
		if work.Title == "" || len(work.Title) > 80 {
			err = fmt.Errorf("unexpect params: title=%q", work.Title)
			break
		}
		if work.Author == "" || len(work.Author) > 80 {
			err = fmt.Errorf("unexpect params: author=%q", work.Author)
			break
		}
		if work.CType < 0 || work.CType > 4 {
			err = fmt.Errorf("unexpect params: ctype=%d", work.CType)
			break
		}
		if len(work.TagStr) > 120 {
			err = fmt.Errorf("unexpect params: tagStr=%q", work.TagStr)
			break
		}
		if len(work.InputDesc) > 1200 {
			err = fmt.Errorf("unexpect params: inputDesc=%q", work.InputDesc)
			break
		}
		if len(work.DemoInput) > 16000 {
			err = fmt.Errorf("unexpect params: DemoInput=%q", work.DemoInput)
			break
		}
		if len(work.DemoOutput) > 16000 {
			err = fmt.Errorf("unexpect params: demoOuput=%q", work.DemoOutput)
			break
		}
		if len(work.Code) > 50000 {
			err = fmt.Errorf("unexpect params: code=%q", work.Code)
			break
		}
		if len(work.Desc) > 800 || work.Desc == "" {
			err = fmt.Errorf("unexpect params: code=%q", work.Code)
			break
		}
		if len(work.Detail) > 80000 {
			err = fmt.Errorf("unexpect params: Detail=%q", work.Detail)
			break
		}
		if len(work.CoverURL) > 800 || work.CoverURL == "" {
			err = fmt.Errorf("unexpect params: coverUrl=%q", work.CoverURL)
			break
		}

		// 设置默认值
		work.Timestamp = time.Now().Unix()
		work.Score = 30
		work.ID = fmt.Sprintf("%d_%s", time.Now().Unix(), toolbox.GetRandomString(2))

		// 保存到数据库
		err = model.InsertCodeMasterWork(&work)
		if err != nil {
			logs.Error("save work failed: error=%v work=%+v", err, work)
			break
		}
		logs.Info("save work success")
	}
	if err != nil {
		logs.Warn("upload codeMaster work failed: error=%v work=%+v", err, work)
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 查询作品列表
func getAllWorksHandler(w http.ResponseWriter, r *http.Request) {
	var resp respStruct
	type payloadStruct struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		CType     int    `json:"ctype"`
		Author    string `json:"author"`
		TagStr    string `json:"tagStr"`
		Desc      string `json:"desc"`
		CoverURL  string `json:"coverUrl"`
		Timestamp int64  `json:"timestamp"`
		Score     int    `json:"score"`
	}
	payload := make([]payloadStruct, 0)
	var allWorks []*model.CodeMasterWork
	var err error
	for loop := true; loop; loop = false {
		allWorks, err = model.GetAllCodeMasterWork()
		if err != nil {
			break
		}
		for _, v := range allWorks {
			payload = append(payload, payloadStruct{
				ID:        v.ID,
				Title:     v.Title,
				CType:     v.CType,
				Author:    v.Author,
				TagStr:    v.TagStr,
				Desc:      v.Desc,
				CoverURL:  v.CoverURL,
				Timestamp: v.Timestamp,
				Score:     v.Score,
			})
		}
		logs.Info("query all works success, len=%d", len(payload))
		resp.PayLoad = payload
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 查询作品详细信息
func getCodeDetail(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ID string `json:"id"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}

		// 检查参数
		if params.ID == "" {
			logs.Warning("unexpect params: params=%+v", params)
			err = fmt.Errorf("unexpect params: %+v", params)
			break
		}
		var detail *model.CodeMasterWork
		detail, err = model.GetCodeDetailByID(params.ID)
		if err != nil {
			logs.Warn("get detail failed: error=%v params=%+v", err, params)
			break
		}
		resp.PayLoad = detail
		logs.Info("get detail success: params=%+v", params)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 提交评论
func submitRecommend(w http.ResponseWriter, r *http.Request) {
	var params struct {
		WorkID string `json:"workId"`
		Author string `json:"author"`
		Conent string `json:"conent"`
		ImgSrc string `json:"imgSrc"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}

		// 检查参数
		if params.WorkID == "" || params.Author == "" || params.Conent == "" {
			logs.Warning("unexpect params: params=%+v", params)
			err = fmt.Errorf("unexpect params: %+v", params)
			break
		}
		// 获取旧评论列表并生成新列表
		var commentData *model.CommendList
		commentData, err = model.GetCommentListByWorkID(params.WorkID)
		if err != nil {
			logs.Error("get old comment list failed: error=%v params=%v", err, params)
			break
		}
		if len(commentData.Comments) >= 100 { // 最多保存100条评论
			logs.Warn("commentList too long, give up insert: params=%+v", params)
			err = errors.New("the commentList of it works is too long")
			break
		}
		commentData.Comments = append(commentData.Comments, &model.Comment{
			Timestamp: time.Now().Unix(),
			ImgSrc:    params.ImgSrc,
			Desc:      params.Conent,
			Author:    params.Author,
		})

		// 更新数据库
		err = model.UpdateCommentList(commentData)
		if err != nil {
			logs.Error("add comment failed: eror=%v params=%+v", err, params)
			break
		}
		logs.Info("add new comment success: params=%+v", params)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 获取评论列表
func getRecommend(w http.ResponseWriter, r *http.Request) {
	var params struct {
		WorkID string `json:"workId"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		err = toolbox.MustQueryFromRequest(r, &params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}

		// 检查参数
		if params.WorkID == "" {
			logs.Warning("unexpect params: params=%+v", params)
			err = fmt.Errorf("unexpect params: %+v", params)
			break
		}
		var commemtList *model.CommendList
		commemtList, err = model.GetCommentListByWorkID(params.WorkID)
		if err != nil {
			logs.Warn("get commemtList failed: error=%v params=%+v", err, params)
			break
		}
		resp.PayLoad = commemtList
		logs.Info("get commemtList success: params=%+v", params)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}
