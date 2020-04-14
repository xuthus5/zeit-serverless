package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var (
	// 配置信息
	oss = &Oss{
		Bucket:   "xuthus",                                       //Bucket名称
		Ak:       "LTAIeNu9L0MzBtJH",                             //Accesskey
		Sk:       "3WIo0HtVYaqm9bawbjhIJ3gIEyErzd",               //Secretkey
		Endpoint: "oss-cn-shanghai.aliyuncs.com",                 //地域节点
		Domain:   "https://xuthus.oss-cn-shanghai.aliyuncs.com/", //OSS外网访问域名 [结尾请带/]
	}
	//操作仓库对象
	bucket *sdk.Bucket
	//返回值
	response []byte
)

// Oss OSS配置项
type Oss struct {
	Bucket   string //Bucket
	Ak       string //AccessKey ID
	Sk       string //Access Key Secret
	Endpoint string //外网访问地域节点(非Bucket域名)
	Domain   string //自定义域名(Bucket域名或自定义)
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

type ListObject struct {
	Filename   string    `json:"filename"`
	Prefix     string    `json:"prefix"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"is_dir"`
	CreateTime time.Time `json:"create_time"`
}

// Init 初始化操作
func Init() *Response {
	client, err := sdk.New(oss.Endpoint, oss.Ak, oss.Sk)
	if err != nil {
		return &Response{
			Code:    500,
			Message: "ErrorInitClient:" + err.Error(),
		}
	}
	// 获取存储空间。
	bucket, err = client.Bucket(oss.Bucket)
	if err != nil {
		return &Response{
			Code:    500,
			Message: "ErrorInitBucket:" + err.Error(),
		}
	}
	return nil
}

// Handler 请求参数信息
// Operate: 操作类型 [list,delete,upload,domain,mkdir]
// Prefix: 操作的前缀(前缀意为操作所在的目录)
// Path: 操作的绝对地址

// Handler 句柄
func Handler(w http.ResponseWriter, r *http.Request) {
	//公共的响应头设置
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, OPTIONS")
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	//初始化
	if err := Init(); err != nil {
		response, _ := json.Marshal(err)
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	}
	//执行何种操作
	var operate = r.URL.Query().Get("operate")
	if operate == "list" {
		// 列举当前目录下的所有文件
		var result []ListObject //结果集
		//设置筛选器
		var path = r.URL.Query().Get("prefix")
		maker := sdk.Marker(path)
		prefix := sdk.Prefix(path)
		//结果入 result
		for {
			lsRes, err := bucket.ListObjects(maker, prefix, sdk.Delimiter("/"))
			if err != nil {
				response, _ := json.Marshal(&Response{
					Code:    500,
					Message: "ErrorListObject:" + err.Error(),
				})
				w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
				_, _ = w.Write(response)
				return
			}
			for _, dirname := range lsRes.CommonPrefixes {
				result = append(result, ListObject{
					Filename:   strings.Replace(dirname, path, "", 1),
					CreateTime: time.Time{},
					IsDir:      true,
					Prefix:     path,
				})
			}
			for _, obj := range lsRes.Objects {
				result = append(result, ListObject{
					Filename:   strings.Replace(obj.Key, path, "", 1),
					CreateTime: obj.LastModified,
					IsDir:      false,
					Prefix:     path,
					Size:       obj.Size,
				})
			}
			prefix = sdk.Prefix(lsRes.Prefix)
			maker = sdk.Marker(lsRes.NextMarker)
			if !lsRes.IsTruncated {
				break
			}
		}
		response, _ = json.Marshal(&List{
			Code:    200,
			Message: oss.Domain,
			Data:    result,
			Count:   len(result),
		})
	} else if operate == "delete" {
		//需要删除的文件绝对路径
		var path = r.URL.Query().Get("path")
		err := bucket.DeleteObject(path)
		if err != nil {
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorObjectDelete:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		response, _ = json.Marshal(&Response{
			Code:    200,
			Message: "ok",
		})
	} else if operate == "upload" {
		var _, header, err = r.FormFile("file")
		var prefix string
		_ = r.ParseMultipartForm(32 << 20)
		if r.MultipartForm != nil {
			values := r.MultipartForm.Value["prefix"]
			if len(values) > 0 {
				prefix = values[0]
			}
		}
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
		err = bucket.PutObject(prefix+dst, source)
		if err != nil {
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorObjectUpload:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		response, _ = json.Marshal(&Response{
			Code:    200,
			Message: "ok",
			Data:    oss.Domain + prefix + dst,
		})
	} else if operate == "domain" {
		response, _ = json.Marshal(&Response{
			Code:    200,
			Message: oss.Domain,
		})
	} else if operate == "mkdir" {
		var prefix = r.URL.Query().Get("prefix")
		var dirname = r.URL.Query().Get("dirname")
		err := bucket.PutObject(prefix+dirname, nil)
		if err != nil {
			response, _ := json.Marshal(&Response{
				Code:    500,
				Message: "ErrorMkdir:" + err.Error(),
			})
			w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
			_, _ = w.Write(response)
			return
		}
		response, _ = json.Marshal(&Response{
			Code:    200,
			Message: "ok",
		})
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
	_, _ = w.Write(response)
	return
}
