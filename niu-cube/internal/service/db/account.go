package db

import (
	"github.com/solutions/niu-cube/internal/common/utils"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	dao "github.com/solutions/niu-cube/internal/service/db/dao"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// AccountController 用户注册、更新信息、登录、退出登录等操作。
type AccountService struct {
	mongoClient      *mgo.Session
	accountColl      *mgo.Collection
	accountTokenColl *mgo.Collection
	xl               *xlog.Logger
}

func NewAccountService(conf utils.MongoConfig, xl *xlog.Logger) (*AccountService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-account-db")
	}
	mongoClient, err := mgo.Dial(conf.URI + "/" + conf.Database)
	if err != nil {
		xl.Errorf("failed to create mongo client, error %v", err)
		return nil, err
	}
	accountColl := mongoClient.DB(conf.Database).C(dao.CollectionAccount)
	accountTokenColl := mongoClient.DB(conf.Database).C(dao.CollectionAccountToken)
	return &AccountService{
		mongoClient:      mongoClient,
		accountColl:      accountColl,
		accountTokenColl: accountTokenColl,
		xl:               xl,
	}, nil
}

// CreateAccount 创建用户账号。
func (c *AccountService) CreateAccount(xl *xlog.Logger, account *model.AccountDo) error {
	if xl == nil {
		xl = c.xl
	}
	account.RegisterTime = time.Now()
	err := c.accountColl.Insert(account)
	if err != nil {
		xl.Errorf("failed to insert user, error %v", err)
		return err
	}
	return nil
}

// GetAccountByPhone 使用电话号码查找账号。
func (c *AccountService) GetAccountByPhone(xl *xlog.Logger, phone string) (*model.AccountDo, error) {
	return c.GetAccountByFields(xl, map[string]interface{}{"phone": phone})
}

// GetOrSaveAccountByPhone 使用电话号码查找账号。
func (c *AccountService) GetOrSaveAccountByPhone(xl *xlog.Logger, phone string) (*model.AccountDo, error) {
	return c.GetAccountByFields(xl, map[string]interface{}{"phone": phone})
}

// GetAccountByID 使用ID查找账号。
func (c *AccountService) GetAccountByID(xl *xlog.Logger, id string) (*model.AccountDo, error) {
	return c.GetAccountByFields(xl, map[string]interface{}{"_id": id})
}

// GetAccountByFields 根据一组key/value关系查找用户账号。
func (c *AccountService) GetAccountByFields(xl *xlog.Logger, fields map[string]interface{}) (*model.AccountDo, error) {
	if xl == nil {
		xl = c.xl
	}
	account := model.AccountDo{}
	err := c.accountColl.Find(fields).One(&account)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("no such user for fields %v", fields)
			return nil, mgo.ErrNotFound
		}
		xl.Errorf("failed to get user, error %v", fields)
		return nil, err
	}
	return &account, nil
}

// UpdateAccount 更新用户信息。
func (c *AccountService) UpdateAccount(xl *xlog.Logger, id string, newAccount *model.AccountDo) (*model.AccountDo, error) {
	if xl == nil {
		xl = c.xl
	}
	account, err := c.GetAccountByID(xl, id)
	if err != nil {
		return nil, err
	}
	if newAccount.Nickname != "" {
		account.Nickname = newAccount.Nickname
	}
	if newAccount.Avatar != "" {
		account.Avatar = newAccount.Avatar
	}
	err = c.accountColl.Update(bson.M{"_id": id}, bson.M{"$set": account})
	if err != nil {
		xl.Errorf("failed to update account %s,error %v", id, err)
		return nil, err
	}
	return account, nil
}

// AccountLogin 设置某个账号为已登录状态。
func (c *AccountService) AccountLogin(xl *xlog.Logger, userID string) (user *model.AccountTokenDo, err error) {
	if xl == nil {
		xl = c.xl
	}
	account, err := c.GetAccountByID(xl, userID)
	if err != nil {
		xl.Errorf("AccountLogin: failed to find account %s", userID)
		return nil, err
	}
	// 查看是否已经登录。
	activeUser := &model.AccountTokenDo{
		ID:        userID,
		AccountId: userID,
	}
	err = c.accountTokenColl.Find(map[string]interface{}{"_id": userID}).
		One(activeUser)
	if err != nil {
		if err != mgo.ErrNotFound {
			xl.Errorf("failed to check logged in users in mongo,error %v", err)
			return nil, err
		}
	} else {
		xl.Infof("user %s has been already logged in, the old session will be invalid", userID)
	}
	// generate token.
	activeUser.Token = c.makeLoginToken(xl, account)
	// update or insert login record.
	_, err = c.accountTokenColl.Upsert(bson.M{"_id": userID}, activeUser)
	if err != nil {
		xl.Errorf("failed to update or insert user login record, error %v", err)
		return nil, err
	}
	// 更新最后登录时间。
	account.LastLoginTime = time.Now()
	err = c.accountColl.Update(bson.M{"_id": userID}, bson.M{"$set": bson.M{"lastLoginTime": time.Now()}})
	if err != nil {
		// 更新登录时间失败不影响正常返回。
		xl.Errorf("failed to update user %s login time, error %v", userID, err)
	}
	return activeUser, nil
}

func (c *AccountService) makeLoginToken(xl *xlog.Logger, account *model.AccountDo) string {
	if xl == nil {
		xl = c.xl
	}
	timestamp := time.Now().UnixNano()
	// TODO: add more secret things for token?
	claims := jwt.MapClaims{
		"userID":    account.ID,
		"timestamp": timestamp,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := t.SignedString([]byte(""))
	return token
}

// AccountLogout 用户退出登录。
func (c *AccountService) AccountLogout(xl *xlog.Logger, userID string) error {
	if xl == nil {
		xl = c.xl
	}
	// 删除用户登录记录。
	err := c.accountTokenColl.RemoveId(userID)
	if err != nil {
		xl.Errorf("failed to remove user ID %s in logged in users, error %v", userID, err)
		return err
	}
	return nil
}

// GetIDByToken 根据token获取账号ID。如果未在已登录用户表查找到这个token，说明该token不合法。
func (c *AccountService) GetIDByToken(xl *xlog.Logger, token string) (id string, err error) {
	if xl == nil {
		xl = c.xl
	}
	accountTokenRecord := &model.AccountTokenDo{}
	err = c.accountTokenColl.Find(map[string]interface{}{"token": token}).One(accountTokenRecord)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("token %s not found in active users", token)
			return "", err
		}
		xl.Errorf("failed to find token in active users, error %v", err)
		return "", err
	}
	return accountTokenRecord.ID, nil
}

func (c *AccountService) DeleteAccount(xl *xlog.Logger, id string) error {
	if xl == nil {
		xl = c.xl
	}
	return c.accountColl.RemoveId(id)
}

func (c *AccountService) ListAll0() ([]model.AccountDo, error) {
	results := make([]model.AccountDo, 0)
	err := c.accountColl.Find(nil).All(&results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
