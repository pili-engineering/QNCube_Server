package db

import (
	"fmt"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type AccountService interface {
	// action
	LoginBySMSCode(id string, code string) (bool, error)
	LoginByPassword(email string, passwd string) (*model.Account, error)
	SignUp(account model.Account) (string, error)

	//LoginByOauth(id string,code string)(bool,error)
	UpdatePassword(id string, old, new string) error
	UpdateNickName(id string, new string) error

	// crud
	GetAccountByEmail(email string) (*model.Account, error)
	GetAccountByPhone(phone string) (*model.Account, error)
	GetIdByEmail(email string) (string, error)
	GetIdByPhone(phone string) (string, error)

	GetOneByMap(filter bson.M) (*model.Account, error)
	GetByMap(filter bson.M) ([]*model.Account, error)
}

type AccountServiceImpl struct {
	accountColl *mgo.Collection
}

func NewAccountService() AccountService {
	client, err := mgo.Dial(common.GetConf().Mongo.URI)
	if err != nil {
		panic(err)
	}
	accountColl := client.DB(common.GetConf().Mongo.Database).C(model.CollectionAccount)
	return &AccountServiceImpl{accountColl: accountColl}
}

// SignUp 基本是一些唯一性校验 数据库设置unique index 或者自己校验
// email phone 唯一
// 根据Form中的type进行区分，目前实现email + password就行
func (a AccountServiceImpl) SignUp(account model.Account) (string, error) {
	_, err := a.GetIdByEmail(account.Email)
	if err != mgo.ErrNotFound {
		return "", fmt.Errorf("邮箱已存在") // validation err
	}
	if account.ID == "" {
		account.ID = bson.NewObjectId().Hex()
	}
	return account.ID, a.accountColl.Insert(account) //log here
}

func (a AccountServiceImpl) GetOneByMap(filter bson.M) (*model.Account, error) {
	var ac model.Account
	err := a.accountColl.Find(filter).One(&ac)
	return &ac, err
}

func (a AccountServiceImpl) GetByMap(filter bson.M) ([]*model.Account, error) {
	acs := make([]*model.Account, 0)
	err := a.accountColl.Find(filter).One(&acs)
	return acs, err
}

func (a AccountServiceImpl) LoginBySMSCode(phone string, code string) (bool, error) {
	panic("implement me")
}

func (a AccountServiceImpl) LoginByPassword(email string, passwd string) (account *model.Account, err error) {
	filter := bson.M{"email": email, "password": passwd}
	user, err := a.GetOneByMap(filter) // more update here
	switch {
	case err != nil:
		return nil, err
	default:
		return user, nil
	}
}

func (a AccountServiceImpl) UpdatePassword(id string, old, new string) error {
	panic("implement me")
}

func (a AccountServiceImpl) UpdateNickName(id string, new string) error {
	panic("implement me")
}

func (a AccountServiceImpl) GetAccountByEmail(email string) (*model.Account, error) {
	filter := bson.M{"email": email}
	var account *model.Account
	err := a.accountColl.Find(filter).One(account)
	switch {
	case err == mgo.ErrNotFound:
		return account, err
	case err != nil && err != mgo.ErrNotFound:
		// log handle
		return account, err
	default:
		return account, err
	}
}

func (a AccountServiceImpl) GetAccountByPhone(phone string) (*model.Account, error) {
	filter := bson.M{"phone": phone}
	return a.GetOneByMap(filter)
}

func (a AccountServiceImpl) GetIdByEmail(email string) (string, error) {
	filter := bson.M{"email": email}
	var account model.Account
	err := a.accountColl.Find(filter).One(&account)
	switch {
	case err == mgo.ErrNotFound:
		return "", err
	case err != nil && err != mgo.ErrNotFound:
		// log handle
		return "", err
	default:
		return account.ID, err
	}
}

func (a AccountServiceImpl) GetIdByPhone(phone string) (id string, err error) {
	ac, err := a.GetAccountByPhone(phone)
	if err == nil {
		return ac.ID, err
	} else {
		return "", err
	}
}
