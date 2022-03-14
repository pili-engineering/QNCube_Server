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

type BaseUserMicDaoInterface interface {
	Insert(xl *xlog.Logger, baseUserMic *model.BaseUserMicDo) (*model.BaseUserMicDo, error)

	// Update 用于麦位数量固定的场景，更新麦位占有属性
	Update(xl *xlog.Logger, baseUserMic *model.BaseUserMicDo) error

	DeleteByUserIdMicId(xl *xlog.Logger, userId, micId string) error

	Delete(xl *xlog.Logger, id string) error

	SelectByRoomIdMicId(xl *xlog.Logger, roomId, micId string) (*model.BaseUserMicDo, error)

	// SelectByRoomIdUserId 这里有潜在的问题，就是如果这样写就无法保证同一个用户存在多个场景
	SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.BaseUserMicDo, error)

	// ListByRoomId 只会列出有人使用的麦
	ListByRoomId(xl *xlog.Logger, roomId string) ([]model.BaseUserMicDo, error)

	// ListByUserId 这个用于允许整个系统存在同一用户且多场景的情形
	ListByUserId(xl *xlog.Logger, userId string) ([]model.BaseUserMicDo, error)
}

type BaseUserMicDaoService struct {
	client          *mgo.Session
	baseUserMicColl *mgo.Collection
	xl              *xlog.Logger
}

func NewBaseUserMicDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*BaseUserMicDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-base-user-mic")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	baseUserMicColl := client.DB(config.Database).C(dao.CollectionBaseUserMic)
	return &BaseUserMicDaoService{
		client,
		baseUserMicColl,
		xl,
	}, nil
}

func (b *BaseUserMicDaoService) Insert(xl *xlog.Logger, baseUserMic *model.BaseUserMicDo) (*model.BaseUserMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	baseUserMic.Id = bson.NewObjectId().Hex()
	baseUserMic.CreatedTime = time.Now()
	baseUserMic.UpdatedTime = time.Now()
	err := b.baseUserMicColl.Insert(baseUserMic)
	if err != nil {
		xl.Error("insert into base_user_mic failed.")
		return nil, err
	}
	return baseUserMic, nil
}

func (b *BaseUserMicDaoService) Update(xl *xlog.Logger, baseUserMic *model.BaseUserMicDo) error {
	if xl == nil {
		xl = b.xl
	}
	baseUserMic.UpdatedTime = time.Now()
	err := b.baseUserMicColl.UpdateId(baseUserMic.Id, baseUserMic)
	if err != nil {
		xl.Error("update base_user_mic failed.")
		return err
	}
	return nil
}

func (b *BaseUserMicDaoService) DeleteByUserIdMicId(xl *xlog.Logger, userId, micId string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseUserMicColl.Remove(bson.M{"user_id": userId, "mic_id": micId})
	if err != nil {
		xl.Error("delete from base_user_mic failed.")
		return err
	}
	return nil
}

func (b *BaseUserMicDaoService) Delete(xl *xlog.Logger, id string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseUserMicColl.RemoveId(id)
	if err != nil {
		xl.Error("delete from base_user_mic failed.")
		return err
	}
	return nil
}

func (b *BaseUserMicDaoService) SelectByRoomIdMicId(xl *xlog.Logger, roomId, micId string) (*model.BaseUserMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var userMic model.BaseUserMicDo
	err := b.baseUserMicColl.Find(bson.M{"room_id": roomId, "mic_id": micId, "status": model.BaseUserMicHold}).One(&userMic)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from base_user_mic.")
		} else {
			xl.Error("select from base_user_mic failed.")
		}
		return nil, err
	}
	return &userMic, nil
}

func (b *BaseUserMicDaoService) SelectByRoomIdUserId(xl *xlog.Logger, roomId, userId string) (*model.BaseUserMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var userMic model.BaseUserMicDo
	err := b.baseUserMicColl.Find(bson.M{"room_id": roomId, "user_id": userId, "status": model.BaseUserMicHold}).One(&userMic)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from base_user_mic.")
		} else {
			xl.Error("select from base_user_mic failed.")
		}
		return nil, err
	}
	return &userMic, nil
}

func (b *BaseUserMicDaoService) ListByRoomId(xl *xlog.Logger, roomId string) ([]model.BaseUserMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var userMicDos []model.BaseUserMicDo
	err := b.baseUserMicColl.Find(bson.M{"room_id": roomId, "status": model.BaseUserMicHold}).All(&userMicDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't list those records from base_user_mic.")
		} else {
			xl.Error("list base_user_mic failed.")
		}
		return nil, err
	}
	return userMicDos, nil
}

func (b *BaseUserMicDaoService) ListByUserId(xl *xlog.Logger, userId string) ([]model.BaseUserMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var userMicDos []model.BaseUserMicDo
	err := b.baseUserMicColl.Find(bson.M{"user_id": userId, "status": model.BaseUserMicHold}).All(&userMicDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't list those records from base_user_mic.")
		} else {
			xl.Error("list base_user_mic failed.")
		}
		return nil, err
	}
	return userMicDos, nil
}
