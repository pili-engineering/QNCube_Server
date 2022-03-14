package model

import "time"

type ImageFileDo struct {
	ID         string    `json:"id" bson:"_id"`
	FileName   string    `json:"fileName" bson:"fileName"`
	FileUrl    string    `json:"fileUrl" bson:"fileUrl"`
	Status     int       `json:"status" bson:"status"`
	CreateTime time.Time `json:"-" bson:"createTime"`
	UpdateTime time.Time `json:"-" bson:"updateTime"`
}

const (
	_ = iota
	ImageFileStatusNormal
)
