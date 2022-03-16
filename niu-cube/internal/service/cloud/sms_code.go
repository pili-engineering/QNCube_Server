package cloud

import (
	"encoding/json"
	"fmt"
	"github.com/solutions/niu-cube/internal/common/utils"
	errors2 "github.com/solutions/niu-cube/internal/protodef/errors"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	dao "github.com/solutions/niu-cube/internal/service/db/dao"
	"math/rand"
	"time"

	qiniuauth "github.com/qiniu/go-sdk/v7/auth"
	qiniusms "github.com/qiniu/go-sdk/v7/sms"
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// SMSCodeParamKey 验证码的模板变量名称。
const SMSCodeParamKey = "code"

var (
	// SMSCodeDefaultResendTimeout 重发短信验证码的过期时间。在该时间内已经发送过验证码的手机号不能重发。
	SMSCodeDefaultResendTimeout = time.Minute
	// SMSCodeDefaultValidateTimeout 短信验证码的有效时间。在短信验证码发出后该时间内，验证码有效，过期失效。
	SMSCodeDefaultValidateTimeout = 5 * time.Minute
	// SMSCodeExpireTimeout 短信验证码过期从数据库删除的时间。
	SMSCodeExpireTimeout = 10 * time.Minute
)

type SmsSender interface {
	SendSmsCode(xl *xlog.Logger, phone string, code string) error
}

type SmsCodeService struct {
	mongoClient     *mgo.Session
	smsCodeColl     *mgo.Collection
	smsSender       SmsSender
	resendTimeout   time.Duration
	validateTimeout time.Duration
	expireTimeout   time.Duration
	randSource      rand.Source
	// fixedCodes 固定的手机号与验证码组合，供测试用。
	fixedCodes map[string]string
	xl         *xlog.Logger
}

func NewSmsCodeService(mongoURI string, database string, config *utils.Config, xl *xlog.Logger) (*SmsCodeService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-sms-code-controller")
	}
	mongoClient, err := mgo.Dial(mongoURI + "/" + mongoURI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error %v", err)
		return nil, err
	}
	smsCodeColl := mongoClient.DB(database).C(dao.CollectionSMSCode)
	c := &SmsCodeService{
		mongoClient:     mongoClient,
		smsCodeColl:     smsCodeColl,
		resendTimeout:   SMSCodeDefaultResendTimeout,
		validateTimeout: SMSCodeDefaultValidateTimeout,
		expireTimeout:   SMSCodeExpireTimeout,
		randSource:      rand.NewSource(time.Now().UnixNano()),
		fixedCodes:      config.SMS.FixedCodes,
		xl:              xl,
	}
	// 创建短信发送器。
	switch config.SMS.Provider {
	// 模拟的短信发送器，仅供测试使用。
	case "test":
		c.smsSender = &mockSmsSender{}
	case "qiniu":
		sender := NewQiniuSmsSender(config)
		c.smsSender = sender
	default:
		xl.Errorf("unsupported SMS provider %s", config.SMS.Provider)
		return nil, fmt.Errorf("unsupported SMS provider")
	}
	return c, nil
}

type mockSmsSender struct {
}

func (m *mockSmsSender) SendSmsCode(xl *xlog.Logger, phone string, code string) error {
	xl.Debugf("mock: send code %s to %s", code, phone)
	return nil
}

// QiniuSmsSender 七牛云短信发送器，对接七牛云短信平台发送验证码。
type QiniuSmsSender struct {
	conf    *utils.Config
	manager *qiniusms.Manager
}

// NewQiniuSmsSender 创建七牛云短信发送器。
func NewQiniuSmsSender(conf *utils.Config) *QiniuSmsSender {
	manager := qiniusms.NewManager(&qiniuauth.Credentials{
		AccessKey: conf.QiniuKeyPair.AccessKey,
		SecretKey: []byte(conf.QiniuKeyPair.SecretKey),
	})
	return &QiniuSmsSender{
		conf:    conf,
		manager: manager,
	}
}

