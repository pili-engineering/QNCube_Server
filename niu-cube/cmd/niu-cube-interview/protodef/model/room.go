package model

import (
	"encoding/json"
	"time"
)

// 用户如何调用？
// 抽离房间概念
// 房间可能多人 design for 多人
type RoomAccount struct {
	ID                string            `json:"id" bson:"id"`
	AccountId         string            `json:"account_id" bson:"account_id"`
	RoomRole          string            `json:"room_role" bson:"room_role"`                     // guest user admin
	AccountRoomStatus AccountRoomStatus `json:"account_room_status" bson:"account_room_status"` // nil in out
	RoomId            string            `json:"room_id" bson:"room_id"`
	CreateAt          time.Time         `json:"create_at" bson:"create_at"`
	UpdateAt          time.Time         `json:"update_at" bson:"update_at"`
}
type AccountRoomStatus string

const (
	AccountRoomStatusNil = "nil"
	AccountRoomStatusIn  = "in"
	AccountRoomStatusOut = "out"
)

// 除了基本字段 存储业务信息
type Room struct {
	ID            string      `json:"id" bson:"_id"` // AppId+Name
	Name          string      `json:"name" bson:"name"`
	AppId         string      `json:"app_id" bson:"app_id"` // -> biz_id
	CreatorId     string      `json:"creator_id" bson:"creator_id"`
	BizExtraId    string      `json:"biz_extra_id" bson:"biz_extra_id"` // authCode isAuth
	BizExtraValue interface{} `json:"biz_extra"`
	ValidBefore   time.Time   `json:"valid_before" bson:"valid_before"`
	CreateAt      time.Time   `json:"create_at" bson:"create_at"`
	UpdateAt      time.Time   `json:"update_at" bson:"update_at"`
}

// FromFlattenMap db return one-item-array BizExtra, here we remove the array
func NewRoomFromFlattenMap(m FlattenMap) *Room {
	val, _ := json.Marshal(&m)
	r := &Room{}
	err := json.Unmarshal(val, r)
	if err != nil {
		panic("err unmarshal room from flatten map")
	}
	if arr, ok := r.BizExtraValue.([]interface{}); ok && len(arr) > 0 {
		r.BizExtraValue = arr[0]
	}
	return r
}

func (r Room) GetBizExtra() FlattenMap {
	if val, ok := r.BizExtraValue.(map[string]interface{}); ok {
		return FlattenMap(val)
	} else {
		return make(map[string]interface{})
	}
}

type BizExtra struct {
	ID     string
	RoomID string
}

type InterviewExtra struct {
	BizExtra
	Title           string    `json:"title" bson:"title"`
	StartTime       time.Time `json:"startTime" bson:"startTime"`
	EndTime         time.Time `json:"endTime" bson:"endTime"`
	Goverment       string    `json:"goverment" bson:"goverment"`
	Career          string    `json:"career" bson:"career"`
	IsRecord        bool      `json:"isRecord" bson:"isRecord"`
	Recorded        bool      `json:"recorded" bson:"recorded"`
	IsAuth          bool      `json:"isAuth" bson:"isAuth"`
	AuthCode        string    `json:"authCode" bson:"authCode"`
	Status          int       `json:"status" bson:"status"`
	CreateTime      time.Time `json:"createTime" bson:"createTime"`
	UpdateTime      time.Time `json:"updateTime" bson:"updateTime"`
	Creator         string    `json:"creator" bson:"creator"`
	Interviewer     string    `json:"interviewer" bson:"interviewer"`
	InterviewerName string    `json:"interviewerName" bson:"interviewerName"`
	Candidate       string    `json:"candidate" bson:"candidate"`
	CandidateName   string    `json:"candidateName" bson:"candidateName"`
	AppletQrcode    string    `json:"applet_qrcode" bson:"applet_qrcode"`
}
