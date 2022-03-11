package db

import (
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
	"gopkg.in/mgo.v2"
)

type QiniuIMUserInterface interface {
	CreateAccount(xl *xlog.Logger, imUserDo *model.IMUserDo) error
	GetAccountByID(xl *xlog.Logger, id string) (*model.IMUserDo, error)
}

// QiniuIMUserService 七牛IM用户
type QiniuIMUserService struct {
	mongoClient     *mgo.Session
	qiniuIMUserColl *mgo.Collection
	xl              *xlog.Logger
}

func NewQiniuIMUserService(conf *utils.MongoConfig, xl *xlog.Logger) (*QiniuIMUserService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-qiniu-im-user-db")
	}
	mongoClient, err := mgo.Dial(conf.URI + "/" + conf.Database)
	if err != nil {
		xl.Errorf("failed to create mongo client, error %v", err)
		return nil, err
	}
	qiniuIMUserColl := mongoClient.DB(conf.Database).C(dao.CollectionQiniuIMUser)
	return &QiniuIMUserService{
		mongoClient:     mongoClient,
		qiniuIMUserColl: qiniuIMUserColl,
		xl:              xl,
	}, nil
}

// CreateAccount 创建用户账号。
func (c *QiniuIMUserService) CreateAccount(xl *xlog.Logger, imUserDo *model.IMUserDo) error {
	if xl == nil {
		xl = c.xl
	}
	err := c.qiniuIMUserColl.Insert(imUserDo)
	if err != nil {
		xl.Errorf("failed to insert qiniu im user, error %v", err)
		return err
	}
	return nil
}

// GetAccountByID 使用ID查找账号。
func (c *QiniuIMUserService) GetAccountByID(xl *xlog.Logger, id string) (*model.IMUserDo, error) {
	return c.GetAccountByFields(xl, map[string]interface{}{"username": id})
}

// GetAccountByFields 根据一组key/value关系查找用户账号。
func (c *QiniuIMUserService) GetAccountByFields(xl *xlog.Logger, fields map[string]interface{}) (*model.IMUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	imUser := model.IMUserDo{}
	err := c.qiniuIMUserColl.Find(fields).One(&imUser)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("no such qiniu im user for fields %v", fields)
			return nil, mgo.ErrNotFound
		}
		xl.Errorf("failed to get qiniu im user, error %v", fields)
		return nil, err
	}
	return &imUser, nil
}
