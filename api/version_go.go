package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var (
	// sqlite数据库-连接引擎
	engine *gorm.DB
)

// Connect 数据库连接
func Connect() error {
	var err error
	if engine, err = gorm.Open("sqlite3", "data/data.db"); err != nil {
		return err
	}
	engine.SingularTable(true)
	return nil
}

// List 会返回给交付层一个列表回应
type List struct {
	Code    int         `json:"code"`    //请求状态代码
	Count   int         `json:"count"`   //数据量
	Message interface{} `json:"message"` //请求结果提示
	Data    interface{} `json:"data"`    //请求结果
}

// GoVersion 版本
type GoVersion struct {
	ID      int        `json:"-"`
	Version string     `gorm:"unique;not null" json:"version"` //版本
	Stable  bool       `json:"stable"`                         //是否为当前主线版本
	List    []GoBranch `json:"list"`
}

// GoBranch 版本分支信息
type GoBranch struct {
	ID          int    `json:"-"`                                 //ID
	GoVersionID int    `json:"-"`                                 //外键
	FileName    string `gorm:"size:32;not null" json:"file_name"` //名称
	Kind        string `gorm:"size:16;not null" json:"kind"`      //安装包类型
	Platform    string `gorm:"size:8;not null" json:"os"`         //平台
	Arch        string `gorm:"size:8;not null" json:"arch"`       //架构
	Size        string `gorm:"size:8;not null" json:"size"`       //大小
	CheckSum    string `gorm:"size:64;not null" json:"check_sum"` //校检
}

// GoParams 接收参数
type GoParams struct {
	Version  string `form:"version"` //版本号
	Platform string `form:"os"`      //平台
	Arch     string `form:"arch"`    //架构
	Kind     string `form:"kind"`    //包类型
	Stable   bool   `form:"stable"`  //是否稳定版
}

// Handler serverless-functions 函数暴露
func Handler(w http.ResponseWriter, r *http.Request) {
	if err := Connect(); err != nil {
		response, _ := json.Marshal(&List{
			Code:    200,
			Message: err.Error(),
		})
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	}
	defer engine.Close()
	goVersion := r.URL.Query().Get("version")
	platform := r.URL.Query().Get("os")
	arch := r.URL.Query().Get("arch")
	kind := r.URL.Query().Get("kind")
	var stable bool
	if r.URL.Query().Get("stable") == "true" {
		stable = true
	}
	var params = &GoParams{
		Version:  goVersion,
		Platform: platform,
		Arch:     arch,
		Kind:     kind,
		Stable:   stable,
	}
	data, err := Fetch(params)
	if err != nil {
		response, _ := json.Marshal(&List{
			Code:    200,
			Message: err.Error(),
		})
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
		_, _ = w.Write(response)
		return
	}
	response, _ := json.Marshal(&List{
		Code:    200,
		Message: "ok",
		Count:   len(data),
		Data:    data,
	})
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(string(response))))
	_, _ = w.Write(response)
}

// Fetch 查询数据
func Fetch(params *GoParams) ([]GoVersion, error) {
	var list []GoVersion
	if reflect.DeepEqual(params, &GoParams{}) {
		err := engine.Preload("List").Find(&list).Error
		return list, err
	}
	var err error
	if params.Version != "" && params.Stable {
		//筛选版本
		err = engine.Where("version = ? AND stable = ?", params.Version, params.Stable).Find(&list).Error
	} else if params.Version != "" {
		err = engine.Where("version = ? ", params.Version).Find(&list).Error
	} else if params.Stable {
		err = engine.Where("stable = ?", params.Stable).Find(&list).Error
	} else {
		//不筛选版本
		err = engine.Find(&list).Error
	}
	if err != nil {
		return nil, err
	}
	//数据项为空
	if len(list) == 0 {
		return nil, errors.New("Couldn't find Version:" + params.Version)
	}
	//从版本中选择下属分支
	var filter = make(map[string]interface{}, 4)
	if params.Platform != "" {
		filter["platform"] = params.Platform
	}
	if params.Kind != "" {
		filter["kind"] = params.Kind
	}
	if params.Arch != "" {
		filter["arch"] = params.Arch
	}
	for i := 0; i < len(list); i++ {
		engine.Where(filter).Model(&list[i]).Related(&list[i].List, "go_branch")
	}
	return list, nil
}
