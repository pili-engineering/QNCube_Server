package form

import (
	"encoding/json"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	pderr "github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/errors"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
)

type AccountCreateForm struct {
	// 手机号，目前要求全局唯一。
	Phone string `json:"phone" bson:"phone" form:"phone"`
	// 邮箱
	Email string `json:"email" bson:"email" form:"email"`
	// TODO：支持账号密码登录。
	Password string `json:"password" bson:"password" form:"password"`
	// 用户昵称
	Nickname string `json:"nickname" bson:"nickname" form:"nickname"`
	// Kind
	Kind model.AccountKind
}

type AccountLoginForm struct {
	// 手机号，目前要求全局唯一。
	Phone string `json:"phone" bson:"phone" form:"phone"`
	// 邮箱
	Email string `json:"email" bson:"email" form:"email"`
	// TODO：支持账号密码登录。
	Password string `json:"password" bson:"password" form:"password"`
	Code     string `json:"code" bson:"code" form:"code"`

	LoginType string `json:"login_type" form:"login_type"`
}

func (i *AccountCreateForm) Validate() error {
	if i.Phone == "" && i.Email == "" {
		return pderr.ErrEmailOrPhoneRequired
	}
	err := validation.ValidateStruct(i,
		validation.Field(&i.Nickname, validation.Required, validation.Length(0, 100).Error(ErrTitleMsg)),
		validation.Field(&i.Phone, PhoneValidate(i.Phone)),
		validation.Field(&i.Email, is.Email.Error(pderr.ErrEmailFormatMsg)),
	)
	return err
}

func (i *AccountCreateForm) Map() map[string]interface{} {
	var res map[string]interface{}
	val, _ := json.Marshal(i)
	_ = json.Unmarshal(val, &res)
	return res
}

func (i *AccountLoginForm) Validate() error {
	err := validation.ValidateStruct(i,
		validation.Field(&i.Phone, PhoneValidate(i.Phone)),
		validation.Field(&i.Email, is.Email.Error(pderr.ErrEmailFormatMsg)),
		validation.Field(&i.LoginType, validation.In("sms", "passwd")), //TODO add to doc
	)
	return err
}

func (i *AccountLoginForm) Map() map[string]interface{} {
	var res map[string]interface{}
	val, _ := json.Marshal(i)
	_ = json.Unmarshal(val, &res)
	return res
}
