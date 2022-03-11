package model

import "time"

type MovieDo struct {
	Id          string    `bson:"_id" json:"movieId"`
	Name        string    `bson:"name" json:"name"`
	Director    string    `bson:"director" json:"director"`
	Image       string    `bson:"image" json:"image"`
	ActorList   []string  `bson:"actor_list" json:"actorList"`
	KindList    []string  `bson:"kind_list" json:"kindList"`
	Duration    uint64    `bson:"duration" json:"duration"`
	PlayUrl     string    `bson:"play_url" json:"playUrl"`
	Lyrics      string    `bson:"lyrics" json:"lyrics"`
	Desc        string    `bson:"desc" json:"desc"`
	DoubanScore float64   `bson:"douban_score" json:"doubanScore"`
	ImdbScore   float64   `bson:"imdb_score" json:"imdbScore"`
	ReleaseTime time.Time `bson:"release_time" json:"releaseTime"`
	CreatedTime time.Time `bson:"created_time" json:"-"`
	UpdatedTime time.Time `bson:"updated_time" json:"-"`
	Status      int       `bson:"status" json:"-"`
}

type RoomUserMovieDo struct {
	Id              string    `bson:"_id"`
	RoomId          string    `bson:"room_id"`
	UserId          string    `bson:"user_id"`
	MovieId         string    `bson:"movie_id"`
	RoomMaster      bool      `bson:"is_room_master"`
	CurrentSchedule uint64    `bson:"current_schedule"`
	Playing         bool      `bson:"is_playing"`
	CreatedTime     time.Time `bson:"created_time"`
	UpdatedTime     time.Time `bson:"updated_time"`
	Status          int       `bson:"status"`
}

const (
	_ = iota
	MovieAvailable
	MovieUnavailable
)

const (
	_ = iota
	RoomUserMovieAvailable
	RoomUserMovieUnavailable
)
