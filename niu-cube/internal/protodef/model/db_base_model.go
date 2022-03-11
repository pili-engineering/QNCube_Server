package model

import "time"

type BaseRoomDo struct {
	Id             string        `bson:"_id" json:"roomId"`
	Title          string        `bson:"title" json:"title"`
	Image          string        `bson:"image" json:"image"`
	Desc           string        `bson:"desc" json:"desc"`
	Status         int           `bson:"status" json:"status"`
	Creator        string        `bson:"creator" json:"creator"`
	Type           string        `bson:"type" json:"type"`
	QiniuIMGroupId int64         `bson:"qiniu_im_group_id" json:"qiniuImGroupId"`
	InvitationCode string        `bson:"invitation_code" json:"-"`
	CreatedTime    time.Time     `bson:"created_time" json:"-"`
	UpdatedTime    time.Time     `bson:"updated_time" json:"-"`
	BaseRoomAttrs  []BaseEntryDo `bson:"base_room_attrs" json:"attrs"`
	BaseRoomParams []BaseEntryDo `bson:"base_room_params" json:"params"`
}

type BaseUserDo struct {
	Id            string        `bson:"_id" json:"userId"`
	Name          string        `bson:"name" json:"name"`
	Nickname      string        `bson:"nickname" json:"nickname"`
	Avatar        string        `bson:"avatar" json:"avatar"`
	Status        int           `bson:"status" json:"status"`
	Profile       string        `bson:"profile" json:"profile"`
	CreatedTime   time.Time     `bson:"created_time" json:"-"`
	UpdatedTime   time.Time     `bson:"updated_time" json:"-"`
	BaseUserAttrs []BaseEntryDo `bson:"base_user_attrs" json:"attrs"`
}

type BaseMicDo struct {
	Id            string        `bson:"_id" json:"micId"`
	Name          string        `bson:"name" json:"name"`
	Status        int           `bson:"status" json:"status"`
	CreatedTime   time.Time     `bson:"created_time" json:"-"`
	UpdatedTime   time.Time     `bson:"updated_time" json:"-"`
	Type          string        `bson:"type" json:"type"`
	BaseMicAttrs  []BaseEntryDo `bson:"base_mic_attrs"`
	BaseMicParams []BaseEntryDo `bson:"base_mic_params"`
}

type BaseEntryDo struct {
	Key    string      `bson:"key" json:"key"`
	Value  interface{} `bson:"value" json:"value"`
	Status int         `bson:"status" json:"status"`
}

type BaseRoomUserDo struct {
	Id                string    `bson:"_id"`
	RoomId            string    `bson:"room_id"`
	UserId            string    `bson:"user_id"`
	UserRole          string    `bson:"user_role"`
	Status            int       `bson:"status"`
	CreatedTime       time.Time `bson:"created_time"`
	UpdatedTime       time.Time `bson:"updated_time"`
	LastHeartbeatTime time.Time `bson:"last_heartbeat_time"`
}

type BaseUserMicDo struct {
	Id string `bson:"_id"`
	// RoomId 冗余字段方便查询
	RoomId      string    `bson:"room_id"`
	UserId      string    `bson:"user_id"`
	MicId       string    `bson:"mic_id"`
	Status      int       `bson:"status"`
	CreatedTime time.Time `bson:"created_time"`
	UpdatedTime time.Time `bson:"updated_time"`
	// 绑定在用户和麦上的一些附加信息
	UserExtension string `bson:"user_extension"`
}

type BaseRoomMicDo struct {
	Id          string    `bson:"_id"`
	RoomId      string    `bson:"room_id"`
	MicId       string    `bson:"mic_id"`
	Index       int       `bson:"index"`
	Status      int       `bson:"status"`
	CreatedTime time.Time `bson:"created_time"`
	UpdatedTime time.Time `bson:"updated_time"`
}

const (
	_ = iota
	BaseRoomCreated
	BaseRoomDestroyed
)

const (
	_ = iota
	BaseEntryAvailable
	BaseEntryUnavailable
)

const (
	_ = iota
	BaseUserLogin
	BaseUserLogout
	BaseUserSignIn
	BaseUserSignOut
	BaseUserTimeout
)

const (
	_ = iota
	BaseRoomUserJoin
	BaseRoomUserLeave
	BaseRoomUserTimeout
)

const (
	_ = iota
	BaseMicAvailable
	BaseMicUnavailable
)

const (
	_ = iota
	BaseRoomMicUsed
	BaseRoomMicUnused
)

const (
	_ = iota
	BaseUserMicHold
	BaseUserMicNonHold
)

const (
	// BaseTypeKtv 固定麦位数
	BaseTypeKtv = "ktv"
	// BaseTypeClassroom 无限制麦位数
	BaseTypeClassroom = "classroom"
	// BaseTypeShow 秀场
	BaseTypeShow = "show"
	// BaseTypeMovie 电影
	BaseTypeMovie = "movie"
	// BaseTypeExam shv在线考试
	BaseTypeExam = "onlineExam"
	// BaseTypeVoiceChat 语聊房
	BaseTypeVoiceChat = "voiceChatRoom"
)

const (
	BaseMicTypeMain      = "main"
	BaseMicTypeSecondary = "secondary"
)
