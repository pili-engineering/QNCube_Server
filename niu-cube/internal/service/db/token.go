package db

import (
	"fmt"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	dao "github.com/solutions/niu-cube/internal/service/db/dao"
	"gopkg.in/mgo.v2"
	"math/rand"
)

type AccountTokenService struct {
	accountTokenCollection *mgo.Collection
	xl                     *xlog.Logger
}

func NewAccountTokenService(xl *xlog.Logger, config utils.MongoConfig) *AccountTokenService {
	v := new(AccountTokenService)
	v.xl = xlog.New("accountToken service")
	db, err := mgo.Dial(config.URI)
	if err != nil {
		v.xl.Fatalf("error dialing service error:%v", err)
	}
	err = db.Ping()
	if err != nil {
		v.xl.Fatalf("err ping db error:%v", err)
	}
	v.accountTokenCollection = db.DB(config.Database).C(dao.CollectionAccountToken)
	return v
}

// CRUD

func (v *AccountTokenService) Create(xl *xlog.Logger, accountToken model.AccountTokenDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if accountToken.ID == "" {
		switch accountToken.AccountId {
		case "":
			logger.Errorf("err create accountTokendo : no account id")
			return fmt.Errorf("err create accountTokendo : no account id")
		default:
			accountToken.ID = accountToken.AccountId
		}
	}
	err := v.accountTokenCollection.Insert(accountToken)
	if err != nil {
		logger.Errorf("error create accountTokenDo %v err:%v", accountToken, err)
	}
	return err
}

func (v *AccountTokenService) Upsert(xl *xlog.Logger, accountToken model.AccountTokenDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if accountToken.ID == "" {
		accountToken.ID = v.generateID()
	}
	err := v.accountTokenCollection.UpdateId(accountToken.ID, accountToken)
	if err != nil {
		logger.Errorf("error upsert accountTokenDo %v err:%v", accountToken, err)
	}
	return err
}

func (v *AccountTokenService) Update(xl *xlog.Logger, accountToken model.AccountTokenDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.accountTokenCollection.UpdateId(accountToken.ID, accountToken)
	if err != nil {
		logger.Errorf("error update accountTokenDo %v err:%v", accountToken, err)
	}
	return err
}

func (v *AccountTokenService) Delete(xl *xlog.Logger, id string) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.accountTokenCollection.RemoveId(id)
	if err != nil {
		logger.Errorf("error update accountTokenId %v err:%v", id, err)
	}
	return err
}

func (c *AccountTokenService) GetOneByID(xl *xlog.Logger, id string) (model.AccountTokenDo, error) {
	return c.GetOneByMap(xl, map[string]interface{}{"_id": id})
}

func (v *AccountTokenService) GetOneByMap(xl *xlog.Logger, filter interface{}) (model.AccountTokenDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	var token model.AccountTokenDo
	err := v.accountTokenCollection.Find(filter).One(&token)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return token, err
	}
	return token, err
}

// GetByMap
func (v *AccountTokenService) GetByMap(xl *xlog.Logger, filter interface{}) ([]model.AccountTokenDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	accountTokens := make([]model.AccountTokenDo, 0)
	var err error
	err = v.accountTokenCollection.Find(filter).All(&accountTokens)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return accountTokens, err
	}
	return accountTokens, err
}

// GetPageByMap
func (v *AccountTokenService) GetPageByMap(xl *xlog.Logger, filter interface{}, pageNum, pageSize int) ([]model.AccountTokenDo, int, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	accountTokens := make([]model.AccountTokenDo, 0)
	var err error
	err = v.accountTokenCollection.Find(filter).Skip((pageNum - 1) * pageSize).Limit(pageSize).All(&accountTokens)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return accountTokens, 0, err
	}
	cnt, err := v.accountTokenCollection.Find(filter).Count()
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return accountTokens, 0, err
	}
	return accountTokens, cnt, err
}

// generateID utils func: for 12-digit random id generation
func (v *AccountTokenService) generateID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	idLength := 12
	id := ""
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		id = id + string(alphaNum[index])
	}
	return id
}

func (v *AccountTokenService) ListActiveUsers(userIds ...string) {

}

func (v *AccountTokenService) Permit(action string, userId string) bool {
	return true
}
