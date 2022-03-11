package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/errors"
	"github.com/solutions/niu-cube/internal/protodef/form"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"github.com/solutions/niu-cube/internal/service/db"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
)

type InterviewApiHandler struct {
	AppConfigService  db.AppConfigInterface
	Account           AccountInterface
	Interview         InterviewInterface
	taskService       *db.TaskService
	weixin            *cloud.WeixinService
	RTC               *cloud.RTCService
	DefaultAvatarURLs []string
	RequestUrlHost    string
	FrontendUrlHost   string
}

type InterviewInterface interface {
	// 创建面试
	CreateInterview(xl *xlog.Logger, interview *model.InterviewDo) (*model.InterviewDo, error)
	//
	ListInterviewsByPage(xl *xlog.Logger, userID string, pageNum int, pageSize int) ([]model.InterviewDo, int, error)
	GetInterviewByID(xl *xlog.Logger, interviewID string) (*model.InterviewDo, error)
	UpdateInterview(xl *xlog.Logger, id string, interview *model.InterviewDo) (*model.InterviewDo, error)
	JoinInterview(xl *xlog.Logger, userID string, interviewID string) ([]model.InterviewUserDo, []model.InterviewUserDo, error)
	LeaveInterview(xl *xlog.Logger, userID string, interviewID string) error
	OnlineInterviewUsers(xl *xlog.Logger, userID string, interviewID string) ([]model.InterviewUserDo, error)
	HeartBeat(xl *xlog.Logger, userId, interviewID string)
	GetRecordURL(xl *xlog.Logger, interviewId string) string
}

func NewInterviewApiHandler(conf utils.Config) *InterviewApiHandler {
	i := new(InterviewApiHandler)
	i.RTC = cloud.NewRtcService(conf)
	//i.weixin = cloud.NewWeixinService(conf)
	var err error
	i.AppConfigService, err = db.NewAppConfigService(conf.IM, nil)
	if err != nil {
		panic(err)
	}
	i.Account, err = db.NewAccountService(*conf.Mongo, nil)
	if err != nil {
		panic(err)
	}
	i.Interview, err = db.NewInterviewService(*conf.Mongo, nil)
	if err != nil {
		panic(err)
	}
	i.taskService = db.NewTaskService(nil, *conf.Mongo)
	i.DefaultAvatarURLs = conf.DefaultAvatars
	i.RequestUrlHost = conf.RequestUrlHost
	i.FrontendUrlHost = conf.FrontendUrlHost
	return i
}

const (
	// DefaultRTCRoomTokenTimeout 默认的RTC加入房间用token的过期时间。
	DefaultRTCRoomTokenTimeout = 60 * time.Second
)

// validateRoomName 校验直播间名称。
func (h *InterviewApiHandler) validateInterviewTitle(roomName string) bool {
	roomNameMaxLength := 100
	if len(roomName) == 0 || len(roomName) > roomNameMaxLength {
		return false
	}
	return true
}

// generateInterviewID 生成直播间ID。
func (h *InterviewApiHandler) generateInterviewID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	roomID := ""
	idLength := 16
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		roomID = roomID + string(alphaNum[index])
	}
	return roomID
}

func (h *InterviewApiHandler) kickOtherUsers(xl *xlog.Logger, roomID string) {
	roomUserIds, _ := h.RTC.ListUser(roomID)
	for _, user := range roomUserIds {
		h.RTC.KickUser(roomID, user)
	}
}

// 生成加入RTC房间的room token。
func (h *InterviewApiHandler) generateRTCRoomToken(roomID string, userID string, permission string) string {
	return h.RTC.GenerateRTCRoomToken(roomID, userID, permission)
}

