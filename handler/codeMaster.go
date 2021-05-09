package handler

import (
	"baseService"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"../config"
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
	case "cmapi/codeDetail/getCommentList":
		getRecommend(w, r)
	case "cmapi/codeDetail/submitComment":
		submitRecommend(w, r)
	case "cmapi/codeDetail/updateWork":
		updateWork(w, r)
	case "cmapi/codeDetail/runWork":
		runWork(w, r)
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
		work.IsRecommend = false
		work.Timestamp = time.Now().Unix()
		work.Score = 30
		work.ID = fmt.Sprintf("%d_%s", time.Now().Unix(), toolbox.GetRandomString(2))

		// 保存到数据库
		err = model.InsertCodeMasterWork(&work)
		if err != nil {
			logs.Error("save work failed: error=%v work=%+v", err, work)
			break
		}
		resp.PayLoad = work.ID
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
		ID          string `json:"id"`
		Title       string `json:"title"`
		CType       int    `json:"ctype"`
		Author      string `json:"author"`
		TagStr      string `json:"tagStr"`
		Desc        string `json:"desc"`
		CoverURL    string `json:"coverUrl"`
		Timestamp   int64  `json:"timestamp"`
		Score       int    `json:"score"`
		IsRecommend bool   `json:"isRecommend"`
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
				ID:          v.ID,
				Title:       v.Title,
				CType:       v.CType,
				Author:      v.Author,
				TagStr:      v.TagStr,
				Desc:        v.Desc,
				CoverURL:    v.CoverURL,
				Timestamp:   v.Timestamp,
				Score:       v.Score,
				IsRecommend: v.IsRecommend,
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
		WorkID  string `json:"workId"`
		Author  string `json:"author"`
		Comment string `json:"comment"`
		ImgSrc  string `json:"imgSrc"`
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

		// 检查参数
		if params.WorkID == "" || params.Author == "" || params.Comment == "" {
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
		if commentData == nil {
			commentData = &model.CommendList{
				WorkID:   params.WorkID,
				Comments: make([]*model.Comment, 0),
			}
		}
		if len(commentData.Comments) >= 100 { // 最多保存100条评论
			logs.Warn("commentList too long, give up insert: params=%+v", params)
			err = errors.New("the commentList of it works is too long")
			break
		}
		commentData.Comments = append(commentData.Comments, &model.Comment{
			Timestamp: time.Now().Unix(),
			ImgSrc:    params.ImgSrc,
			Desc:      params.Comment,
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
		resp.PayLoad = commemtList.Comments
		logs.Info("get commemtList success: params=%+v", params)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 更新作品信息或删除算法作品
func updateWork(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Key         string `json:"key"`    // 认真密钥
		OpType      string `json:"opType"` // [UPDATE|DELETE]
		WorkID      string `json:"workId"`
		Title       string `json:"title"`       // 为空不更新
		IsRecommend int    `json:"isRecommend"` // 0时不更新，大于0推荐，小于0不推荐
		Score       int    `json:"score"`       // 评分，满分为50分,0分时不更新
		CoverURL    string `json:"coverUrl"`    // 封面图片，为空时不更新
		TagStr      string `json:"tagStr"`      // 标签，为空时不更新
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err = toolbox.MustQueryFromRequest(r, &params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}

		// 检查参数
		if params.WorkID == "" {
			logs.Warning("unexpect params: params=%+v", params)
			err = fmt.Errorf("unexpect params: workID=%v", params.WorkID)
			break
		}
		if params.OpType != "UPDATE" && params.OpType != "DELETE" {
			logs.Warning("unexpect params: params=%+v", params)
			err = fmt.Errorf("unexpect params: optype=%s", params.OpType)
			break
		}
		if params.Key != config.ServerConfig.AuthorityKey {
			logs.Warning("unexpect key: %s", params.Key)
			err = errors.New("not Authority")
			break
		}

		// 数据操作
		if params.OpType == "DELETE" {
			err = model.UpdateWorksStatus(params.WorkID, -1)
			if err != nil {
				logs.Error("delete work failed: params=%+v error=%v", params, err)
				break
			}
			logs.Info("delete work success: params=%+v", params)
			break
		}
		if params.OpType == "UPDATE" {
			err = model.UpdateWorksInfo(params.WorkID, params.Score, params.IsRecommend, params.Title, params.CoverURL, params.TagStr)
			if err != nil {
				logs.Error("update work failed: params=%+v error=%v", params, err)
				break
			}
			logs.Info("update work success: params=%+v", params)
			break
		}
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}

// 执行算法作品请求
func runWork(w http.ResponseWriter, r *http.Request) {
	var params struct {
		WorkID string `json:"workId"`
		Input  string `json:"input"`
	}
	var payload struct {
		StdErr string `json:"stdErr"`
		StdOut string `json:"stdOut"`
	}
	var resp respStruct
	var err error
	for loop := true; loop; loop = false {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&params)
		if err != nil {
			logs.Error("parse params failed: error=%v", err)
			break
		}

		// 检查参数
		if params.WorkID == "" {
			logs.Warning("unexpect params: params=%+v", params)
			err = fmt.Errorf("unexpect params: workID=%v", params.WorkID)
			break
		}
		// 根据id获取算法详情
		var work *model.CodeMasterWork
		work, err = model.GetCodeDetailByID(params.WorkID)
		if err != nil {
			logs.Error("get work detail failed: err=%v id=%s", err, params.WorkID)
			break
		}
		logs.Info("lang=%s code=%s", work.Language, work.Code)

		// 执行算法
		var rpcResp *baseService.CommomResp
		if work.Language == "CPP" {
			rpcResp, err = rpc.BuildCpp(work.Code, params.Input)
		} else if work.Language == "C" {
			rpcResp, err = rpc.BuildC(work.Code, params.Input)
		} else if work.Language == "GO" {
			rpcResp, err = rpc.BuildGo(work.Code, params.Input)
		} else {
			err = fmt.Errorf("unexpect params: lang=%q", work.Language)
			break
		}
		if err != nil {
			logs.Error("run work failed: err=%v params=%+v", err, params)
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
		err = json.Unmarshal(rpcResp.Payload, &payload)
		if err != nil {
			logs.Error("unmarshal failed: err=%v", err)
			break
		}
		resp.PayLoad = payload
		logs.Info("run work success, resp=%+v", rpcResp)
	}
	if err != nil {
		resp.Status = -1
		resp.Msg = fmt.Sprint(err)
	}
	responseJson(&w, resp)
}
