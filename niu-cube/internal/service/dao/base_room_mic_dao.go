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

type BaseRoomMicDaoInterface interface {
	Insert(xl *xlog.Logger, baseRoomMicDo *model.BaseRoomMicDo) (*model.BaseRoomMicDo, error)

	// DeleteByRoomIdMicId 做假删除，尽量不要调用这个方法
	DeleteByRoomIdMicId(xl *xlog.Logger, roomId, micId string) error

	// Update 通过更新数据表来实现删除
	Update(xl *xlog.Logger, baseRoomMic *model.BaseRoomMicDo) error

	// Select 返回还在麦位的用户
	Select(xl *xlog.Logger, roomId, micId string) (*model.BaseRoomMicDo, error)

	ListByRoomId(xl *xlog.Logger, roomId string) ([]model.BaseRoomMicDo, error)
}

type BaseRoomMicDaoService struct {
	client          *mgo.Session
	baseRoomMicColl *mgo.Collection
	xl              *xlog.Logger
}

func NewBaseRoomMicDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*BaseRoomMicDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-base-room-mic")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	baseRoomMicColl := client.DB(config.Database).C(dao.CollectionBaseRoomMic)
	return &BaseRoomMicDaoService{
		client,
		baseRoomMicColl,
		xl,
	}, nil
}

func (b *BaseRoomMicDaoService) Insert(xl *xlog.Logger, baseRoomMicDo *model.BaseRoomMicDo) (*model.BaseRoomMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	baseRoomMicDo.Id = bson.NewObjectId().Hex()
	baseRoomMicDo.CreatedTime = time.Now()
	baseRoomMicDo.UpdatedTime = time.Now()
	err := b.baseRoomMicColl.Insert(baseRoomMicDo)
	if err != nil {
		xl.Error("insert into base_room_mic failed.")
		return nil, err
	}
	return baseRoomMicDo, nil
}

func (b *BaseRoomMicDaoService) DeleteByRoomIdMicId(xl *xlog.Logger, roomId, micId string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseRoomMicColl.Remove(bson.M{"room_id": roomId, "mic_id": micId})
	if err != nil {
		xl.Error("delete from base_room_mic by roomId:[%s] micId:[%s] failed.", roomId, micId)
		return err
	}
	return nil
}

func (b *BaseRoomMicDaoService) Update(xl *xlog.Logger, baseRoomMic *model.BaseRoomMicDo) error {
	if xl == nil {
		xl = b.xl
	}
	baseRoomMic.UpdatedTime = time.Now()
	err := b.baseRoomMicColl.UpdateId(baseRoomMic.Id, baseRoomMic)
	if err != nil {
		xl.Error("update base_room_mic failed.")
		return err
	}
	return nil
}

func (b *BaseRoomMicDaoService) Select(xl *xlog.Logger, roomId, micId string) (*model.BaseRoomMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var roomMic model.BaseRoomMicDo
	err := b.baseRoomMicColl.Find(bson.M{"room_id": roomId, "mic_id": micId, "status": model.BaseRoomMicUsed}).One(&roomMic)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't find this record from base_room.")
		} else {
			xl.Error("select from base_room failed.")
		}
		return nil, err
	}
	return &roomMic, nil
}

func (b *BaseRoomMicDaoService) ListByRoomId(xl *xlog.Logger, roomId string) ([]model.BaseRoomMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var roomMics []model.BaseRoomMicDo
	err := b.baseRoomMicColl.Find(bson.M{"room_id": roomId}).All(&roomMics)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't list those records from base_room_mic.")
		} else {
			xl.Error("list from base_room_mic by roomId:[%s] failed.", roomId)
		}
		return nil, err
	}
	return roomMics, nil
}
