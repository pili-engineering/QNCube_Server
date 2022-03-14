package model

import (
	"encoding/json"
	"time"
)

// AccountDo 用户账号信息。
type Account struct {
	// 用户ID，作为数据库唯一标识。
	ID string `json:"id" bson:"_id"`
	// 手机号，目前要求全局唯一。
	Phone string `json:"phone" bson:"phone"`
	// TODO： 暂用邮箱
	Email string `json:"email" bson:"email"`
	// TODO：支持账号密码登录。
	Password string `json:"password" bson:"password"`
	// 用户昵称
	Nickname string `json:"nickname" bson:"nickname"`
	// Avartar 头像URL地址
	Avatar string `json:"avatar,omitempty" bson:"avatar,omitempty"`
	// RegisterIP 用户注册（首次登录）时使用的IP。
	RegisterIP string `json:"registerIP" bson:"registerIP"`
	// RegisterTime 用户注册（首次登录）时间。
	RegisterTime time.Time `json:"registerTime" bson:"registerTime"`
	// LastLoginTime 上次登录时间。
	LastLoginTime time.Time `json:"lastLoginTime" bson:"lastLoginTime"`
	// Kind
	Kind AccountKind
}

func (a *Account) Map() FlattenMap {
	val, _ := json.Marshal(a)
	var res map[string]interface{}
	_ = json.Unmarshal(val, &res)
	return res
}

type AccountKind string
