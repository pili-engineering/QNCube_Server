package form

import (
	"fmt"
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/qiniu/x/xlog"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"reflect"
	"regexp"
)

const (
	ErrVersionMsg = "版本号必须符合 V1.1.1 格式"
)

var (
	ErrVersionIdNeeded = fmt.Errorf("versionId 是必要的")
)

var (
	defaultLogger = xlog.New("Form Validate")
)

type VersionCreateForm struct {
	//ID string `json:"id" bson:"id"`
	AppName     string                   `json:"app_name" bson:"app_name" form:"app_name"`
	Platform    model.VersionPlatform    `json:"platform" bson:"platform" form:"platform"`
	Version     string                   `json:"version" bson:"version" form:"version"` //V1.1.1 Regex: V\d\.\d\.\d\.
	CommitHash  string                   `json:"commit_hash" bson:"commit_hash" form:"commit_hash"`
	UpgradeCode model.VersionUpgradeCode `json:"upgrade_code" bson:"upgrade_code" form:"upgrade_code"`
	URL         string                   `json:"url" bson:"url" form:"url"`          //	跳转链接
	Prompt      string                   `json:"prompt" bson:"prompt" form:"prompt"` // 升级提示语
}

func shallowIn(tag string, target interface{}, values ...interface{}) error {
	V := reflect.ValueOf(target)
	T := reflect.TypeOf(target)
	var value interface{}
	switch T.Kind() {
	case reflect.String:
		value = V.String()
	case reflect.Int, reflect.Int64:
		value = int(V.Int())
	default:
		value = V.Interface()
	}
	ok := false
	for _, item := range values {
		if value == item {
			ok = true
			continue
		}
		//fmt.Printf("%t %t\n",value,item)
	}
	if !ok {
		return fmt.Errorf("%v 必须在 %v中", tag, values)
	}
	return nil
}

func (v *VersionCreateForm) Validate() error {
	versionReg := regexp.MustCompile("[vV][0-9]\\.[0-9]\\.[0-9]")
	err := shallowIn("platform", v.Platform, model.VersionPlatformAndroid, model.VersionPlatformIos)
	if err != nil {
		return err
	}
	err = shallowIn("upgrade_code", v.UpgradeCode, model.VersionUpgradeCodeNotify, model.VersionUpgradeCodeForce)
	if err != nil {
		return err
	}
	err = validation.ValidateStruct(v,
		validation.Field(&v.AppName, validation.Required.Error("必填")),
		validation.Field(&v.Platform, validation.Required.Error("必填")),
		validation.Field(&v.Version, validation.Required.Error("必填"), validation.Match(versionReg).Error(ErrVersionMsg)),
	)
	val, ok := err.(validation.InternalError)
	if ok {
		defaultLogger.Errorf("error validate error: %v", val)
	}
	return err
}

// VersionFilterForm for filtering version
type VersionFilterForm struct {
	//ID string `json:"id" bson:"id"`
	AppName     string                   `json:"app_name" bson:"app_name" form:"app_name"`
	Platform    model.VersionPlatform    `json:"platform" bson:"platform" form:"platform"`
	Version     string                   `json:"version" bson:"version" form:"version"` //V1.1.1 Regex: V\d\.\d\.\d\.
	CommitHash  string                   `json:"commit_hash" bson:"commit_hash" form:"commit_hash"`
	UpgradeCode model.VersionUpgradeCode `json:"upgrade_code" bson:"upgrade_code" form:"upgrade_code"`
}

// TODO: replace with reflect
func (v *VersionFilterForm) Filter() interface{} {
	filter := make(map[string]interface{}, 0)
	if v.AppName != "" {
		filter["app_name"] = v.AppName
	}
	if v.Platform != "" {
		filter["platform"] = v.Platform
	}
	if v.Version != "" {
		filter["version"] = v.Version
	}
	if v.CommitHash != "" {
		filter["commit_hash"] = v.CommitHash
	}
	if v.UpgradeCode != 0 {
		filter["upgrade_code"] = v.UpgradeCode
	}
	return filter
}
