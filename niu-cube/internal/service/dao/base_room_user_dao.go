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

type BaseRoomUserDaoInterface interface {
	Insert(xl *xlog.Logger, baseRoomUserDo *model.BaseRoomUserDo) (*model.BaseRoomUserDo, error)

	// SelectByRoomIdUserId 只会返回尚且在房间的用户，已经离线的不算
	SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.BaseRoomUserDo, error)

	Update(xl *xlog.Logger, baseRoomUserDo *model.BaseRoomUserDo) error

	// ListByRoomId 依旧只会返回还在房间的用户
	ListByRoomId(xl *xlog.Logger, roomId string) ([]model.BaseRoomUserDo, error)

	ListByHeartbeatTimeout(xl *xlog.Logger, thresholdTimeout time.Time) ([]model.BaseRoomUserDo, error)

	// DeleteByRoomIdUserId 最好不要调用
	DeleteByRoomIdUserId(xl *xlog.Logger, roomId, userId string) error
}

type BaseRoomUserDaoService struct {
	client           *mgo.Session
	baseRoomUserColl *mgo.Collection
	xl               *xlog.Logger
}

func NewBaseRoomUserDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*BaseRoomUserDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-base-room-user")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	baseRoomUserColl := client.DB(config.Database).C(dao.CollectionBaseRoomUser)
	return &BaseRoomUserDaoService{
		client,
		baseRoomUserColl,
		xl,
	}, nil
}

func (b *BaseRoomUserDaoService) Insert(xl *xlog.Logger,
	baseRoomUserDo *model.BaseRoomUserDo) (*model.BaseRoomUserDo, error) {
	if xl == nil {
		xl = b.xl
	}
	baseRoomUserDo.Id = bson.NewObjectId().Hex()
	baseRoomUserDo.CreatedTime = time.Now()
	baseRoomUserDo.UpdatedTime = time.Now()
	err := b.baseRoomUserColl.Insert(baseRoomUserDo)
	if err != nil {
		xl.Error("insert into base_room_user failed.")
		return nil, err
	}
	return baseRoomUserDo, nil
}

func (b *BaseRoomUserDaoService) SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.BaseRoomUserDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var roomUser model.BaseRoomUserDo
	err := b.baseRoomUserColl.Find(bson.M{"room_id": roomId, "user_id": userId, "status": model.BaseRoomUserJoin}).One(&roomUser)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't find this record from base_room_user by roomId:[%s] userId:[%s].", roomId, userId)
		} else {
			xl.Error("select from base_room_user failed.")
		}
		return nil, err
	}
	return &roomUser, nil
}

func (b *BaseRoomUserDaoService) Update(xl *xlog.Logger, baseRoomUserDo *model.BaseRoomUserDo) error {
	if xl == nil {
		xl = b.xl
	}
	baseRoomUserDo.UpdatedTime = time.Now()
	err := b.baseRoomUserColl.UpdateId(baseRoomUserDo.Id, baseRoomUserDo)
	if err != nil {
		xl.Error("update base_room_user failed.")
		return err
	}
	return nil
}

func (b *BaseRoomUserDaoService) ListByRoomId(xl *xlog.Logger, roomId string) ([]model.BaseRoomUserDo, error) {
	if xl == nil {
		xl = b.xl
	}
	roomUserDos := make([]model.BaseRoomUserDo, 0, 1)
	err := b.baseRoomUserColl.Find(bson.M{"room_id": roomId, "status": model.BaseRoomUserJoin}).Sort("-updated_time").All(&roomUserDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't list those records:[%s] base_room_user.", roomId)
		} else {
			xl.Error("list base_room_user failed.")
		}
		return nil, err
	}
	return roomUserDos, nil
}

func (b *BaseRoomUserDaoService) ListByHeartbeatTimeout(xl *xlog.Logger, thresholdTimeout time.Time) ([]model.BaseRoomUserDo, error) {
	if xl == nil {
		xl = b.xl
	}
	roomUserDos := make([]model.BaseRoomUserDo, 0, 1)
	err := b.baseRoomUserColl.Find(bson.M{"last_heartbeat_time": bson.M{"$lt": thresholdTimeout}, "status": model.BaseRoomUserJoin}).All(&roomUserDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't list those records:[%s] base_room_user.", thresholdTimeout)
		} else {
			xl.Error("list base_room_user failed.")
		}
		return nil, err
	}
	return roomUserDos, nil
}

func (b *BaseRoomUserDaoService) DeleteByRoomIdUserId(xl *xlog.Logger, roomId, userId string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseRoomUserColl.Remove(bson.M{"room_id": roomId, "user_id": userId})
	if err != nil {
		xl.Error("delete from base_room_user failed.")
		return err
	}
	return nil
}
