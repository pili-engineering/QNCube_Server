package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

/*
	model.go: 规定数据存储的格式。
*/

// IMUserDo 对应IM 用户信息。
type IMUserDo struct {
	UserID           string    `json:"id"`
	Username         string    `json:"name"`
	Token            string    `json:"token"`
	LastRegisterTime time.Time `json:"lastRegisterTime"`
	LastOnlineTime   time.Time `json:"lastOnlineTime"`
	LastOfflineTime  time.Time `json:"lastOfflineTime"`
	Password         string    `json:"password"`
	Salt             string    `json:"salt"`
}

var (
	PasswordEncodeSecretKey = []byte("qiniu")
)

// GetPassword Generate User password for maxim
// password = hmac_sha256(password+salt,secret_key)
func (a *IMUserDo) GetPassword() string {
	if a.Salt == "" {
		panic("salt could not be empty")
	}
	payload := a.Password + a.Salt
	macData := hmac.New(sha256.New, PasswordEncodeSecretKey).Sum([]byte(payload))
	return hex.EncodeToString(macData)
}

// AccountDo 用户账号信息。
type AccountDo struct {
	// 用户ID，作为数据库唯一标识。
	ID string `json:"id" bson:"_id"`
	// 手机号，目前要求全局唯一。
	Phone string `json:"phone" bson:"phone"`
	// TODO：支持账号密码登录。
	Password string `json:"password" bson:"password"`
	// 用户昵称
	Nickname string `json:"nickname" bson:"nickname"`
	// Avatar 头像URL地址
	Avatar string `json:"avatar,omitempty" bson:"avatar,omitempty"`
	// RegisterIP 用户注册（首次登录）时使用的IP。
	RegisterIP string `json:"registerIP" bson:"registerIP"`
	// RegisterTime 用户注册（首次登录）时间。
	RegisterTime time.Time `json:"registerTime" bson:"registerTime"`
	// LastLoginTime 上次登录时间。
	LastLoginTime time.Time `json:"lastLoginTime" bson:"lastLoginTime"`
}

func (a AccountDo) Map() FlattenMap {
	val, _ := json.Marshal(&a)
	res := make(map[string]interface{})
	_ = json.Unmarshal(val, &res)
	return res
}

// AccountTokenDo 已登录用户的信息。
type AccountTokenDo struct {
	ID        string `json:"id" bson:"_id"`
	AccountId string `json:"accountId" bson:"accountId"`
	// Token 本次登录使用的token。
	Token          string    `json:"token" bson:"token"`
	LastModifyTime time.Time `json:"lastModifyTime"`
}

// SMSCodeDo 已发送的验证码记录。
type SMSCodeDo struct {
	ID       string    `json:"id" bson:"_id"`
	Phone    string    `json:"phone" bson:"phone"`
	SMSCode  string    `json:"smsCode" bson:"smsCode"`
	SendTime time.Time `json:"sendTime" bson:"sendTime"`
	ExpireAt time.Time `json:"-" bson:"expireAt"`
}

type SolutionDo struct {
	ID      string `json:"id" bson:"_id"`
	Title   string `json:"title" bson:"title"`
	Icon    string `json:"icon" bson:"icon"`
	Os      string `json:"os" bson:"os"`
	Version string `json:"version" bson:"version"`
	Url     string `json:"url" bson:"url"`
	Desc    string `json:"desc" bson:"desc"`
}

type AppDo struct {
	ID           string `json:"id" bson:"_id"`
	Version      string `json:"version" bson:"version"`
	BuildVersion string `json:"buildVersion" bson:"buildVersion"`
	Os           string `json:"os" bson:"os"`
}

type RoomStatus int

const (
	RoomStatusInit  RoomStatus = 0
	RoomStatusStart RoomStatus = 10
	RoomStatusEnd   RoomStatus = -10
)

type RoomDo struct {
	ID      string `json:"id" bson:"_id"`
	Name    string `json:"name" bson:"name"`
	Status  int    `json:"status" bson:"status"`
	BizType string `json:"bizType" bson:"bizType"`
}

type RoomUserDo struct {
	ID        string    `json:"id" bson:"_id"`
	RoomId    string    `json:"roomId" bson:"roomId"`
	AccountId string    `json:"accountId" bson:"accountId"`
	Role      string    `json:"role" bson:"role"`
	Status    string    `json:"status" bson:"status"`
	StartTime time.Time `json:"startTime" bson:"startTime"`
	EndTime   time.Time `json:"endTime" bson:"endTime"`
}

type InterviewStatusCode int
type InterviewStatusName string
type InterviewRoleCode int
type InterviewRoleName string

const (
	InterviewStatusCodeInit      InterviewStatusCode = 0
	InterviewStatusCodeStart     InterviewStatusCode = 10
	InterviewStatusCodeEnd       InterviewStatusCode = -10
	InterviewStatusNameInit      InterviewStatusName = "待面试"
	InterviewStatusNameStart     InterviewStatusName = "面试中"
	InterviewStatusNameEnd       InterviewStatusName = "已结束"
	InterviewRoleCodeInterviewer InterviewRoleCode   = 1
	InterviewRoleCodeCandidate   InterviewRoleCode   = 2
	InterviewRoleNameInterviewer InterviewRoleName   = "面试官"
	InterviewRoleNameCandidate   InterviewRoleName   = "应聘者"
)

