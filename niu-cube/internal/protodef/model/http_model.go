// Copyright 2020 Qiniu Cloud (qiniu.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
	protocol.go: 规定API的参数与返回值的定义，***Args 表示 *** 接口的参数，***Response表示 *** 接口的返回体格式。
*/

const (
	// RequestIDHeader 七牛 request ID 头部。
	RequestIDHeader = "X-Reqid"
	// XLogKey gin context中，用于获取记录请求相关日志的 xlog logger的key。
	XLogKey = "xlog-logger"

	// LoginTokenKey 登录用的token。
	LoginTokenKey = "qiniu-cube-login-token"

	// UserIDContextKey 存放在请求context 中的用户ID。
	UserIDContextKey = "userID"
	// UserContextKey 存放用户对象
	UserContextKey = "user"

	//ActionLogContentKey 用于存放log
	ActionLogContentKey = "action-log"

	// TokenSourceContextKey 存放在请求context 中的TOKEN获取来源
	TokenSourceContextKey = "tokenSource"
	// TOKEN获取来源
	TokenSourceFromInterviewToken TokenSource = "interviewToken"
	TokenSourceFromHeader         TokenSource = "header"

	UAContextKey            = "UA"
	UAMobile        UAValue = "mobile"
	UAMobileAndroid UAValue = "android"
	UAMobileApple   UAValue = "apple"
	UANoneMobile    UAValue = "noneMobile"

	// UserIDContextKey 存放在请求context 中的用户ID。
	PageNumContextKey  = "pageNum"
	PageSizeContextKey = "pageSize"

	// RequestStartKey 存放在gin context中的请求开始的时间戳，单位为纳秒。
	RequestStartKey = "request-start-timestamp-nano"

	// RequestApiVersion
	RequestApiVersion            = "request-api-version"
	ApiVersionV1      ApiVersion = "v1"
	ApiVersionV2      ApiVersion = "v2"

	HeartBeatInterval = 30

	// 状态码和状态信息
	ResponseStatusCodeSuccess    ResponseStatusCode    = 0
	ResponseStatusMessageSuccess ResponseStatusMessage = "success"

	DefaultAccountProfile = "MediaPaaS的第一选择~"

	ImTypeRongyun ImType = 1
	ImTypeQiniu   ImType = 2
)

// API Version
type ApiVersion string

// token来源枚举
type TokenSource string
type UAValue string

// 状态码和状态信息
type ResponseStatusCode int
type ResponseStatusMessage string

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"requestId"`
}

// NewHTTPErrorBadRequest 一般的HTTP bad request 错误。
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    int(ResponseStatusCodeSuccess),
		Message: string(ResponseStatusMessageSuccess),
		Data:    data,
	}
}

// NewHTTPErrorBadRequest 一般的HTTP bad request 错误。
func NewFailResponse(err ResponseError) *Response {
	return &Response{
		Code:    int(err.Code),
		Message: string(err.Message),
	}
}

func (r *Response) WithRequestID(requestID string) *Response {
	r.RequestID = requestID
	return r
}

func (r *Response) WithErrorMessage(message string) *Response {
	r.Message = string(message)
	return r
}

func (r *Response) Send(c *gin.Context) {
	c.JSON(http.StatusOK, r)
}

type WelcomeResponse struct {
	Image string `json:"image"`
	Url   string `json:"url"`
}

type AppConfigResponse struct {
	WelcomeResponse `json:"welcome"`
}

type Pagination struct {
	Total          int           `json:"total"`
	NextId         string        `json:"nextId"`
	Cnt            int           `json:"cnt"`
	CurrentPageNum int           `json:"currentPageNum"`
	NextPageNum    int           `json:"nextPageNum"`
	PageSize       int           `json:"pageSize"`
	EndPage        bool          `json:"endPage"`
	List           []interface{} `json:"list"`
}

type SolutionResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Url   string `json:"url"`
	Desc  string `json:"desc"`
	Icon  string `json:"icon"`
}

type SolutionListResponse struct {
	Pagination
}

// UserInfoResponse 用户的信息，包括ID、昵称等。
type UserInfoResponse struct {
	ID       string `json:"accountId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Phone    string `json:"phone"`
	Profile  string `json:"profile"`
}

