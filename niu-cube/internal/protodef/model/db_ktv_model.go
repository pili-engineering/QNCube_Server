package model

import "time"

type SongDo struct {
	Id               string    `bson:"_id" json:"songId"`
	Name             string    `bson:"name" json:"name"`
	Album            string    `bson:"album" json:"album"`
	Image            string    `bson:"image" json:"image"`
	Author           string    `bson:"author" json:"author"`
	Kind             string    `bson:"kind" json:"kind"`
	OriginUrl        string    `bson:"origin_url" json:"originUrl"`
	AccompanimentUrl string    `bson:"accompaniment_url" json:"accompanimentUrl"`
	Lyrics           string    `bson:"lyrics" json:"lyrics"`
	Status           int       `bson:"status" json:"-"`
	CreatedTime      time.Time `bson:"created_time" json:"-"`
	UpdatedTime      time.Time `bson:"updated_time" json:"-"`
}

type RoomUserSongDo struct {
	Id          string    `bson:"_id"`
	RoomId      string    `bson:"room_id"`
	UserId      string    `bson:"user_id"`
	SongId      string    `bson:"song_id"`
	Status      int       `bson:"status"`
	CreatedTime time.Time `bson:"created_time"`
	UpdatedTime time.Time `bson:"updated_time"`
}

const (
	_ = iota
	SongAvailable
	SongUnavailable
)

const (
	_ = iota
	RoomUserSongAvailable
	RoomUserSongUnavailable
)