func (h *InterviewApiHandler) CreatInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	args := &form.InterviewCreateForm{}
	args.FillDefault(c)
	err := c.Bind(args)
	if err != nil {
		xl.Infof("invalid args in body, error %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if err := args.Validate(); err != nil {
		xl.Infof("form validation error: %v", err)
		responseErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}

	candidateByPhone, candidateByPhoneErr := h.Account.GetAccountByPhone(xl, args.CandidatePhone)
	if candidateByPhoneErr != nil {
		if candidateByPhoneErr.Error() == "not found" {
			xl.Infof("candidate's phone number %s not found, create new account", args.CandidatePhone)
			newAccount := &model.AccountDo{
				ID:       h.generateUserID(),
				Nickname: h.generateNicknameByPhone(args.CandidatePhone),
				Phone:    args.CandidatePhone,
				Avatar:   h.generateInitialAvatar(),
			}
			createErr := h.Account.CreateAccount(xl, newAccount)
			if createErr != nil {
				xl.Errorf("failed to craete candidate's account, error %v", err)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
				c.JSON(http.StatusOK, resp)
				return
			}
			candidateByPhone = newAccount
		} else {
			xl.Errorf("get candidate's account by phone number failed, error %v", err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}
	interviewerByPhone := &model.AccountDo{}
	interviewerName := ""
	if args.InterviewerName != "" && args.InterviewerPhone != "" {
		interviewerByArgPhone, err := h.Account.GetAccountByPhone(xl, args.InterviewerPhone)
		if err != nil {
			if err.Error() == "not found" {
				xl.Infof("interviewer's phone number %s not found, create new account", args.InterviewerPhone)
				newAccount := &model.AccountDo{
					ID:       h.generateUserID(),
					Nickname: h.generateNicknameByPhone(args.InterviewerPhone),
					Phone:    args.InterviewerPhone,
					Avatar:   h.generateInitialAvatar(),
				}
				createErr := h.Account.CreateAccount(xl, newAccount)
				if createErr != nil {
					xl.Errorf("failed to create interviewer's account, error %v", err)
					responseErr := model.NewResponseErrorInternal()
					resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
					c.JSON(http.StatusOK, resp)
					return
				}
				interviewerByArgPhone = newAccount
			} else {
				xl.Errorf("get interviewer's account by phone number failed, error %v", err)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
				c.JSON(http.StatusOK, resp)
				return
			}
		}
		interviewerByPhone = interviewerByArgPhone
		interviewerName = args.InterviewerName
	} else {
		userInfo, userInfoErr := h.Account.GetAccountByID(xl, userID)
		if userInfoErr != nil {
			xl.Errorf("get interviewer's account by phone number failed, error %v", err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		interviewerByPhone = userInfo
		interviewerName = userInfo.Nickname
	}

	interviewID := h.generateInterviewID()
	interview := &model.InterviewDo{
		ID:              interviewID,
		Title:           args.Title,
		RoomId:          interviewID,
		StartTime:       time.Unix(args.StartTime, 0),
		EndTime:         time.Unix(args.EndTime, 0),
		Goverment:       args.Goverment,
		Career:          args.Career,
		IsRecord:        args.IsRecorded,
		IsAuth:          args.IsAuth,
		AuthCode:        args.AuthCode,
		Status:          int(model.InterviewStatusCodeInit),
		CreateTime:      time.Now(),
		UpdateTime:      time.Now(),
		Creator:         userID,
		Updator:         userID,
		Interviewer:     interviewerByPhone.ID,
		InterviewerName: interviewerName,
		Candidate:       candidateByPhone.ID,
		CandidateName:   args.CandidateName,
	}

	// 若房间之前不存在，返回创建的房间。若房间已存在，返回已经存在的房间。
	//candidateToken := h.InterviewToken(candidateByPhone.ID)
	//qrcodeURL, err := h.weixin.GetAndUploadQRCode(interview.ID, candidateToken)
	//if err != nil {
	//	xl.Errorf("error get qrcode link err:%v", err)
	//}
	//interview.AppletQrcode = qrcodeURL

	// 创建七牛IM群ID
	qiniuImGroupId, err := h.AppConfigService.GetGroupId(xl, interviewID)
	if err != nil {
		xl.Errorf("failed to create qiniu's im group, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	interview.QiniuIMGroupId = qiniuImGroupId

	interviewRes, err := h.Interview.CreateInterview(xl, interview)
	if err != nil {
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	xl.Infof("user %s created or refreshed interview: ID %s, name %s", userID, interviewRes.ID, args.Title)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.UpsertInterviewResponse{
			ID: interviewRes.ID,
		},
	}
	c.JSON(http.StatusOK, resp)
}

// ListAllInterviews 列出全部房间。
func (h *InterviewApiHandler) ListAllInterviews(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	pageNum := c.GetInt(model.PageNumContextKey)
	pageSize := c.GetInt(model.PageSizeContextKey)
	interviews, total, err := h.Interview.ListInterviewsByPage(xl, userID, pageNum, pageSize)
	if err != nil {
		xl.Errorf("failed to list all rooms, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	interviewListResp := &model.InterviewListResponse{}
	for _, interview := range interviews {
		getInterviewResp, err := h.makeGetInterviewResponse(xl, &interview, userID)
		if err != nil {
			xl.Errorf("failed to make get room response for room %s", interview.ID)
			continue
		}
		interviewListResp.List = append(interviewListResp.List, *getInterviewResp)
	}
	interviewListResp.Total = total
	interviewListResp.Cnt = len(interviewListResp.List)
	interviewListResp.PageSize = pageSize
	interviewListResp.CurrentPageNum = pageNum
	if len(interviewListResp.List)+(pageNum-1)*pageSize >= int(total) {
		interviewListResp.EndPage = true
		interviewListResp.NextPageNum = pageNum
	} else {
		interviewListResp.EndPage = false
		interviewListResp.NextPageNum = pageNum + 1
	}
	interviewListResp.NextId = ""
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data:    interviewListResp,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) makeGetInterviewResponse(xl *xlog.Logger, interview *model.InterviewDo, userID string) (*model.InterviewResponse, error) {
	if interview == nil {
		return nil, fmt.Errorf("nil room")
	}
	candidateInfo, err := h.Account.GetAccountByID(xl, interview.Candidate)
	if err != nil {
		xl.Errorf("failed to get account info for user %s, Candidate of room %s", interview.Candidate, interview.ID)
		return nil, fmt.Errorf("nil Candidate")
	}
	interviewerInfo, err := h.Account.GetAccountByID(xl, interview.Interviewer)
	if err != nil {
		xl.Errorf("failed to get account info for user %s, interviewer of room %s", interview.Interviewer, interview.ID)
		return nil, fmt.Errorf("nil Interviewer")
	}
	interviewStatusName := ""
	roleCode := 0
	roleName := ""
	options := []model.InterviewOptionResponse{}
	switch interview.Status {
	case int(model.InterviewStatusCodeInit):
		interviewStatusName = string(model.InterviewStatusNameInit)
		if interview.Candidate == userID {
			roleCode = int(model.InterviewRoleCodeCandidate)
			roleName = string(model.InterviewRoleNameCandidate)
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeJoin),
				Title:      string(model.InterviewOptionNameJoin),
				RequestUrl: "niucube://interview/joinInterview?interviewId=" + interview.ID,
				Method:     string(model.InterviewOptionMethodNil),
			})
		} else {
			roleCode = int(model.InterviewRoleCodeInterviewer)
			roleName = string(model.InterviewRoleNameInterviewer)
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeJoin),
				Title:      string(model.InterviewOptionNameJoin),
				RequestUrl: "niucube://interview/joinInterview?interviewId=" + interview.ID,
				Method:     string(model.InterviewOptionMethodNil),
			})
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeModify),
				Title:      string(model.InterviewOptionNameModify),
				RequestUrl: "niucube://interview/updateInterview?interviewId=" + interview.ID,
				Method:     string(model.InterviewOptionMethodNil),
			})
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeCancel),
				Title:      string(model.InterviewOptionNameCancel),
				RequestUrl: h.RequestUrlHost + "/v1/cancelInterview/" + interview.ID,
				Method:     string(model.InterviewOptionMethodPost),
			})
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeShare),
				Title:      string(model.InterviewOptionNameShare),
				RequestUrl: "",
				Method:     string(model.InterviewOptionMethodNil),
			})
		}
	case int(model.InterviewStatusCodeStart):
		interviewStatusName = string(model.InterviewStatusNameStart)
		if interview.Candidate == userID {
			roleCode = int(model.InterviewRoleCodeCandidate)
			roleName = string(model.InterviewRoleNameCandidate)
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeJoin),
				Title:      string(model.InterviewOptionNameJoin),
				RequestUrl: "niucube://interview/joinInterview?interviewId=" + interview.ID,
				Method:     string(model.InterviewOptionMethodNil),
			})
		} else {
			roleCode = int(model.InterviewRoleCodeInterviewer)
			roleName = string(model.InterviewRoleNameInterviewer)
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeJoin),
				Title:      string(model.InterviewOptionNameJoin),
				RequestUrl: "niucube://interview/joinInterview?interviewId=" + interview.ID,
				Method:     string(model.InterviewOptionMethodNil),
			})
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeEnd),
				Title:      string(model.InterviewOptionNameEnd),
				RequestUrl: h.RequestUrlHost + "/v1/endInterview/" + interview.ID,
				Method:     string(model.InterviewOptionMethodPost),
			})
			options = append(options, model.InterviewOptionResponse{
				Type:       int(model.InterviewOptionCodeShare),
				Title:      string(model.InterviewOptionNameShare),
				RequestUrl: "",
				Method:     string(model.InterviewOptionMethodNil),
			})
		}
	case int(model.InterviewStatusCodeEnd):
		interviewStatusName = string(model.InterviewStatusNameEnd)
		if interview.Candidate == userID {
			roleCode = int(model.InterviewRoleCodeCandidate)
			roleName = string(model.InterviewRoleNameCandidate)
		} else {
			roleCode = int(model.InterviewRoleCodeInterviewer)
			roleName = string(model.InterviewRoleNameInterviewer)
		}
		//if interview.Recorded{
		//	taskResult,err:=h.taskService.GetTask(xl,"interview","record",interview.ID)
		//	if err==nil{
		//		recordURL = taskResult.Result
		//	}
		//	options = append(options,model.InterviewOptionResponse{
		//		Type: int(model.InterviewOptionCodeView),
		//		Title: "面试录制回看",
		//		RequestUrl: recordURL,
		//		Method: string(model.InterviewOptionMethodGet),
		//	})
		//}
	default:
		interviewStatusName = string(model.InterviewStatusNameEnd)
		if interview.Candidate == userID {
			roleCode = int(model.InterviewRoleCodeCandidate)
			roleName = string(model.InterviewRoleNameCandidate)
		} else {
			roleCode = int(model.InterviewRoleCodeInterviewer)
			roleName = string(model.InterviewRoleNameInterviewer)
		}
	}

	candidateMap, _ := json.Marshal(map[string]string{
		"userId": interview.Candidate,
	})
	candidateUrl := h.FrontendUrlHost + "/meeting-entrance/" + interview.ID + "?interviewToken=" + base64.StdEncoding.EncodeToString([]byte(candidateMap))
	interviewTime := interview.StartTime.Format("2006-01-02 15:04")
	candidateContent := fmt.Sprintf("您的面试部门为：%s，职位：%s，时间为：%s。请提前预留时间参加面试，面试链接为：%s（请使用电脑浏览器打开链接并进行）", interview.Goverment, interview.Career, interviewTime, candidateUrl)
	var recordURL string
	if interview.Recorded {
		recordURL = h.Interview.GetRecordURL(xl, interview.ID)
	}
	interviewResp := model.InterviewResponse{
		ID:               interview.ID,
		Title:            interview.Title,
		Goverment:        interview.Goverment,
		Career:           interview.Career,
		CandidateID:      interview.Candidate,
		CandidateName:    interview.CandidateName,
		CandidatePhone:   candidateInfo.Phone,
		InterviewerID:    interview.Interviewer,
		InterviewerName:  interview.InterviewerName,
		InterviewerPhone: interviewerInfo.Phone,
		StartTime:        interview.StartTime.Unix(),
		EndTime:          interview.EndTime.Unix(),
		Status:           interviewStatusName,
		StatusCode:       interview.Status,
		RoleCode:         roleCode,
		Role:             roleName,
		IsAuth:           interview.IsAuth,
		AuthCode:         interview.AuthCode,
		IsRecorded:       interview.IsRecord,
		Options:          options,
		RecordURL:        recordURL,
		AppletQrcode:     interview.AppletQrcode,
		ShareInfo: model.ShareInfoResponse{
			Url:     candidateUrl,
			Icon:    "https://demo-qnrtc-files.qnsdk.com/default_icon.png",
			Content: candidateContent,
		},
	}

	return &interviewResp, nil
}

