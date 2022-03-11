package model

import "time"

type RepairRoomDo struct {
	ID             string    `json:"id" bson:"_id"`
	RoomId         string    `json:"roomId" bson:"roomId"`
	Title          string    `json:"title" bson:"title"`
	Status         int       `json:"status" bson:"status"`
	Image          string    `json:"image" bson:"image"`
	CreateTime     time.Time `json:"createTime" bson:"createTime"`
	UpdateTime     time.Time `json:"updateTime" bson:"updateTime"`
	Creator        string    `json:"creator" bson:"creator"`
	Updator        string    `json:"updator" bson:"updator"`
	QiniuIMGroupId int64     `json:"qiniuIMGroupId" bson:"qiniuIMGroupId"`
}

type RepairRoomUserDo struct {
	ID                string    `json:"id" bson:"_id"`
	RoomId            string    `json:"roomId" bson:"roomId"`
	UserID            string    `json:"userId" bson:"userId"`
	Role              string    `json:"role" bson:"role"`
	Status            int       `json:"status" bson:"status"`
	CreateTime        time.Time `json:"createTime" bson:"createTime"`
	UpdateTime        time.Time `json:"updateTime" bson:"updateTime"`
	LastHeartBeatTime time.Time `json:"last_heart_beat_time" bson:"lastHeartBeatTime"`
}

type RepairRoomStatusCode int
type RepairRoomUserStatusCode int
type RepairRoomRoleType string
type RepairRoomOptionTitle string

const (
	RepairRoomStatusCodeClose      RepairRoomStatusCode     = 0
	RepairRoomStatusCodeOpen       RepairRoomStatusCode     = 1
	RepairRoomUserStatusCodeDelete RepairRoomUserStatusCode = 0
	RepairRoomUserStatusCodeNormal RepairRoomUserStatusCode = 1

	RepairRoomRoleStudent     RepairRoomRoleType    = "student"
	RepairRoomRoleProfessor   RepairRoomRoleType    = "professor"
	RepairRoomRoleStaff       RepairRoomRoleType    = "staff"
	RepairRoomOptionStudent   RepairRoomOptionTitle = "学生进入"
	RepairRoomOptionProfessor RepairRoomOptionTitle = "专家进入"
	RepairRoomOptionStaff     RepairRoomOptionTitle = "检修员进入"
)
