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

// BaseMicDaoInterface 麦位相关数据库方法
type BaseMicDaoInterface interface {
	InsertBaseMic(xl *xlog.Logger, baseMic *model.BaseMicDo) (*model.BaseMicDo, error)

	Delete(xl *xlog.Logger, micId string) error

	Update(xl *xlog.Logger, baseMic *model.BaseMicDo) error

	Select(xl *xlog.Logger, micId string) (*model.BaseMicDo, error)
}

// BaseMicDaoService 键在这里生成，不需要传参制定
type BaseMicDaoService struct {
	client      *mgo.Session
	baseMicColl *mgo.Collection
	xl          *xlog.Logger
}

func NewBaseMicDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*BaseMicDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-base-mic")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	baseMicColl := client.DB(config.Database).C(dao.CollectionBaseMic)
	return &BaseMicDaoService{
		client,
		baseMicColl,
		xl,
	}, nil
}

func (b *BaseMicDaoService) InsertBaseMic(xl *xlog.Logger, baseMic *model.BaseMicDo) (*model.BaseMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	baseMic.CreatedTime = time.Now()
	baseMic.UpdatedTime = time.Now()
	baseMic.Id = bson.NewObjectId().Hex()
	err := b.baseMicColl.Insert(baseMic)
	if err != nil {
		xl.Error("insert into base_mic failed.")
		return nil, err
	}
	return baseMic, nil
}

func (b *BaseMicDaoService) Delete(xl *xlog.Logger, micId string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseMicColl.RemoveId(micId)
	if err != nil {
		xl.Error("delete from base_mic failed.")
		return err
	}
	return nil
}

func (b *BaseMicDaoService) Update(xl *xlog.Logger, baseMic *model.BaseMicDo) error {
	if xl == nil {
		xl = b.xl
	}
	baseMic.UpdatedTime = time.Now()
	err := b.baseMicColl.UpdateId(baseMic.Id, baseMic)
	if err != nil {
		xl.Error("update base_mic failed.")
		return err
	}
	return nil
}

func (b *BaseMicDaoService) Select(xl *xlog.Logger, micId string) (*model.BaseMicDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var mic model.BaseMicDo
	err := b.baseMicColl.FindId(micId).One(&mic)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't find this records:[%s] from base_mic.", micId)
		} else {
			xl.Error("select from base_mic failed.")
		}
		return nil, err
	}
	return &mic, nil
}
