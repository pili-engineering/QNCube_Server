package common

import (
	qconfig "github.com/qiniu/x/config"
	"github.com/qiniu/x/xlog"
	"sync"
)

var (
	defaultConf   Config
	once          sync.Once
	defaultLogger = xlog.New("common")
)

const (
	defaultConfigFilePath = "/Users/hades300/Zone/Repo/solutions/Server/niu-cube/cmd/niu-cube-interview/niu-cube-interview.dev.conf"
)

func GetConf() Config {
	once.Do(func() {
		err := qconfig.LoadFile(&defaultConf, defaultConfigFilePath)
		if err != nil {
			defaultLogger.Fatalf("failed to load config file, error %v", err)
		}
	})
	return defaultConf
}

// SignalingConfig 控制信令相关的配置。
type SignalingConfig struct {
	// Type 信令通道的类型，设为websocket/ws表示通过websocket收发信令，im表示通过im收发信令。
	Type string `json:"type" validate:"nonzero"`
	// PKRequestTimeoutSecond PK请求超时时间。
	PKRequestTimeoutSecond int `json:"pk_request_timeout_s"`
	// JoinRequestTimeoutSecond 连麦请求超时时间。
	JoinRequestTimeoutSecond int `json:"join_request_timeout_s"`
}

// MongoConfig mongo 数据库配置。
type MongoConfig struct {
	URI      string `json:"uri"`
	Database string `json:"database"`
}

// QiniuKeyPair 七牛APIaccess key/secret key配置。
type QiniuKeyPair struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// QiniuSMSConfig 七牛云短信配置。
type QiniuSMSConfig struct {
	KeyPair     QiniuKeyPair `json:"key_pair"`
	SignatureID string       `json:"signature_id"`
	TemplateID  string       `json:"template_id"`
}

// MailConfig 发送邮件的配置。
type MailConfig struct {
	Enabled             bool     `json:"enabled"`
	SMTPHost            string   `json:"smtp_host"`
	SMTPPort            int      `json:"smtp_port"`
	From                string   `json:"from"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	To                  []string `json:"to"`
	RetryTimes          int      `json:"retry_times"`
	RetryIntervalSecond int      `json:"retry_interval_s"`
}

// SMSConfig 短信服务配置。
type SMSConfig struct {
	Provider string `json:"provider"`
	// FixedCodes 固定的手机号->验证码组合，供测试用。
	FixedCodes map[string]string `json:"fixed_codes,omitempty"`
	QiniuSMS   *QiniuSMSConfig   `json:"qiniu_sms"`
}

// QiniuRTCConfig 七牛RTC服务配置。
// PlayBackHost  点播地址
// Hub 直播空间名字
// StreamPattern 流命名模式
type QiniuRTCConfig struct {
	KeyPair QiniuKeyPair `json:"key_pair"`
	AppID   string       `json:"app_id"`
	// RTC room token的有效时间。
	RoomTokenExpireSecond int    `json:"room_token_expire_s"`
	PlayBackURL           string `json:"play_back_url"`
	Hub                   string `json:"hub"`
	StreamPattern         string `json:"stream_pattern"`
	PublishURL            string `json:"publish_url"`
}

// QiniuStorageConfig 七牛对象存储服务配置。
type QiniuStorageConfig struct {
	KeyPair QiniuKeyPair `json:"key_pair"`
	// Bucket 上传的文件所在的七牛对象存储bucket。
	Bucket string `json:"bucket"`
	// URLPrefix 上传的文件的下载URL前缀，一般为该bucket对应的默认域名。
	URLPrefix string `json:"url_prefix"`
}

// RongCloudIMConfig 融云IM服务配置。
type RongCloudIMConfig struct {
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
}

// IMConfig IM服务配置。
type IMConfig struct {
	Provider string `json:"provider"`
	// SystemUserID 系统用户ID。该用户用于与直播用户通过IM传递控制消息。
	SystemUserID      string             `json:"system_user_id"`
	PingTickerSecond  int                `json:"ping_ticker_s"`
	PongTimeoutSecond int                `json:"pong_timeout_s"`
	RongCloud         *RongCloudIMConfig `json:"rongcloud"`
}

type Solution struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	URL   string `json:"url"`
	Desc  string `json:"desc"`
}

// Bucket 保存小程序分享码的七牛存储空间名
// QRFilePattern 保存小程序码的文件名模式 默认 interview-qrcode/%s-%s
// Link cdn host
type Weixin struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Bucket    string `json:"bucket"`
	Link      string `json:"link"`
}

// Config 后端配置。
type Config struct {
	// debug等级，为1时输出info/warn/error日志，为0除以上外还输出debug日志
	DebugLevel int    `json:"debug_level"`
	ListenAddr string `json:"listen_addr"`
	// 默认头像列表，用户新注册时随机从中选取一个作为初始头像。
	DefaultAvatars []string `json:"default_avatars"`
	// 请求默认host
	RequestUrlHost string `json:"request_url_host"`
	// 前端页面host
	FrontendUrlHost string `json:"frontend_url_host"`

	WelcomeImage string         `json:"welcome_image"`
	WelcomeURL   string         `json:"welcome_url"`
	Mongo        MongoConfig    `json:"mongo"`
	SMS          SMSConfig      `json:"sms"`
	RTC          QiniuRTCConfig `json:"rtc"`
	IM           IMConfig       `json:"im"`

	Solutions []Solution `json:"solutions"`
	Weixin    Weixin     `json:"weixin"`

	JwtKey string `json:"jwt_key"`
}
