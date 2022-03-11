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

type RoomUserSongDaoInterface interface {
	Insert(xl *xlog.Logger, roomUserSongDo *model.RoomUserSongDo) (*model.RoomUserSongDo, error)

	Select(xl *xlog.Logger, id string) (*model.RoomUserSongDo, error)

	SelectByRoomIdSongId(xl *xlog.Logger, roomId, songId string) (*model.RoomUserSongDo, error)

	SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.RoomUserSongDo, error)

	Update(xl *xlog.Logger, roomUserSongDo *model.RoomUserSongDo) error

	ListByRoomId(xl *xlog.Logger, roomId string, pageNum, pageSize int) ([]model.RoomUserSongDo, int, int, error)
}

type RoomUserSongDaoService struct {
	client           *mgo.Session
	roomUserSongColl *mgo.Collection
	xl               *xlog.Logger
}

func NewRoomUserSongDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*RoomUserSongDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-room-user-song")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	roomUserSongColl := client.DB(config.Database).C(dao.CollectionRoomUserSong)
	return &RoomUserSongDaoService{
		client,
		roomUserSongColl,
		xl,
	}, nil
}

func (r *RoomUserSongDaoService) Insert(xl *xlog.Logger, roomUserSongDo *model.RoomUserSongDo) (*model.RoomUserSongDo, error) {
	if xl == nil {
		xl = r.xl
	}
	roomUserSongDo.Id = bson.NewObjectId().Hex()
	roomUserSongDo.CreatedTime = time.Now()
	roomUserSongDo.UpdatedTime = time.Now()
	err := r.roomUserSongColl.Insert(roomUserSongDo)
	if err != nil {
		xl.Error("insert into room_user_song failed.")
		return nil, err
	}
	return roomUserSongDo, nil
}

func (r *RoomUserSongDaoService) Select(xl *xlog.Logger, id string) (*model.RoomUserSongDo, error) {
	if xl == nil {
		xl = r.xl
	}
	var roomUserSongDo model.RoomUserSongDo
	err := r.roomUserSongColl.FindId(id).One(&roomUserSongDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_song.")
		} else {
			xl.Error("select room_user_song failed.")
		}
		return nil, err
	}
	return &roomUserSongDo, nil
}

func (r *RoomUserSongDaoService) SelectByRoomIdSongId(xl *xlog.Logger, roomId, songId string) (*model.RoomUserSongDo, error) {
	if xl == nil {
		xl = r.xl
	}
	var roomUserSongDo model.RoomUserSongDo
	err := r.roomUserSongColl.Find(bson.M{"room_id": roomId, "song_id": songId}).One(&roomUserSongDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_song.")
		} else {
			xl.Error("select from room_user_song failed.")
		}
		return nil, err
	}
	return &roomUserSongDo, nil
}

func (r *RoomUserSongDaoService) SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.RoomUserSongDo, error) {
	if xl == nil {
		xl = r.xl
	}
	var roomUserSongDo model.RoomUserSongDo
	err := r.roomUserSongColl.Find(bson.M{"room_id": roomId, "user_id": userId}).One(&roomUserSongDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from room_user_song.")
		} else {
			xl.Error("select from room_user_song failed.")
		}
		return nil, err
	}
	return &roomUserSongDo, nil
}

func (r *RoomUserSongDaoService) Update(xl *xlog.Logger, roomUserSongDo *model.RoomUserSongDo) error {
	if xl == nil {
		xl = r.xl
	}
	roomUserSongDo.UpdatedTime = time.Now()
	err := r.roomUserSongColl.UpdateId(roomUserSongDo.Id, roomUserSongDo)
	if err != nil {
		xl.Error("update room_user_song failed.")
		return err
	}
	return nil
}

func (r *RoomUserSongDaoService) ListByRoomId(xl *xlog.Logger, roomId string, pageNum, pageSize int) ([]model.RoomUserSongDo, int, int, error) {
	if xl == nil {
		xl = r.xl
	}
	var roomUserSongDos []model.RoomUserSongDo
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	err := r.roomUserSongColl.Find(bson.M{"room_id": roomId, "status": model.RoomUserSongAvailable}).Sort("-created_time").Skip(skip).Limit(limit).All(&roomUserSongDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find those records from room_user_song.")
		} else {
			xl.Error("list room_user_song failed.")
		}
		return nil, 0, 0, err
	}
	total, _ := r.roomUserSongColl.Find(bson.M{"room_id": roomId, "status": model.RoomUserSongAvailable}).Count()
	return roomUserSongDos, total, len(roomUserSongDos), nil
}