type InterviewDo struct {
	ID              string    `json:"id" bson:"_id"`
	RoomId          string    `json:"InterviewRoleCoderoomId" bson:"roomId"`
	Title           string    `json:"title" bson:"title"`
	StartTime       time.Time `json:"startTime" bson:"startTime"`
	EndTime         time.Time `json:"endTime" bson:"endTime"`
	Goverment       string    `json:"goverment" bson:"goverment"`
	Career          string    `json:"career" bson:"career"`
	IsRecord        bool      `json:"isRecord" bson:"isRecord"`
	Recorded        bool      `json:"recorded" bson:"recorded"`
	IsAuth          bool      `json:"isAuth" bson:"isAuth"`
	AuthCode        string    `json:"authCode" bson:"authCode"`
	Status          int       `json:"status" bson:"status"`
	CreateTime      time.Time `json:"createTime" bson:"createTime"`
	UpdateTime      time.Time `json:"updateTime" bson:"updateTime"`
	Creator         string    `json:"creator" bson:"creator"`
	Updator         string    `json:"updator" bson:"updator"`
	Interviewer     string    `json:"interviewer" bson:"interviewer"`
	InterviewerName string    `json:"interviewerName" bson:"interviewerName"`
	Candidate       string    `json:"candidate" bson:"candidate"`
	CandidateName   string    `json:"candidateName" bson:"candidateName"`
	AppletQrcode    string    `json:"applet_qrcode" bson:"applet_qrcode"`
	QiniuIMGroupId  int64     `json:"qiniuIMGroupId" bson:"qiniuIMGroupId"`
}

type InterviewUserDo struct {
	ID                string    `json:"id" bson:"_id"`
	InterviewID       string    `json:"interviewId" bson:"interviewId"`
	UserID            string    `json:"userId" bson:"userId"`
	Status            int       `json:"status" bson:"status"`
	LastModifyTime    time.Time `json:"lastModifyTime" bson:"lastModifyTime"`
	LastHeartBeatTime time.Time `json:"last_heart_beat_time" bson:"lastHeartBeatTime"`
}

type VersionUpgradeType string
type VersionPlatform string
type VersionUpgradeCode int

func (v *VersionPlatform) Validate() error {
	fmt.Println("validating platfomr !!!! ", v)
	return validation.Validate(v, validation.In(VersionPlatformIos, VersionPlatformAndroid))
}

const (
	VersionUpgradeTypeForce  = VersionUpgradeType("force")
	VersionUpgradeTypeNotify = VersionUpgradeType("notify")

	VersionUpgradeCodeForce  = VersionUpgradeCode(1)
	VersionUpgradeCodeNotify = VersionUpgradeCode(2)

	VersionPlatformAndroid = VersionPlatform("android")
	VersionPlatformIos     = VersionPlatform("ios")
)

// VersionDo UpgradeType 为枚举类型 force\notify
type VersionDo struct {
	ID          string             `json:"id" bson:"_id"`
	AppName     string             `json:"app_name" bson:"app_name"`
	Platform    VersionPlatform    `json:"platform" bson:"platform"`
	Version     string             `json:"version" bson:"version"` //V1.1.1 Regex: V\d\.\d\.\d\.
	CommitHash  string             `json:"commit_hash" bson:"commit_hash"`
	UpgradeType VersionUpgradeType `json:"upgrade_type" bson:"upgrade_type"`
	UpgradeCode VersionUpgradeCode `json:"upgrade_code" bson:"upgrade_code"`
	URL         string             `json:"url" bson:"url"`       //	跳转链接
	Prompt      string             `json:"prompt" bson:"prompt"` // 升级提示语
	CreateAt    time.Time          `json:"create_at" bson:"create_at"`
	UpdateAt    time.Time          `json:"update_at" bson:"update_at"`
}

func (v VersionUpgradeCode) Type() VersionUpgradeType {
	switch v {
	case VersionUpgradeCodeForce:
		return VersionUpgradeTypeForce
	case VersionUpgradeCodeNotify:
		return VersionUpgradeTypeNotify
	default:
		return VersionUpgradeTypeNotify
	}
}

type BoardStatusCode int

const (
	BoardStatusCodeOpen  = BoardStatusCode(1)
	BoardStatusCodeClose = BoardStatusCode(2)
)

type BoardStatus string

const (
	BoardStatusOpen  = BoardStatus("board-open")
	BoardStatusClose = BoardStatus("board-close")
)

func (b BoardStatusCode) String() BoardStatus {
	switch b {
	case BoardStatusCodeOpen:
		return BoardStatusOpen
	case BoardStatusCodeClose:
		return BoardStatusClose
	}
	return ""
}

