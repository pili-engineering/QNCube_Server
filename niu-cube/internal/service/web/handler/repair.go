package handler

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/form"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"github.com/solutions/niu-cube/internal/service/db"
)

type RepairApiHandler struct {
	AppConfigService  db.AppConfigInterface
	Account           AccountInterface
	Repair            db.RepairInterface
	weixin            *cloud.WeixinService
	RTC               *cloud.RTCService
	DefaultAvatarURLs []string
	RequestUrlHost    string
	FrontendUrlHost   string
}

const (
	ADMIN = "admin"
	USER  = "user"
)

func NewRepairApiHandler(conf utils.Config) *RepairApiHandler {
	i := new(RepairApiHandler)
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
	i.Repair, err = db.NewRepairService(*conf.Mongo, nil)
	if err != nil {
		panic(err)
	}
	i.DefaultAvatarURLs = conf.DefaultAvatars
	i.RequestUrlHost = conf.RequestUrlHost
	i.FrontendUrlHost = conf.FrontendUrlHost
	return i
}

func (r *RepairApiHandler) CreateRoom(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)

	//参数获取&校验
	args := &form.RepairCreateForm{}
	err := c.Bind(args)
	if err != nil {
		xl.Infof("invalid args in body, error %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if err := args.Validate(); err != nil {
		xl.Infof("form validate error:%v", err)
		responseErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}

	roomTitle := args.Title
	role := args.Role

	xl.Infof("roomTitle : %s,role: %s", roomTitle, role)

	// 创建rtc房间
	userInfo, userInfoErr := r.Account.GetAccountByID(xl, userID)
	if userInfoErr != nil {
		xl.Errorf("failed to get account info for user %s, ", userID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	// 创建应用自己的房间
	roomID := r.generateRoomID()
	repairRoom := &model.RepairRoomDo{
		ID:         roomID,
		Title:      roomTitle,
		RoomId:     roomID,
		Image:      userInfo.Avatar,
		Status:     int(model.RepairRoomStatusCodeOpen),
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
		Creator:    userID,
		Updator:    userID,
	}
	// 创建七牛IM群ID
	qiniuImGroupId, err := r.AppConfigService.GetGroupId(xl, roomID)
	if err != nil {
		xl.Errorf("failed to craete qiniu's im group, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	repairRoom.QiniuIMGroupId = qiniuImGroupId
	_, err = r.Repair.CreateRoom(xl, repairRoom)
	if err != nil {
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	xl.Infof("createRoom success : %v", repairRoom)

	repairRoomUser := &model.RepairRoomUserDo{
		ID:                roomID + "_" + userID,
		RoomId:            roomID,
		UserID:            userID,
		Role:              role,
		Status:            int(model.RepairRoomUserStatusCodeNormal),
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
		LastHeartBeatTime: time.Now(),
	}
	_, err = r.Repair.CreateRoomUser(xl, repairRoomUser)

	if err != nil {
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	xl.Infof("createRoomUser success  :%v", repairRoomUser)

	// rtc相关的操作
	roomToken := r.RTC.GenerateRTCRoomToken(roomID, userID, ADMIN)
	// 构建返回值
	joinRepairResp := model.JoinRepairResponse{
		UserInfo: model.UserInfoResponse{
			ID:       userInfo.ID,
			Nickname: userInfo.Nickname,
			Avatar:   userInfo.Avatar,
			Phone:    userInfo.Phone,
			Profile:  string(model.DefaultAccountProfile),
		},
		RoomInfo: model.RepairRoomInfoResponse{
			RoomId: repairRoom.RoomId,
			Title:  repairRoom.Title,
			Image:  repairRoom.Image,
			Status: repairRoom.Status,
		},
		RtcInfo: model.RtcInfoResponse{
			RoomToken:   roomToken,
			PublishUrl:  r.RTC.StreamPubURL(roomID),
			RtmpPlayUrl: r.RTC.StreamRtmpPlayURL(roomID),
			FlvPlayUrl:  r.RTC.StreamFlvPlayURL(roomID),
			HlsPlayUrl:  r.RTC.StreamHlsPlayURL(roomID),
		},
		RoomToken:  roomToken,
		PublishURL: r.RTC.StreamPubURL(roomID),
	}

	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      joinRepairResp,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
}

// JoinRoom 用户进入房间
func (r *RepairApiHandler) JoinRoom(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	// 参数获取&校验
	args := &form.RepairJoinForm{}
	err := c.Bind(args)
	if err != nil {
		xl.Infof("invalid args in body, error %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if err := args.Validate(); err != nil {
		xl.Infof("form validate error:%v", err)
		responseErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}
	roomId := args.RoomId
	role := args.Role
	xl.Infof("roomId : %s, role: %s", roomId, role)
	repairRoom, allUserDos, joinRoomErr := r.Repair.JoinRoom(xl, userID, roomId, role)
	// 数据库插入失败
	if joinRoomErr != nil {
		xl.Infof("joinRoomErr error:%v", joinRoomErr)
		responseErr := model.NewResponseErrorJoinRoom()
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}
	allUserList := make([]model.RepairUserInfoResponse, len(allUserDos))
	for index, userDo := range allUserDos {
		accountDo, err := r.Account.GetAccountByID(xl, userDo.UserID)
		if err != nil {
			xl.Errorf("failed to make userInfo response for room %s", roomId)
			continue
		}
		userInfo := model.RepairUserInfoResponse{
			ID:       accountDo.ID,
			Nickname: accountDo.Nickname,
			Avatar:   accountDo.Avatar,
			Phone:    accountDo.Phone,
			Role:     userDo.Role,
			Profile:  model.DefaultAccountProfile,
		}
		allUserList[index] = userInfo
	}
	// rtc相关的操作
	roomToken := r.RTC.GenerateRTCRoomToken(roomId, userID, ADMIN)
	// 构建返回值
	userInfo, userInfoErr := r.Account.GetAccountByID(xl, userID)
	if userInfoErr != nil {
		xl.Errorf("failed to get account info for user %s, ", userID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	joinRepairResp := model.JoinRepairResponse{
		UserInfo: model.UserInfoResponse{
			ID:       userInfo.ID,
			Nickname: userInfo.Nickname,
			Avatar:   userInfo.Avatar,
			Phone:    userInfo.Phone,
			Profile:  string(model.DefaultAccountProfile),
		},
		RoomInfo: model.RepairRoomInfoResponse{
			RoomId: repairRoom.RoomId,
			Title:  repairRoom.Title,
			Image:  repairRoom.Image,
			Status: repairRoom.Status,
		},
		RtcInfo: model.RtcInfoResponse{
			RoomToken:   roomToken,
			PublishUrl:  r.RTC.StreamPubURL(roomId),
			RtmpPlayUrl: r.RTC.StreamRtmpPlayURL(roomId),
			FlvPlayUrl:  r.RTC.StreamFlvPlayURL(roomId),
			HlsPlayUrl:  r.RTC.StreamHlsPlayURL(roomId),
		},
		RoomToken:   roomToken,
		PublishURL:  r.RTC.StreamPubURL(roomId),
		AllUserList: allUserList,
	}
	// 赋值IM群ID
	joinRepairResp.ImConfig = model.ImConfigResponse{
		IMGroupId: repairRoom.QiniuIMGroupId,
		Type:      int(model.ImTypeQiniu),
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      joinRepairResp,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
}

func (r *RepairApiHandler) LeaveRoom(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	roomID := c.Param("roomId")

	if len(roomID) <= 0 || len(roomID) >= 100 {
		xl.Infof("form validate error roomId :%s", roomID)
		responseErr := model.NewResponseErrorBadRequest()
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}
	// 设置roomUser
	err := r.Repair.LeaveRoom(xl, userID, roomID)
	if err != nil {
		xl.Errorf("LeaveRoom fail, roomID:%s, userID:%s", roomID, userID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	// 返回值
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      "",
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
}

func (r *RepairApiHandler) ListRoom(c *gin.Context) {

	// 1.获取查询参数
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	pageNum := c.GetInt(model.PageNumContextKey)
	pageSize := c.GetInt(model.PageSizeContextKey)

	if pageNum < 1 || pageNum > 10000 || pageSize < 1 || pageSize > 100 {
		xl.Infof("form validate pageNum:%v,pageSize:%v", pageNum, pageSize)
		responseErr := model.NewResponseError(model.ResponseErrorValidation, "参数异常")
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}

	// 1.1 从room中查看所有存在的房间
	repairRoomDos, total, err := r.Repair.ListRoomsByPage(xl, userID, pageNum, pageSize)

	if err != nil {
		xl.Errorf("ListRoom fail, userID:%s", userID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	// 2.列表查询
	// 3.返回结果

	roomListResp := &model.RepairRoomListResponse{}
	var repairRoomResponse model.RepairRoomInfoResponse
	for _, repairRoom := range repairRoomDos {
		repairRoomResponse.Image = repairRoom.Image
		repairRoomResponse.RoomId = repairRoom.RoomId
		repairRoomResponse.Title = repairRoom.Title
		repairRoomResponse.Status = repairRoom.Status
		options := []model.RepairRoomOptionResponse{}
		containStaff, _ := r.Repair.ContainStaff(repairRoom.RoomId)

		if !containStaff {
			options = append(options, model.RepairRoomOptionResponse{
				Role:  string(model.RepairRoomRoleStaff),
				Title: string(model.RepairRoomOptionStaff),
			})
		}
		options = append(options, model.RepairRoomOptionResponse{
			Role:  string(model.RepairRoomRoleProfessor),
			Title: string(model.RepairRoomOptionProfessor),
		})
		options = append(options, model.RepairRoomOptionResponse{
			Role:  string(model.RepairRoomRoleStudent),
			Title: string(model.RepairRoomOptionStudent),
		})

		repairRoomResponse.Options = options
		roomListResp.List = append(roomListResp.List, repairRoomResponse)
	}

	roomListResp.Total = total
	roomListResp.Cnt = len(roomListResp.List)
	roomListResp.PageSize = pageSize
	roomListResp.CurrentPageNum = pageNum
	if len(roomListResp.List)+(pageNum-1)*pageSize >= int(total) {
		roomListResp.EndPage = true
		roomListResp.NextPageNum = pageNum
	} else {
		roomListResp.EndPage = false
		roomListResp.NextPageNum = pageNum + 1
	}
	roomListResp.NextId = ""

	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		RequestID: requestID,
		Data:      roomListResp,
	}
	c.JSON(http.StatusOK, resp)

}

func (r *RepairApiHandler) HeartBeat(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	roomID := c.Param("roomId")

	if len(roomID) <= 0 || len(roomID) >= 100 {
		xl.Infof("form validate error roomId :%s", roomID)
		responseErr := model.NewResponseError(model.ResponseErrorValidationRoomId, "房间号不正确")
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}

	// 设置roomUser
	err := r.Repair.HeartBeat(xl, userID, roomID)
	if err != nil {
		xl.Errorf("HeartBeat fail, roomID:%s, userID:%s", roomID, userID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	// 返回值
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.RepairHeartBeatResponse{
			Interval: model.HeartBeatInterval,
		},
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)

}

func (r *RepairApiHandler) GetRoomInfo(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	roomId := c.Param("roomId")

	if len(roomId) <= 0 || len(roomId) >= 100 {
		xl.Infof("form validate error roomId :%s", roomId)
		responseErr := model.NewResponseError(model.ResponseErrorValidationRoomId, "房间号不正确")
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}

	repairRoom, allUserDos, joinRoomErr := r.Repair.GetRoomContent(xl, userID, roomId)
	if joinRoomErr != nil {
		xl.Infof("joinRoomErr error:%v", joinRoomErr)
		responseErr := model.NewResponseError(model.ResponseErrorGetRoomContent, "GetRoomContent fail")
		model.NewFailResponse(*responseErr).WithRequestID(requestID).Send(c)
		return
	}

	allUserList := make([]model.RepairUserInfoResponse, len(allUserDos))
	for index, userDo := range allUserDos {
		accountDo, err := r.Account.GetAccountByID(xl, userDo.UserID)
		if err != nil {
			xl.Errorf("failed to make userInfo response for room %s", roomId)
			continue
		}
		userInfo := model.RepairUserInfoResponse{
			ID:       accountDo.ID,
			Nickname: accountDo.Nickname,
			Avatar:   accountDo.Avatar,
			Phone:    accountDo.Phone,
			Role:     userDo.Role,
			Profile:  model.DefaultAccountProfile,
		}
		allUserList[index] = userInfo
	}

	// 构建返回值
	userInfo, userInfoErr := r.Account.GetAccountByID(xl, userID)
	if userInfoErr != nil {
		xl.Errorf("failed to get account info for user %s, ", userID)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	roomContentResp := model.RepairRoomContentResponse{
		UserInfo: model.UserInfoResponse{
			ID:       userInfo.ID,
			Nickname: userInfo.Nickname,
			Avatar:   userInfo.Avatar,
			Phone:    userInfo.Phone,
			Profile:  string(model.DefaultAccountProfile),
		},
		RoomInfo: model.RepairRoomInfoResponse{
			RoomId: repairRoom.RoomId,
			Title:  repairRoom.Title,
			Image:  repairRoom.Image,
			Status: repairRoom.Status,
		},

		RtcInfo: model.RtcInfoResponse{
			PublishUrl:  r.RTC.StreamPubURL(roomId),
			RtmpPlayUrl: r.RTC.StreamRtmpPlayURL(roomId),
			FlvPlayUrl:  r.RTC.StreamFlvPlayURL(roomId),
			HlsPlayUrl:  r.RTC.StreamHlsPlayURL(roomId),
		},
		PublishURL:  r.RTC.StreamPubURL(roomId),
		AllUserList: allUserList,
	}

	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      roomContentResp,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
}

// generateRoomID 生成直播间ID。
func (r *RepairApiHandler) generateRoomID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	roomID := ""
	idLength := 16
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		roomID = roomID + string(alphaNum[index])
	}
	return roomID
}