// RepairRoomInfoResponse 房间信息表
type RepairRoomInfoResponse struct {
	RoomId  string                     `json:"roomId"`
	Title   string                     `json:"title"`
	Image   string                     `json:"image"`
	Status  int                        `json:"status"`
	Options []RepairRoomOptionResponse `json:"options"`
}

// GetSmsCodeArgs 通过短信登录的参数
type GetSmsCodeArgs struct {
	Phone string `json:"phone" form:"phone"`
}

// SMSLoginArgs 通过短信登录的参数
type SMSLoginArgs struct {
	Phone   string `json:"phone" form:"phone"`
	SMSCode string `json:"smsCode" form:"smsCode"`
}

type ImType int

type ImConfigResponse struct {
	ImToken    string `json:"imToken"`
	Type       int    `json:"type"`
	IMUsername string `json:"imUsername"`
	IMPassword string `json:"imPassword"`
	IMUid      string `json:"imUid"`
	IMGroupId  int64  `json:"imGroupId"`
}

// SignUpOrInResponse 登录的返回结果。
type SignUpOrInResponse struct {
	UserInfoResponse
	Token            string `json:"loginToken"`
	ImConfigResponse `json:"imConfig"`
}

// UpdateAccountInfoArgs 修改用户信息接口。
type UpdateAccountInfoArgs struct {
	Nickname string `json:"nickname" form:"nickname"`
	Avatar   string `json:"avatar,omitempty" form:"avatar,omitempty"`
}

// UpdateAccountInfoResponse 修改用户信息的返回结果。
type UpdateAccountInfoResponse struct {
	UserInfoResponse
}

// GetAccountInfoResponse 查询用户信息的返回结果。
type GetAccountInfoResponse struct {
	UserInfoResponse
}

// 面试相关
// InterviewArgs 创建或者更新的面试对象
type InterviewArgs struct {
	Title            string `form:"title"`
	StartTime        int64  `form:"startTime"`
	EndTime          int64  `form:"endTime"`
	Goverment        string `form:"goverment"`
	Career           string `form:"career"`
	IsAuth           bool   `form:"isAuth"`
	AuthCode         string `form:"authCode"`
	IsRecorded       bool   `form:"isRecorded"`
	CandidateName    string `form:"candidateName"`
	CandidatePhone   string `form:"candidatePhone"`
	InterviewerName  string `form:"interviewerName"`
	InterviewerPhone string `form:"interviewerPhone"`
}

// UpsertInterviewResponse 创建或者更新的面试结果
type UpsertInterviewResponse struct {
	ID string `json:"id"`
}

// InterviewListResponse 面试列表结果
type InterviewListResponse struct {
	Pagination
}

type RepairRoomListResponse struct {
	Pagination
}

type InterviewOptionCode int
type InterviewOptionName string
type InterviewOptionMethod string

const (
	InterviewOptionCodeModify InterviewOptionCode   = 1
	InterviewOptionCodeView   InterviewOptionCode   = 2
	InterviewOptionCodeCancel InterviewOptionCode   = 50
	InterviewOptionCodeEnd    InterviewOptionCode   = 51
	InterviewOptionCodeJoin   InterviewOptionCode   = 100
	InterviewOptionCodeLeave  InterviewOptionCode   = 150
	InterviewOptionCodeShare  InterviewOptionCode   = 200
	InterviewOptionNameModify InterviewOptionName   = "修改面试"
	InterviewOptionNameView   InterviewOptionName   = "查看面试"
	InterviewOptionNameCancel InterviewOptionName   = "取消面试"
	InterviewOptionNameEnd    InterviewOptionName   = "结束面试"
	InterviewOptionNameJoin   InterviewOptionName   = "进入面试"
	InterviewOptionNameLeave  InterviewOptionName   = "离开面试"
	InterviewOptionNameShare  InterviewOptionName   = "分享面试"
	InterviewOptionMethodNil  InterviewOptionMethod = ""
	InterviewOptionMethodGet  InterviewOptionMethod = "GET"
	InterviewOptionMethodPost InterviewOptionMethod = "POST"
)

type InterviewOptionResponse struct {
	Type       int    `json:"type"`
	Title      string `json:"title"`
	RequestUrl string `json:"requestUrl"`
	Method     string `json:"method"`
}

type ShareInfoResponse struct {
	Url     string `json:"url"`
	Icon    string `json:"icon"`
	Content string `json:"content"`
}

