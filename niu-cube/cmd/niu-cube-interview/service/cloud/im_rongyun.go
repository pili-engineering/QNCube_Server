package cloud

import (
	"github.com/qiniu/x/xlog"
	rcsdk "github.com/rongcloud/server-sdk-go/v3/sdk"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"sync"
	"time"
)

const (
	// DefaultPortraitURL 默认IM头像地址。
	DefaultPortraitURL = "https://developer.rongcloud.cn/static/images/newversion-logo.png"
	// DefaultPingPeriod 轮询用户是否在线的时间间隔。
	DefaultPingPeriod = 5 * time.Second
	// DefaultInactiveTimeout 判定用户掉线的超时时间。
	DefaultInactiveTimeout = 20 * time.Second
)

// RongCloudIMService 融云IM控制器，执行IM用户及聊天室管理。
type RongCloudIMService struct {
	appKey    string
	appSecret string
	// systemUserID 系统用户ID，发送到该ID的IM消息将被当作发送给系统的信令处理。
	systemUserID string
	// pingPeriod 发送ping消息并清理非活跃用户的时间。
	pingPeriod time.Duration
	// inactiveTimeout 清理不活跃用户的超时时间，该段时间内为发送过消息的用户将被清理。
	inactiveTimeout time.Duration
	userLock        sync.RWMutex
	userMap         map[string]*model.IMUser
	rongCloudClient *rcsdk.RongCloud
	xl              *xlog.Logger
	stopCh          chan struct{}
}

type IMService interface {
	GetUserToken(xl *xlog.Logger, userID string) (*model.IMUser, error)
}

// NewRongCloudIMService 创建新的融云IM控制器。
func NewRongCloudIMService() *RongCloudIMService {
	conf := common.GetConf().IM
	appKey := conf.RongCloud.AppKey
	appSecret := conf.RongCloud.AppSecret
	systemUserID := conf.SystemUserID

	c := &RongCloudIMService{
		appKey:          appKey,
		appSecret:       appSecret,
		systemUserID:    systemUserID,
		userMap:         map[string]*model.IMUser{},
		rongCloudClient: rcsdk.NewRongCloud(appKey, appSecret),
		xl:              xlog.New("qlive-rongcloud-im-controller"),
		stopCh:          make(chan struct{}),
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

	_, err := c.GetUserToken(c.xl, systemUserID)
	if err != nil {
		c.xl.Fatalf("failed to get user token for system user %s, error %v", systemUserID, err)
	}
	return c
}

// GetUserToken 用户注册，生成User token
func (c *RongCloudIMService) GetUserToken(xl *xlog.Logger, userID string) (*model.IMUser, error) {
	if xl == nil {
		xl = c.xl
	}
	userRes, err := c.rongCloudClient.UserRegister(userID, userID, DefaultPortraitURL)
	if err != nil {
		xl.Errorf("failed to get user token from rongcloud, error %v", err)
		return nil, err
	}
	now := time.Now()
	IMUser := &model.IMUser{
		UserID:           userRes.UserID,
		Username:         userRes.UserID,
		Token:            userRes.Token,
		LastRegisterTime: now,
		LastOnlineTime:   now,
	}
	c.setIMUser(IMUser)
	return IMUser, nil
}

func (c *RongCloudIMService) setIMUser(user *model.IMUser) {
	if user == nil || user.UserID == "" {
		return
	}

	c.userLock.Lock()
	defer c.userLock.Unlock()
	c.userMap[user.UserID] = user
}

func (c *RongCloudIMService) getIMUser(userID string) (user *model.IMUser, ok bool) {
	c.userLock.RLock()
	defer c.userLock.RUnlock()

	user, ok = c.userMap[userID]
	return user, ok
}
