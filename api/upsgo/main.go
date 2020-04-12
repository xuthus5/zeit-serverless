package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/upyun/go-sdk/upyun"
)

// Ups Upyun 配置
type Ups struct {
	Bucket   string `yaml:"Bucket"`   //服务名称
	Operator string `yaml:"Operator"` //权的操作员名称
	Password string `yaml:"Password"` //授权的操作员密码
	Domain   string `yaml:"domain"`   //加速域名
}

// Response 是交付层的基本回应
type Response struct {
	Code    int         `json:"code"`    //请求状态代码
	Message interface{} `json:"message"` //请求结果提示
	Data    interface{} `json:"data"`    //请求结果与错误原因
}

// List 会返回给交付层一个列表回应
type List struct {
	Code    int         `json:"code"`    //请求状态代码
	Count   int         `json:"count"`   //数据量
	Message interface{} `json:"message"` //请求结果提示
	Data    interface{} `json:"data"`    //请求结果
}

// Handler 逻辑处理
func Handler(w http.ResponseWriter, r *http.Request) {
	//初始化
	conf := &Ups{
		Bucket:   "xuthus-static-files",
		Operator: "xuthus",
		Password: "afqoIZRsCt9nlPO4JPpQQIqCMw4d5ojM",
		Domain:   "https://images.xuthus.cc",
	}
	var up = upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   conf.Bucket,
		Operator: conf.Operator,
		Password: conf.Password,
	})
	//公共的响应头设置
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	//执行何种操作
	var operate = r.URL.Query().Get("operate")
	if operate == "list" {
		var path = r.URL.Query().Get("path")
		// path 为空 默认根目录
		if path == "" {
			path = "/"
		}

		objsChan := make(chan *upyun.FileInfo, 10)
		go func() {
			up.List(&upyun.GetObjectsConfig{
				Path:        path,
				ObjectsChan: objsChan,
			})
		}()

		var list []*upyun.FileInfo

		for obj := range objsChan {
			list = append(list, obj)
		}

		//返回信息
		response, _ := json.Marshal(&List{
			Code:    200,
			Message: "ok",
			Data:    list,
			Count:   len(list),
		})

		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	} else if operate == "delete" {
		//需要删除的文件绝对路径
		var path = r.URL.Query().Get("path")
		//执行删除
		if err := up.Delete(&upyun.DeleteObjectConfig{
			Path:  path,
			Async: false,
		}); err != nil {
			//删除失败
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorDelete:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		response, _ := json.Marshal(&Response{
			Code:    200,
			Message: "ok",
		})
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	} else if operate == "upload" {
		var path = r.Form.Get("path")
		var _, header, err = r.FormFile("file")
		if err != nil {
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorUpload:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		dst := header.Filename
		source, _ := header.Open()
		if err := up.Put(&upyun.PutObjectConfig{
			Path:   path + "/" + dst,
			Reader: source,
		}); err != nil {
			//上传失败
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorUpload:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		response, _ := json.Marshal(&Response{
			Code:    200,
			Message: "ok",
		})
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	} else if operate == "mkdir" {
		var dir = r.URL.Query().Get("dir")
		if err := up.Mkdir(dir); err != nil {
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorMkdir:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		response, _ := json.Marshal(&Response{
			Code:    200,
			Message: "ok",
		})
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	} else if operate == "domain" {
		response, _ := json.Marshal(&Response{
			Code:    200,
			Message: conf.Domain,
		})
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	}
}
