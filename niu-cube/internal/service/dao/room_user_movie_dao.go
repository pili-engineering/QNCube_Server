package dao

import (
	"time"

	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
)

type RoomUserMovieInterface interface {
	Insert(xl *xlog.Logger, roomUserMovieDo *model.RoomUserMovieDo) error

	Delete(xl *xlog.Logger, roomUserMovieId string) error

	Update(xl *xlog.Logger, roomUserMovieDo *model.RoomUserMovieDo) error

	Select(xl *xlog.Logger, roomUserMovieId string) (*model.RoomUserMovieDo, error)

	SelectByRoomIdMovieId(xl *xlog.Logger, roomId, movieId string) (*model.RoomUserMovieDo, error)

	SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.RoomUserMovieDo, error)

	SelectByRoomIdPlaying(xl *xlog.Logger, roomId string) (*model.RoomUserMovieDo, error)

	ListByRoomId(xl *xlog.Logger, roomId string, pageNum, pageSize int) ([]model.RoomUserMovieDo, int, error)
}

type RoomUserMovieDaoService struct {
	client            *mgo.Session
	roomUserMovieColl *mgo.Collection
	xl                *xlog.Logger
}

func NewRoomUserMovieService(xl *xlog.Logger, config *utils.MongoConfig) (*RoomUserMovieDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-room-user-movie")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Error("failed to create mongo client, error: %v", err)
		return nil, err
	}
	roomUserMovieColl := client.DB(config.Database).C(dao.CollectionRoomUserMovie)
	return &RoomUserMovieDaoService{
		client,
		roomUserMovieColl,
		xl,
	}, nil
}

func (r *RoomUserMovieDaoService) Insert(xl *xlog.Logger, roomUserMovieDo *model.RoomUserMovieDo) error {
	if xl == nil {
		xl = r.xl
	}
	roomUserMovieDo.Id = bson.NewObjectId().Hex()
	roomUserMovieDo.CreatedTime = time.Now()
	roomUserMovieDo.UpdatedTime = time.Now()
	err := r.roomUserMovieColl.Insert(roomUserMovieDo)
	if err != nil {
		xl.Error("insert into room_user_movie failed.")
		return err
	}
	return nil
}

func (r *RoomUserMovieDaoService) Delete(xl *xlog.Logger, roomUserMovieId string) error {
	if xl == nil {
		xl = r.xl
	}
	err := r.roomUserMovieColl.RemoveId(roomUserMovieId)
	if err != nil {
		xl.Error("delete from room_user_movie failed.")
		return err
	}
	return nil
}

func (r *RoomUserMovieDaoService) Update(xl *xlog.Logger, roomUserMovieDo *model.RoomUserMovieDo) error {
	if xl == nil {
		xl = r.xl
	}
	roomUserMovieDo.UpdatedTime = time.Now()
	err := r.roomUserMovieColl.UpdateId(roomUserMovieDo.Id, roomUserMovieDo)
	if err != nil {
		xl.Error("update room_user_movie failed.")
		return err
	}
	return nil
}

func (r *RoomUserMovieDaoService) Select(xl *xlog.Logger, roomUserMovieId string) (*model.RoomUserMovieDo, error) {
	if xl == nil {
		xl = r.xl
	}
	roomUserMovieDo := model.RoomUserMovieDo{}
	err := r.roomUserMovieColl.FindId(roomUserMovieId).One(&roomUserMovieDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_movie")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &roomUserMovieDo, nil
}

func (r *RoomUserMovieDaoService) SelectByRoomIdMovieId(xl *xlog.Logger, roomId, movieId string) (*model.RoomUserMovieDo, error) {
	if xl == nil {
		xl = r.xl
	}
	roomUserMovieDo := model.RoomUserMovieDo{}
	err := r.roomUserMovieColl.Find(bson.M{"room_id": roomId, "movie_id": movieId, "status": model.RoomUserMovieAvailable}).One(&roomUserMovieDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_movie")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &roomUserMovieDo, nil
}

func (r *RoomUserMovieDaoService) SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.RoomUserMovieDo, error) {
	if xl == nil {
		xl = r.xl
	}
	roomUserMovieDo := model.RoomUserMovieDo{}
	err := r.roomUserMovieColl.Find(bson.M{"room_id": roomId, "user_id": userId, "status": model.RoomUserMovieAvailable}).One(&roomUserMovieDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_movie")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &roomUserMovieDo, nil
}

func (r *RoomUserMovieDaoService) SelectByRoomIdPlaying(xl *xlog.Logger, roomId string) (*model.RoomUserMovieDo, error) {
	if xl == nil {
		xl = r.xl
	}
	result := model.RoomUserMovieDo{}
	err := r.roomUserMovieColl.Find(bson.M{"status": model.RoomUserMovieAvailable, "is_playing": true, "room_id": roomId}).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_movie")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &result, nil
}

func (r *RoomUserMovieDaoService) ListByRoomId(xl *xlog.Logger, roomId string, pageNum, pageSize int) ([]model.RoomUserMovieDo, int, error) {
	if xl == nil {
		xl = r.xl
	}
	roomUserMovieDos := make([]model.RoomUserMovieDo, 0, pageSize)
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	err := r.roomUserMovieColl.Find(bson.M{"room_id": roomId, "status": model.RoomUserMovieAvailable}).Sort("created_time").Skip(skip).Limit(limit).All(&roomUserMovieDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find those records from song.")
		} else {
			xl.Error("list song failed.")
		}
		return nil, 0, err
	}
	total, _ := r.roomUserMovieColl.Find(bson.M{"room_id": roomId, "status": model.RoomUserMovieAvailable}).Count()
	return roomUserMovieDos, total, nil
}