type InterviewResponse struct {
	ID               string                    `json:"id"`
	Title            string                    `json:"title"`
	Goverment        string                    `json:"goverment"`
	Career           string                    `json:"career"`
	CandidateID      string                    `json:"candidateId"`
	CandidateName    string                    `json:"candidateName"`
	CandidatePhone   string                    `json:"candidatePhone"`
	InterviewerName  string                    `json:"interviewerName"`
	InterviewerPhone string                    `json:"interviewerPhone"`
	InterviewerID    string                    `json:"interviewerId"`
	StartTime        int64                     `json:"startTime"`
	EndTime          int64                     `json:"endTime"`
	Status           string                    `json:"status"`
	StatusCode       int                       `json:"statusCode"`
	RoleCode         int                       `json:"roleCode"`
	Role             string                    `json:"role"`
	IsAuth           bool                      `json:"isAuth"`
	AuthCode         string                    `json:"authCode"`
	IsRecorded       bool                      `json:"isRecorded"`
	AppletQrcode     string                    `json:"appletQrcode"`
	Recorded         bool                      `json:"recorded"`
	RecordURL        string                    `json:"recordUrl"`
	Options          []InterviewOptionResponse `json:"options"`
	ShareInfo        ShareInfoResponse         `json:"shareInfo"`
}

type InterviewTokenArgs struct {
	UserID string `json:"userId"`
}

type JoinInterviewResponse struct {
	Interview      InterviewResponse  `json:"interview"`
	UserInfo       UserInfoResponse   `json:"userInfo"`
	RoomToken      string             `json:"roomToken"`
	OnlineUserList []UserInfoResponse `json:"onlineUserList"`
	AllUserList    []UserInfoResponse `json:"allUserList"`
	PublishURL     string             `json:"publishUrl"`
	ImConfig       ImConfigResponse   `json:"imConfig"`
}

type JoinRepairResponse struct {
	UserInfo    UserInfoResponse         `json:"userInfo"`
	RoomToken   string                   `json:"roomToken"`
	PublishURL  string                   `json:"publishUrl"`
	RoomInfo    RepairRoomInfoResponse   `json:"roomInfo"`
	AllUserList []RepairUserInfoResponse `json:"allUserList"`
	RtcInfo     RtcInfoResponse          `json:"rtcInfo"`
	ImConfig    ImConfigResponse         `json:"imConfig"`
}

type RepairRoomContentResponse struct {
	UserInfo    UserInfoResponse         `json:"userInfo"`
	PublishURL  string                   `json:"publishUrl"`
	RoomInfo    RepairRoomInfoResponse   `json:"roomInfo"`
	AllUserList []RepairUserInfoResponse `json:"allUserList"`
	RtcInfo     RtcInfoResponse          `json:"rtcInfo"`
}

type HeartBeatOptionResponse struct {
	ShowLeaveInterview bool `json:"showLeaveInterview"`
}

type HeartBeatResponse struct {
	Interval       int                     `json:"interval"`
	OnlineUserList []UserInfoResponse      `json:"onlineUserList"`
	Options        HeartBeatOptionResponse `json:"options"`
}

type IERoomResponse struct {
	RoomID     string `json:"roomId"`
	Title      string `json:"title"`
	Notice     string `json:"notice"`
	RoomAvatar string `json:"roomAvatar"`
}

type JoinIERoomResponse struct {
	Room       IERoomResponse   `json:"room"`
	Creator    UserInfoResponse `json:"creator"`
	RoomToken  string           `json:"roomToken"`
	PublishURL string           `json:"publishUrl"`
	PullURL    string           `json:"pullUrl"`
	ImConfig   ImConfigResponse `json:"imConfig"`
}

// RepairUserInfoResponse 用户的信息，包括ID、昵称等。
type RepairUserInfoResponse struct {
	ID       string `json:"accountId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	Profile  string `json:"profile"`
}

type RepairHeartBeatResponse struct {
	Interval int `json:"interval"`
}

type RepairRoomOptionResponse struct {
	Role  string `json:"role"`
	Title string `json:"title"`
}

type RtcInfoResponse struct {
	RoomToken   string `json:"roomToken"`
	PublishUrl  string `json:"publishUrl"`
	RtmpPlayUrl string `json:"rtmpPlayUrl"`
	FlvPlayUrl  string `json:"flvPlayUrl"`
	HlsPlayUrl  string `json:"hlsPlayUrl"`
}
