package model

import "time"

// IMUser 对应IM 用户信息。
type IMUser struct {
	UserID           string    `json:"id"`
	Username         string    `json:"name"`
	Token            string    `json:"token"`
	LastRegisterTime time.Time `json:"lastRegisterTime"`
	LastOnlineTime   time.Time `json:"lastOnlineTime"`
	LastOfflineTime  time.Time `json:"lastOfflineTime"`
}

// ImType represent IM module provider
type ImType int

const (
	ImTypeRongyun ImType = 1
	ImTypeQiniu   ImType = 2
)