func (b BoardStatus) Int() BoardStatusCode {
	switch b {
	case BoardStatusOpen:
		return BoardStatusCodeOpen
	case BoardStatusClose:
		return BoardStatusCodeClose
	}
	return BoardStatusCodeOpen
}

type BoardCmd string

const (
	BoardCmdOpen  = "cmd-open"
	BoardCmdClose = "cmd-close"
	BoardCmdReset = "cmd-reset"
)

type BoardActionFunc func(action BoardCmd, userId string, interview InterviewDo) (BoardStatus, error)

type BoardDo struct {
	InterviewID string `json:"interview_id" bson:"interview_id"`
	ID          string `json:"id" bson:"_id"`
	//StatusCode BoardStatusCode `json:"status_code" bson:"status_code"`
	Status        BoardStatus `json:"status" bson:"status"`
	CurrentUserID string      `json:"current_user_id" bson:"current_user_id"`
	CreatedAt     time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at" bson:"updated_at"`
}

// TaskResultDo 定时任务的记录: 关闭过期房间\录制已结束面试
type TaskResultDo struct {
	ID         string                            `json:"id" bson:"_id"`
	CreateAt   time.Time                         `json:"create_at" bson:"create_at"`
	UpdateAt   time.Time                         `json:"update_at" bson:"update_at"`
	Subject    string                            `json:"subject" bson:"subject"`
	Action     string                            `json:"action" bson:"action"`
	Result     string                            `json:"result" bson:"result"`
	Status     TaskStatus                        `json:"status" bson:"status"`
	SubjectID  string                            `json:"subject_id" bson:"subject_id"`
	HandleFunc func() (result string, err error) `json:"-" bson:"-"`
	RetryCount int                               `json:"retry_count" bson:"retry_count"`
}

const (
	DefaultTaskRetryCountMax = 5
)

type Task interface {
	Start(c *mgo.Collection, xl *xlog.Logger)
	Handle(handle func() (result string, err error)) Task
}

type TaskStatus string

const (
	TaskStatusRunning = TaskStatus("running")
	TaskStatusSuccess = TaskStatus("success")
	TaskStatusFailed  = TaskStatus("failed")
)

// NewTask
func NewTask(subjectId, subject, action string) *TaskResultDo {
	task := &TaskResultDo{
		Subject:   subject,
		Action:    action,
		SubjectID: subjectId,
	}
	return task
}

func (m *TaskResultDo) beforeRun(c *mgo.Collection, xl *xlog.Logger) (err error) {
	var old TaskResultDo
	condition := bson.M{"subject": m.Subject, "action": m.Action, "subject_id": m.SubjectID}
	err = c.Find(condition).One(&old)
	switch err {
	case mgo.ErrNotFound:
		// new task
		m.ID = utils.GenerateID()
		m.CreateAt = time.Now()
		m.UpdateAt = time.Now()
		m.Status = TaskStatusRunning
		err = c.Insert(*m)
		if err != nil && xl != nil {
			xl.Errorf("error insert task %v err:%v", m.ID, err)
		}
		return nil
	case nil:
		// old task
		m.RetryCount = old.RetryCount + 1
		if m.RetryCount > DefaultTaskRetryCountMax {
			return fmt.Errorf("reach max retry count")
		}
		m.ID = old.ID
		m.CreateAt = old.CreateAt
		m.UpdateAt = time.Now()
		m.Result = old.Result
		m.Status = TaskStatusRunning
		err = c.UpdateId(m.ID, *m)
		if err != nil && xl != nil {
			xl.Errorf("error update task %v err:%v", m.ID, err)
		}
		return err
	}
	return fmt.Errorf("未知错误 %w", err)
}

// success invoke when handle func return nil error
func (m *TaskResultDo) success(c *mgo.Collection, result string, xl *xlog.Logger) {
	m.Result = result
	m.Status = TaskStatusSuccess
	err := c.UpdateId(m.ID, *m)
	if err != nil {
		xl.Errorf("error update task %v:%v", m.ID, err)
	}
}

// failure invoke when handle func return error
func (m *TaskResultDo) failure(c *mgo.Collection, err error, xl *xlog.Logger) {
	m.Result = err.Error()
	m.Status = TaskStatusFailed
	err = c.UpdateId(m.ID, *m)
	if err != nil {
		xl.Errorf("error update task %v:%v", m.ID, err)
	}
}

// Handle set handle func
func (m *TaskResultDo) Handle(handle func() (result string, err error)) (res Task) {
	m.HandleFunc = handle
	return m
}

// Start spawn a goroutine and start task
// fail fast if reach DefaultTaskRetryCountMax
func (m *TaskResultDo) Start(c *mgo.Collection, xl *xlog.Logger) {
	go func() {
		err := m.beforeRun(c, xl)
		if err != nil {
			//m.failure(c, err,xl)
			return
		}
		result, err := m.HandleFunc()
		if err != nil {
			m.failure(c, err, xl)
		} else {
			m.success(c, result, xl)
		}
	}()
}
