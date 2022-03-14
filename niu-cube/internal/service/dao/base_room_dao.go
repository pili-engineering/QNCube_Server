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

// BaseRoomDaoInterface 通用房间数据库相关操作
type BaseRoomDaoInterface interface {
	Insert(xl *xlog.Logger, baseRoomDo *model.BaseRoomDo) (*model.BaseRoomDo, error)

	Delete(xl *xlog.Logger, roomId string) error

	Update(xl *xlog.Logger, baseRoomDo *model.BaseRoomDo) error

	Select(xl *xlog.Logger, roomId string) (*model.BaseRoomDo, error)

	SelectByInvitationCode(xl *xlog.Logger, invitationCode string) (*model.BaseRoomDo, error)

	ListByRoomType(xl *xlog.Logger, roomType string, pageNum, pageSize int) ([]model.BaseRoomDo, int, int, error)

	ListByTimeout(xl *xlog.Logger, threshold time.Time) ([]model.BaseRoomDo, error)

	// ListAllForce 测试用
	ListAllForce(xl *xlog.Logger) ([]model.BaseRoomDo, error)
}

// BaseRoomDaoService 键在这里生成，不需要传参指定
type BaseRoomDaoService struct {
	client       *mgo.Session
	baseRoomColl *mgo.Collection
	xl           *xlog.Logger
}

func NewBaseRoomDaoService(xl *xlog.Logger, conf *utils.MongoConfig) (*BaseRoomDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-base-room")
	}
	client, err := mgo.Dial(conf.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	baseRoomColl := client.DB(conf.Database).C(dao.CollectionBaseRoom)
	return &BaseRoomDaoService{
		client,
		baseRoomColl,
		xl,
	}, nil
}

func (b *BaseRoomDaoService) Insert(xl *xlog.Logger, baseRoomDo *model.BaseRoomDo) (*model.BaseRoomDo, error) {
	if xl == nil {
		xl = b.xl
	}
	baseRoomDo.Id = bson.NewObjectId().Hex()
	baseRoomDo.CreatedTime = time.Now()
	baseRoomDo.UpdatedTime = time.Now()
	err := b.baseRoomColl.Insert(baseRoomDo)
	if err != nil {
		xl.Error("insert base_room failed.")
		return nil, err
	}
	return baseRoomDo, nil
}

func (b *BaseRoomDaoService) Delete(xl *xlog.Logger, roomId string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseRoomColl.RemoveId(roomId)
	if err != nil {
		xl.Error("delete base_room failed.")
		return err
	}
	return nil
}

func (b *BaseRoomDaoService) Update(xl *xlog.Logger, baseRoomDo *model.BaseRoomDo) error {
	if xl == nil {
		xl = b.xl
	}
	baseRoomDo.UpdatedTime = time.Now()
	err := b.baseRoomColl.Update(bson.M{"_id": baseRoomDo.Id}, baseRoomDo)
	if err != nil {
		xl.Error("update base_room failed.")
		return err
	}
	return nil
}

func (b *BaseRoomDaoService) Select(xl *xlog.Logger, roomId string) (*model.BaseRoomDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var room model.BaseRoomDo
	err := b.baseRoomColl.Find(bson.M{"_id": roomId, "status": model.BaseRoomCreated}).One(&room)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't find this record:[%s] from base_room.", roomId)
		} else {
			xl.Error("select base_room failed.")
		}
		return nil, err
	}
	return &room, nil
}

func (b *BaseRoomDaoService) SelectByInvitationCode(xl *xlog.Logger, invitationCode string) (*model.BaseRoomDo, error) {
	if xl == nil {
		xl = b.xl
	}
	result := model.BaseRoomDo{}
	err := b.baseRoomColl.Find(bson.M{"status": model.BaseRoomCreated, "invitation_code": invitationCode}).One((&result))
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't find this record:[%s] from base_room.", invitationCode)
		} else {
			xl.Error("select base_room failed.")
		}
		return nil, err
	}
	return &result, nil
}

func (b *BaseRoomDaoService) ListByRoomType(xl *xlog.Logger, roomType string, pageNum, pageSize int) ([]model.BaseRoomDo, int, int, error) {
	if xl == nil {
		xl = b.xl
	}
	var baseRoomDos []model.BaseRoomDo
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	err := b.baseRoomColl.Find(bson.M{"status": model.BaseRoomCreated, "type": roomType}).Sort("-created_time").Skip(skip).Limit(limit).All(&baseRoomDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't list those records:[%s] from base_room.", roomType)
		} else {
			xl.Errorf("list by %s from base_room failed.", roomType)
		}
		return nil, 0, 0, err
	}
	total, _ := b.baseRoomColl.Find(bson.M{"status": model.BaseRoomCreated, "type": roomType}).Count()
	return baseRoomDos, total, len(baseRoomDos), nil
}

func (b *BaseRoomDaoService) ListByTimeout(xl *xlog.Logger, threshold time.Time) ([]model.BaseRoomDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var rooms []model.BaseRoomDo
	err := b.baseRoomColl.Find(bson.M{"status": model.BaseRoomCreated, "updated_time": bson.M{"$lt": threshold}}).All(&rooms)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't list those records:[%s] from base_room", threshold)
		} else {
			xl.Error("list by updated_time from base_room failed.")
		}
		return nil, err
	}
	return rooms, nil
}

func (b *BaseRoomDaoService) ListAllForce(xl *xlog.Logger) ([]model.BaseRoomDo, error) {
	if xl == nil {
		xl = b.xl
	}
	result := make([]model.BaseRoomDo, 0, 1)
	_ = b.baseRoomColl.Find(nil).All(&result)
	return result, nil
}
