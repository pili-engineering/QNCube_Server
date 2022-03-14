package db

import (
	"fmt"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/cloud/maxim"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"gopkg.in/mgo.v2"
	"math/rand"
	"sync"
	"time"

	"github.com/qiniu/x/xlog"
)

const (
	// DefaultPingPeriod 轮询用户是否在线的时间间隔。
	DefaultPingPeriod = 5 * time.Second
	// DefaultInactiveTimeout 判定用户掉线的超时时间。
	DefaultInactiveTimeout = 20 * time.Second
)

// QiniuIMService 七牛IM控制器，执行IM用户及聊天室管理。
type QiniuIMService struct {
	appEnvPrefix string
	appKey       string
	appSecret    string
	// pingPeriod 发送ping消息并清理非活跃用户的时间。
	pingPeriod time.Duration
	// inactiveTimeout 清理不活跃用户的超时时间，该段时间内为发送过消息的用户将被清理。
	inactiveTimeout    time.Duration
	userLock           sync.RWMutex
	userMap            map[string]*model.IMUserDo
	qiniuIMClient      *maxim.MaximClient
	xl                 *xlog.Logger
	stopCh             chan struct{}
	qiniuIMUserService QiniuIMUserInterface
}

type AppConfigInterface interface {
	GetUserToken(xl *xlog.Logger, userID string) (*model.IMUserDo, error)
	GetGroupId(xl *xlog.Logger, groupId string) (int64, error)
	DestroyGroupChat(xl *xlog.Logger, groupId int64) error
}

func NewAppConfigService(conf *utils.IMConfig, xl *xlog.Logger) (AppConfigInterface, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-im-config")
	}
	switch conf.Provider {
	case "qiniu":
		if conf.Qiniu == nil {
			return nil, fmt.Errorf("empty config for qiniu IM")
		}
		return NewQiniuIMService(conf, xl)
	case "test":
		return &mockIMService{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %s", conf.Provider)
	}
}

// NewQiniuIMService 创建新的七牛IM控制器。
func NewQiniuIMService(conf *utils.IMConfig, xl *xlog.Logger) (*QiniuIMService, error) {
	if xl == nil {
		xl = xlog.New("qlive-qiniu-im-controller")
	}

	appEnvPrefix := conf.Qiniu.AppEnvPrefix
	appKey := conf.Qiniu.AppId
	appSecret := conf.Qiniu.AppToken

	qiniuIMUserService, err := NewQiniuIMUserService(conf.Qiniu.Mongo, xl)
	if err != nil {
		xl.Errorf("failed to get user token from qiniu im, error %v", err)
		return nil, err
	}

	c := &QiniuIMService{
		appEnvPrefix:       appEnvPrefix,
		appKey:             appKey,
		appSecret:          appSecret,
		userMap:            map[string]*model.IMUserDo{},
		qiniuIMClient:      maxim.NewMaximClient(conf.Qiniu),
		xl:                 xl,
		stopCh:             make(chan struct{}),
		qiniuIMUserService: qiniuIMUserService,
	}

	if conf.PingTickerSecond == 0 {
		c.pingPeriod = DefaultPingPeriod
	} else {
		c.pingPeriod = time.Duration(conf.PingTickerSecond) * time.Second
	}

	if conf.PongTimeoutSecond == 0 {
		c.inactiveTimeout = DefaultInactiveTimeout
	} else {
		c.inactiveTimeout = time.Duration(conf.PongTimeoutSecond) * time.Second
	}

	return c, nil
}

// GetUserToken 用户注册，生成User token
func (c *QiniuIMService) GetUserToken(xl *xlog.Logger, userID string) (*model.IMUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	qiniuUserId := c.appEnvPrefix + userID

	qiniuIMuser, err := c.qiniuIMUserService.GetAccountByID(xl, qiniuUserId)
	if err != nil && err != mgo.ErrNotFound {
		xl.Errorf("GetUserToken.GetAccountByID error %+v", err)
		return nil, err
	}

	if qiniuIMuser != nil {
		return qiniuIMuser, nil
	}

	qiniuIMuser = NewQiniuIMUser(qiniuUserId)
	result, err := c.qiniuIMClient.RegisterUser(xl, qiniuIMuser.Username, qiniuIMuser.GetPassword())
	if err != nil {
		xl.Errorf("maximClient.RegisterUser %s err:%v result:%v", qiniuIMuser.Username, err, result)
		return nil, err
	}
	qiniuIMuser.UserID = result.Get("data.user_id").String()

	if err = c.qiniuIMUserService.CreateAccount(xl, qiniuIMuser); err != nil {
		return nil, err
	}

	return qiniuIMuser, nil
}

func (c *QiniuIMService) GetGroupId(xl *xlog.Logger, groupId string) (int64, error) {
	currentGroupId := c.appEnvPrefix + groupId
	imGroupId, err := c.qiniuIMClient.CreateChatroom(xl, currentGroupId)
	if err != nil {
		xl.Errorf("CreateChatroom %s, error %+v", groupId, err)
		return 0, err
	}
	return imGroupId, nil
}

func (c *QiniuIMService) DestroyGroupChat(xl *xlog.Logger, groupId int64) error {
	return c.qiniuIMClient.DestroyGroupChat(xl, groupId)
}

// RandomSixDigitId generate 6 digit letters
const letters = "qwertyuiopasdfghjklzxcvbnm1234567890"

func RandomSixDigitId() string {
	res := make([]byte, 6)
	for cnt := 0; cnt < 6; cnt++ {
		res[cnt] = letters[rand.Intn(36)]
	}
	return fmt.Sprintf("%s", res)
}

func NewQiniuIMUser(userId string) *model.IMUserDo {
	return &model.IMUserDo{
		Username: userId,
		Password: RandomSixDigitId(),
		Salt:     RandomSixDigitId(),
	}
}

type mockIMService struct{}

func (m *mockIMService) DestroyGroupChat(xl *xlog.Logger, groupId int64) error {
	return nil
}

func (m *mockIMService) GetUserToken(xl *xlog.Logger, userID string) (*model.IMUserDo, error) {
	return &model.IMUserDo{
		UserID:   userID,
		Username: userID,
		Token:    "im-token." + userID,
	}, nil
}

func (m *mockIMService) GetGroupId(xl *xlog.Logger, groupId string) (int64, error) {
	return 0, nil
}
