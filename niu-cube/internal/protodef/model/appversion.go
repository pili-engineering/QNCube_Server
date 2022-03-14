package model

import "time"

type AppVersion struct {
	Id          string    `bson:"_id" json:"-"`
	Version     string    `bson:"version" json:"version"`
	Msg         string    `bson:"msg" json:"msg"`
	PackagePage string    `bson:"package_page" json:"packagePage"`
	PackageUrl  string    `bson:"package_url" json:"packageUrl"`
	Arch        string    `bson:"arch" json:"arch"`
	CreatedTime time.Time `bson:"created_time" json:"-"`
}
