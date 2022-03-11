package dao

import (
	"time"

	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db"
	"github.com/solutions/niu-cube/internal/service/db/dao"
)

// BaseUserDaoInterface 通用用户相关数据库操作
type BaseUserDaoInterface interface {
	Insert(xl *xlog.Logger, baseUserDo *model.BaseUserDo) (*model.BaseUserDo, error)

	Delete(xl *xlog.Logger, userId string) error

	Update(xl *xlog.Logger, baseUserDo *model.BaseUserDo) error

	Select(xl *xlog.Logger, userId string) (*model.BaseUserDo, error)

	ListAll() ([]model.BaseUserDo, error)
}

// BaseUserDaoService 主键在这里生成，不需要传参制定
type BaseUserDaoService struct {
	client         *mgo.Session
	baseUserColl   *mgo.Collection
	accountService *db.AccountService
	xl             *xlog.Logger
}

func NewBaseUserDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*BaseUserDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-base-user")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	baseUserColl := client.DB(config.Database).C(dao.CollectionBaseUser)
	accountService, err := db.NewAccountService(*config, xl)
	return &BaseUserDaoService{
		client,
		baseUserColl,
		accountService,
		xl,
	}, nil
}

func (b *BaseUserDaoService) Insert(xl *xlog.Logger, baseUserDo *model.BaseUserDo) (*model.BaseUserDo, error) {
	if xl == nil {
		xl = b.xl
	}
	if baseUserDo.Id == "" {
		baseUserDo.Id = bson.NewObjectId().Hex()
	}
	baseUserDo.CreatedTime = time.Now()
	baseUserDo.UpdatedTime = time.Now()
	err := b.baseUserColl.Insert(baseUserDo)
	if err != nil {
		xl.Error("insert into base_user failed.")
		return nil, err
	}
	return baseUserDo, nil
}

func (b *BaseUserDaoService) Delete(xl *xlog.Logger, userId string) error {
	if xl == nil {
		xl = b.xl
	}
	err := b.baseUserColl.RemoveId(userId)
	if err != nil {
		xl.Error("delete from base_user failed.")
		return err
	}
	return nil
}

func (b *BaseUserDaoService) Update(xl *xlog.Logger, baseUserDo *model.BaseUserDo) error {
	if xl == nil {
		xl = b.xl
	}
	baseUserDo.UpdatedTime = time.Now()
	err := b.baseUserColl.UpdateId(baseUserDo.Id, baseUserDo)
	if err != nil {
		xl.Error("update base_user failed.")
		return err
	}
	return nil
}

func (b *BaseUserDaoService) Select(xl *xlog.Logger, userId string) (*model.BaseUserDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var baseUserDo model.BaseUserDo
	err := b.baseUserColl.FindId(userId).One(&baseUserDo)
	if err != nil {
		// 查询旧表
		if err == mgo.ErrNotFound {
			val, err := b.accountService.GetAccountByID(xl, userId)
			if err != nil {
				xl.Error("select from old user collection failed.")
				return nil, err
			}
			baseUserDo = model.BaseUserDo{
				Id:            userId,
				Name:          val.Nickname,
				Nickname:      val.Nickname,
				Avatar:        val.Avatar,
				Status:        model.BaseUserLogin,
				Profile:       "",
				CreatedTime:   val.RegisterTime,
				UpdatedTime:   time.Now(),
				BaseUserAttrs: nil,
			}
			err = b.baseUserColl.Insert(&baseUserDo)
			if err != nil {
				xl.Error("insert into base_user failed.")
				return nil, err
			}
			return &baseUserDo, nil
		} else {
			xl.Error("select from base_user failed.")
			return nil, err
		}
	}
	return &baseUserDo, nil
}

func (b *BaseUserDaoService) ListAll() ([]model.BaseUserDo, error) {
	results := make([]model.BaseUserDo, 0)
	err := b.baseUserColl.Find(nil).All(&results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