func (h *InterviewApiHandler) GetInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")

	interview, err := h.Interview.GetInterviewByID(xl, interviewID)
	if err != nil {
		serverErr, ok := err.(*errors.ServerError)
		if ok {
			switch serverErr.Code {
			case errors.ServerErrorRoomNotFound:
				responseErr := model.NewResponseErrorNoSuchInterview()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
				c.JSON(http.StatusOK, resp)
				return
			}
		}
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	interviewResp, err := h.makeGetInterviewResponse(xl, interview, userID)
	if err != nil {
		xl.Errorf("failed to get make get room response, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	xl.Debugf("user %s get info of room %s", userID, interviewID)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data:    interviewResp,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) UpdateInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")
	//args := model.InterviewArgs{}
	args := form.InterviewUpdateForm{}
	err := c.Bind(&args)
	if err != nil {
		xl.Infof("invalid args in body, error %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if err := args.Validate(); err != nil {
		xl.Infof("form valdation error:%v", err)
		responseErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}
	interview, err := h.Interview.GetInterviewByID(xl, interviewID)
	if err != nil {
		// todo
		serverErr, ok := err.(*errors.ServerError)
		if !ok {
			xl.Errorf("failed to get current interview %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		switch serverErr.Code {
		case errors.ServerErrorRoomNotFound:
			xl.Infof("room %s not found", interviewID)
			responseErr := model.NewResponseErrorNoSuchInterview()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		default:
			xl.Errorf("failed to get current interview %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	if interview.Creator != userID && interview.Interviewer != userID {
		xl.Infof("user %s try to update interview %s, no permission", userID, interviewID)
		responseErr := model.NewResponseErrorUnauthorized()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	if args.Title != interview.Title {
		interview.Title = args.Title
	}
	if args.Title != interview.Title {
		interview.Title = args.Title
	}
	if time.Unix(args.StartTime, 0) != interview.StartTime {
		interview.StartTime = time.Unix(args.StartTime, 0)
	}
	if time.Unix(args.EndTime, 0) != interview.EndTime {
		interview.EndTime = time.Unix(args.EndTime, 0)
	}
	if args.Goverment != interview.Goverment {
		interview.Goverment = args.Goverment
	}
	if args.Career != interview.Career {
		interview.Career = args.Career
	}
	if args.IsRecorded != interview.IsRecord {
		interview.IsRecord = args.IsRecorded
	}
	if args.IsAuth != interview.IsAuth {
		interview.IsAuth = args.IsAuth
	}
	if args.AuthCode != interview.AuthCode {
		interview.AuthCode = args.AuthCode
	}
	interview.UpdateTime = time.Now()
	interview.Updator = userID
	if args.InterviewerName != interview.InterviewerName {
		interview.InterviewerName = args.InterviewerName
	}
	if args.CandidateName != interview.CandidateName {
		interview.CandidateName = args.CandidateName
	}
	candidateByPhone, candidateByPhoneErr := h.Account.GetAccountByPhone(xl, args.CandidatePhone)
	if candidateByPhoneErr != nil {
		if candidateByPhoneErr.Error() == "not found" {
			xl.Infof("candidate's phone number %s not found, create new account", args.CandidatePhone)
			newAccount := &model.AccountDo{
				ID:       h.generateUserID(),
				Nickname: h.generateNicknameByPhone(args.CandidatePhone),
				Phone:    args.CandidatePhone,
				Avatar:   h.generateInitialAvatar(),
			}
			createErr := h.Account.CreateAccount(xl, newAccount)
			if createErr != nil {
				xl.Errorf("failed to craete candidate's account, error %v", err)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
				c.JSON(http.StatusOK, resp)
				return
			}
			candidateByPhone = newAccount
		} else {
			xl.Errorf("get candidate's account by phone number failed, error %v", err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}
	interview.Candidate = candidateByPhone.ID
	interviewerByPhone := &model.AccountDo{}
	interviewerName := ""
	if args.InterviewerName != "" && args.InterviewerPhone != "" {
		interviewerByArgPhone, err := h.Account.GetAccountByPhone(xl, args.InterviewerPhone)
		if err != nil {
			if err.Error() == "not found" {
				xl.Infof("interviewer's phone number %s not found, create new account", args.InterviewerPhone)
				newAccount := &model.AccountDo{
					ID:       h.generateUserID(),
					Nickname: h.generateNicknameByPhone(args.InterviewerPhone),
					Phone:    args.InterviewerPhone,
					Avatar:   h.generateInitialAvatar(),
				}
				createErr := h.Account.CreateAccount(xl, newAccount)
				if createErr != nil {
					xl.Errorf("failed to craete interviewer's account, error %v", err)
					responseErr := model.NewResponseErrorInternal()
					resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
					c.JSON(http.StatusOK, resp)
					return
				}
				interviewerByArgPhone = newAccount
			} else {
				xl.Errorf("get interviewer's account by phone number failed, error %v", err)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
				c.JSON(http.StatusOK, resp)
				return
			}
		}
		interviewerByPhone = interviewerByArgPhone
		interviewerName = args.InterviewerName
	} else {
		userInfo, userInfoErr := h.Account.GetAccountByID(xl, userID)
		if userInfoErr != nil {
			xl.Errorf("get interviewer's account by phone number failed, error %v", err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		interviewerByPhone = userInfo
		interviewerName = userInfo.Nickname
	}
	interview.InterviewerName = interviewerName
	interview.Interviewer = interviewerByPhone.ID

	interview, err = h.Interview.UpdateInterview(xl, interview.ID, interview)
	if err != nil {
		xl.Errorf("failed to update room, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.UpsertInterviewResponse{
			ID: interview.ID,
		},
	}
	xl.Infof("room %s updated by user %s", interviewID, userID)
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) EndInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")

	interview, err := h.changeInterviewStatus(xl, userID, interviewID, model.InterviewStatusCodeStart, model.InterviewStatusCodeEnd)
	if err != nil {
		xl.Errorf("failed to change Interview Status room %s, error %v", interviewID, err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	h.kickOtherUsers(xl, interview.ID)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.UpsertInterviewResponse{
			ID: interview.ID,
		},
	}
	xl.Infof("room %s updated by user %s", interviewID, userID)
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) changeInterviewStatus(xl *xlog.Logger, userID string, interviewID string, from model.InterviewStatusCode, to model.InterviewStatusCode) (*model.InterviewDo, error) {
	interview, err := h.Interview.GetInterviewByID(xl, interviewID)
	if err != nil {
		serverErr, ok := err.(*errors.ServerError)
		if !ok {
			return nil, fmt.Errorf("failed to get current room %s, error %v", interviewID, err)
		}
		switch serverErr.Code {
		case errors.ServerErrorRoomNotFound:
			return nil, fmt.Errorf("room %s not found", interviewID)
		default:
			return nil, fmt.Errorf("failed to get current room %s, error %v", interviewID, err)
		}
	}

	if interview.Creator != userID && interview.Interviewer != userID {
		return nil, fmt.Errorf("user %s try to update room %s, no permission", userID, interviewID)
	}

	if interview.Status != int(from) {
		// TODO 当前面试状态不能取消
		return nil, fmt.Errorf("user %s try to update room %s, no permission", userID, interviewID)
	}

	interview.Status = int(to)
	interview, err = h.Interview.UpdateInterview(xl, interview.ID, interview)
	if err != nil {
		return nil, fmt.Errorf("failed to update room, error %v", err)
	}
	return interview, nil
}

func (h *InterviewApiHandler) CancelInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")

	interview, err := h.changeInterviewStatus(xl, userID, interviewID, model.InterviewStatusCodeInit, model.InterviewStatusCodeEnd)
	if err != nil {
		xl.Errorf("failed to change Interview Status room %s, error %v", interviewID, err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.UpsertInterviewResponse{
			ID: interview.ID,
		},
	}
	xl.Infof("room %s updated by user %s", interviewID, userID)
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) JoinInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")

	userInfo, userInfoErr := h.Account.GetAccountByID(xl, userID)
	if userInfoErr != nil {
		xl.Errorf("failed to get account info for user %s, Candidate of room %s", userID, interviewID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	interview, err := h.Interview.GetInterviewByID(xl, interviewID)
	if err != nil {
		serverErr, ok := err.(*errors.ServerError)
		if !ok {
			xl.Errorf("failed to get current room %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		switch serverErr.Code {
		case errors.ServerErrorRoomNotFound:
			xl.Infof("room %s not found", interviewID)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		default:
			xl.Errorf("failed to get current room %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	if interview.Creator != userID && interview.Interviewer != userID && interview.Candidate != userID {
		xl.Infof("user %s try to update room %s, no permission", userID, interviewID)
		responseErr := model.NewResponseErrorNoSuchInterview()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	interviewResp, err := h.makeGetInterviewResponse(xl, interview, userID)
	if err != nil {
		xl.Errorf("failed to make get room response for room %s", interview.ID)
		responseErr := model.NewResponseErrorNoSuchInterview()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
	}
	permission := ""
	switch interviewResp.RoleCode {
	case int(model.InterviewRoleCodeInterviewer):
		permission = "admin"
	default:
		permission = "user"
	}

	onlineUserDos, allUserDos, joinInterviewErr := h.Interview.JoinInterview(xl, userID, interviewID)
	if joinInterviewErr != nil {
		xl.Errorf("failed to get current room %s, error %v", interviewID, err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	onlineUserList := make([]model.UserInfoResponse, len(onlineUserDos))
	for index, interviewUserDo := range onlineUserDos {
		accountDo, err := h.Account.GetAccountByID(xl, interviewUserDo.UserID)
		if err != nil {
			xl.Errorf("failed to make userInfo response for room %s", interview.ID)
			continue
		}
		userInfo := model.UserInfoResponse{
			ID:       accountDo.ID,
			Nickname: accountDo.Nickname,
			Avatar:   accountDo.Avatar,
			Phone:    accountDo.Phone,
			Profile:  model.DefaultAccountProfile,
		}
		onlineUserList[index] = userInfo
	}

	allUserList := make([]model.UserInfoResponse, len(allUserDos))
	for index, interviewUserDo := range allUserDos {
		accountDo, err := h.Account.GetAccountByID(xl, interviewUserDo.UserID)
		if err != nil {
			xl.Errorf("failed to make userInfo response for room %s", interview.ID)
			continue
		}
		userInfo := model.UserInfoResponse{
			ID:       accountDo.ID,
			Nickname: accountDo.Nickname,
			Avatar:   accountDo.Avatar,
			Phone:    accountDo.Phone,
			Profile:  model.DefaultAccountProfile,
		}
		allUserList[index] = userInfo
	}

	jointInterviewResp := model.JoinInterviewResponse{
		Interview: *interviewResp,
		UserInfo: model.UserInfoResponse{
			ID:       userInfo.ID,
			Nickname: userInfo.Nickname,
			Avatar:   userInfo.Avatar,
			Phone:    userInfo.Phone,
			Profile:  string(model.DefaultAccountProfile),
		},
		RoomToken:      h.generateRTCRoomToken(interviewID, userID, permission),
		OnlineUserList: onlineUserList,
		PublishURL:     h.RTC.StreamPubURL(interviewID),
		AllUserList:    allUserList,
	}

	// 赋值IM群ID
	jointInterviewResp.ImConfig = model.ImConfigResponse{
		IMGroupId: interview.QiniuIMGroupId,
		Type:      int(model.ImTypeQiniu),
	}

	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data:    jointInterviewResp,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) LeaveInterview(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")
	interview, err := h.Interview.GetInterviewByID(xl, interviewID)
	if err != nil {
		serverErr, ok := err.(*errors.ServerError)
		if !ok {
			xl.Errorf("failed to get current room %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		switch serverErr.Code {
		case errors.ServerErrorRoomNotFound:
			xl.Infof("room %s not found", interviewID)
			responseErr := model.NewResponseErrorNoSuchInterview()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		default:
			xl.Errorf("failed to get current room %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	if interview.Creator != userID && interview.Interviewer != userID && interview.Candidate != userID {
		xl.Infof("user %s try to update room %s, no permission", userID, interviewID)
		responseErr := model.NewResponseErrorNoSuchInterview()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	leavelErr := h.Interview.LeaveInterview(xl, userID, interviewID)
	if leavelErr != nil {
		xl.Infof("error when leaving room, error: %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
	}

	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data:    "",
	}
	xl.Infof("LeaveInterview  %s updated by user %s", interviewID, userID)
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) HeartBeat(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	interviewID := c.Param("interviewId")

	interviewUserDos, onlineInterviewUsersErr := h.Interview.OnlineInterviewUsers(xl, userID, interviewID)
	if onlineInterviewUsersErr != nil {
		xl.Errorf("failed to get current room %s, error %v", interviewID, onlineInterviewUsersErr)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	// mark heartbeat time
	h.Interview.HeartBeat(xl, userID, interviewID)

	onlineUserList := make([]model.UserInfoResponse, len(interviewUserDos))
	for index, interviewUserDo := range interviewUserDos {
		interviewUserId := interviewUserDo.UserID
		accountDo, err := h.Account.GetAccountByID(xl, interviewUserId)
		if err != nil {
			xl.Errorf("failed to make userInfo response for room %s", interviewID)
			continue
		}
		userInfo := model.UserInfoResponse{
			ID:       accountDo.ID,
			Nickname: accountDo.Nickname,
			Avatar:   accountDo.Avatar,
			Phone:    accountDo.Phone,
			Profile:  model.DefaultAccountProfile,
		}
		onlineUserList[index] = userInfo
	}

	interview, err := h.Interview.GetInterviewByID(xl, interviewID)
	if err != nil {
		serverErr, ok := err.(*errors.ServerError)
		if !ok {
			xl.Errorf("failed to get current room %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		switch serverErr.Code {
		case errors.ServerErrorRoomNotFound:
			xl.Infof("room %s not found", interviewID)
			responseErr := model.NewResponseErrorNoSuchInterview()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		default:
			xl.Errorf("failed to get current room %s, error %v", interviewID, err)
			responseErr := model.NewResponseErrorNoSuchInterview()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}
	options := model.HeartBeatOptionResponse{
		ShowLeaveInterview: false,
	}
	if interview.Status == int(model.InterviewStatusCodeStart) && (interview.Creator == userID || interview.Interviewer == userID) {
		options.ShowLeaveInterview = true
	}

	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.HeartBeatResponse{
			Interval:       model.HeartBeatInterval,
			OnlineUserList: onlineUserList,
			Options:        options,
		},
	}
	xl.Infof("HeartBeat  %s updated by user %s", interviewID, userID)
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) generateNicknameByPhone(phone string) string {
	namePrefix := "用户_"
	if len(phone) < 4 {
		return namePrefix + phone
	}
	return namePrefix + phone[len(phone)-4:]
}

func (h *InterviewApiHandler) generateInitialAvatar() string {
	if len(h.DefaultAvatarURLs) == 0 {
		return ""
	}
	index := rand.Intn(len(h.DefaultAvatarURLs))
	return h.DefaultAvatarURLs[index]
}

// generateUserID 生成新的用户ID。
func (h *InterviewApiHandler) generateUserID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	idLength := 12
	id := ""
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		id = id + string(alphaNum[index])
	}
	return id
}

func (h *InterviewApiHandler) InterviewUrlFromId(c *gin.Context) {
	interviewID := c.Param("interviewId")
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	interview, _ := h.Interview.GetInterviewByID(xl, interviewID)
	interviewer, _ := json.Marshal(map[string]string{
		"userId": interview.Interviewer,
	})
	candidate, _ := json.Marshal(map[string]string{
		"userId": interview.Candidate,
	})
	resp := model.NewSuccessResponse(map[string]string{
		"interviewer": h.FrontendUrlHost + "/meeting-entrance/" + interviewID + "?interviewToken=" + base64.StdEncoding.EncodeToString([]byte(interviewer)),
		"candidate":   h.FrontendUrlHost + "/meeting-entrance/" + interviewID + "?interviewToken=" + base64.StdEncoding.EncodeToString([]byte(candidate)),
	})
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewApiHandler) InterviewToken(id string) string {
	payload := map[string]string{
		"userId": id,
	}
	val, _ := json.Marshal(payload)
	return base64.StdEncoding.EncodeToString(val)
}