// SendMessage 发送验证码为code的短信。
func (s *QiniuSmsSender) SendSmsCode(xl *xlog.Logger, phone string, code string) error {
	_, err := s.manager.SendMessage(qiniusms.MessagesRequest{
		SignatureID: s.conf.SMS.QiniuSMS.SignatureID,
		TemplateID:  s.conf.SMS.QiniuSMS.TemplateID,
		Mobiles:     []string{phone},
		Parameters:  map[string]interface{}{SMSCodeParamKey: code},
	})
	if err != nil {
		xl.Errorf("failed to send message, error %v", err)
		return err
	}
	return nil
}

// Send 对给定手机号发送验证码。
func (c *SmsCodeService) Send(xl *xlog.Logger, phone string) error {
	if xl == nil {
		xl = c.xl
	}
	// TODO: 校验手机号码。
	// 首先查找是否有1分钟内发送给该手机号的记录。
	now := time.Now()
	filter := map[string]interface{}{
		"phone": phone,
		"sendTime": map[string]interface{}{
			"$gt": now.Add(-c.resendTimeout),
		},
	}
	sendCount, err := c.smsCodeColl.Find(filter).Count()
	if err != nil && err != mgo.ErrNotFound {
		xl.Errorf("failed to find sms code record in mongo, error %v", err)
		return err
	}
	if err == nil && sendCount > 0 {
		xl.Infof("phone number %s has already been sent to in 1 minute", phone)
		return &errors2.ServerError{Code: errors2.ServerErrorSMSSendTooFrequent, Summary: ""}
	}

	code := fmt.Sprintf("%06d", c.randSource.Int63()%1000000)

	// TODO: use transaction to save sms code record(need mongo >= 4.0).
	smsCodeID := bson.NewObjectId().String()
	smsCodeRecord := &model.SMSCodeDo{
		ID:       smsCodeID,
		Phone:    phone,
		SMSCode:  code,
		SendTime: time.Now(),
		ExpireAt: time.Now().Add(c.expireTimeout),
	}
	err = c.smsCodeColl.Insert(smsCodeRecord)
	if err != nil {
		xl.Errorf("failed to insert SMS code record, error %v", err)
		return err
	}
	err = c.smsSender.SendSmsCode(xl, phone, code)
	if err != nil {
		xl.Errorf("failed to send SMS code, error %v", err)
		// 删除已插入的发送记录。
		xl.Errorf("===========> %s", smsCodeID)
		deleteErr := c.smsCodeColl.Remove(map[string]interface{}{"_id": smsCodeID})
		if deleteErr != nil {
			xl.Errorf("failed to delete sms code record in mongo, error %v", deleteErr)
		}
		return err
	}
	xl.Debugf("sent code %s to phone number %s", code, phone)
	return nil
}

// Validate 检验手机号与验证码是否符合。
func (c *SmsCodeService) Validate(xl *xlog.Logger, phone string, code string) error {
	if xl == nil {
		xl = c.xl
	}
	// 处理固定验证码组合。
	if c.fixedCodes != nil {
		fixedCode, ok := c.fixedCodes[phone]
		if ok && code == fixedCode {
			xl.Infof("SmsCodeService Validate By fixedCode, phone: %s", phone)
			return nil
		}
	}
	now := time.Now()
	filter := map[string]interface{}{
		"phone":   phone,
		"smsCode": code,
		"sendTime": map[string]interface{}{
			"$gt": now.Add(-c.validateTimeout),
		},
	}
	smsCodeRecord := model.SMSCodeDo{}
	b, jsonerr := json.Marshal(filter)
	if jsonerr != nil {
		fmt.Println("error:", jsonerr)
	}
	xl.Infof("filter > %s", string(b))

	err := c.smsCodeColl.Find(filter).One(&smsCodeRecord)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("sms code is not found or expired")
		} else {
			xl.Errorf("failed to find sms code record, error %v", err)
		}
		return err
	}
	return nil
}
