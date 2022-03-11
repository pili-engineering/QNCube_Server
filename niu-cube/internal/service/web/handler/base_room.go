package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/solutions/niu-cube/internal/service/db"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	dao2 "github.com/solutions/niu-cube/internal/service/dao"
)

type BaseRoomApi interface {
	CreateRoom(context *gin.Context)

	JoinRoom(context *gin.Context)

	LeaveRoom(context *gin.Context)

	ListRooms(context *gin.Context)

	RoomInfo(context *gin.Context)

	UpdateRoomInfo(context *gin.Context)

	RoomInfoAttr(context *gin.Context)

	TruncateGroupChat(context *gin.Context)
}

type BaseRoomApiHandler struct {
	baseRoomDao      dao2.BaseRoomDaoInterface
	baseUserDao      dao2.BaseUserDaoInterface
	baseMicDao       dao2.BaseMicDaoInterface
	baseRoomUserDao  dao2.BaseRoomUserDaoInterface
	baseUserMicDao   dao2.BaseUserMicDaoInterface
	baseRoomMicDao   dao2.BaseRoomMicDaoInterface
	roomUserSongDao  dao2.RoomUserSongDaoInterface
	roomUserMovieDao dao2.RoomUserMovieInterface
	rtcService       *cloud.RTCService
	appConfigService db.AppConfigInterface
	xl               *xlog.Logger
}

func NewBaseRoomApiHandler(xl *xlog.Logger, config *utils.Config) *BaseRoomApiHandler {
	baseRoomDao, err := dao2.NewBaseRoomDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create BaseRoomDaoService failed.")
		return nil
	}
	baseUserDao, err := dao2.NewBaseUserDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create BaseUserDaoService failed.")
		return nil
	}
	baseMicDao, err := dao2.NewBaseMicDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create BaseMicDaoService failed")
		return nil
	}
	baseRoomUserDao, err := dao2.NewBaseRoomUserDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create BaseRoomUserDaoService failed")
		return nil
	}
	baseUserMicDao, err := dao2.NewBaseUserMicDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create BaseUserMicDaoService failed")
		return nil
	}
	baseRoomMicDao, err := dao2.NewBaseRoomMicDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create BaseRoomMicDaoService failed")
		return nil
	}
	roomUserSongDao, err := dao2.NewRoomUserSongDaoService(xl, config.Mongo)
	if err != nil {
		xl.Error("create RoomUserSongDaoService failed.")
		return nil
	}
	roomUserMovieDao, err := dao2.NewRoomUserMovieService(xl, config.Mongo)
	if err != nil {
		xl.Error("create RoomUserMovieDaoService failed.")
		return nil
	}
	rtcService := cloud.NewRtcService(*config)
	appConfigService, _ := db.NewAppConfigService(config.IM, xl)
	if xl == nil {
		xl = xlog.New("base-room-logger")
	}
	return &BaseRoomApiHandler{
		baseRoomDao,
		baseUserDao,
		baseMicDao,
		baseRoomUserDao,
		baseUserMicDao,
		baseRoomMicDao,
		roomUserSongDao,
		roomUserMovieDao,
		rtcService,
		appConfigService,
		xl,
	}
}

func (b *BaseRoomApiHandler) CreateRoom(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	var title string
	var desc = "no-desc"
	var image = ""
	var roomType string
	if title0, ok := input["title"].(string); ok {
		title = title0
	} else {
		xl.Infof("miss title in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if desc0, ok := input["desc"].(string); ok {
		desc = desc0
	}
	if image0, ok := input["image"].(string); ok {
		image = image0
	}
	if roomType0, ok := input["type"].(string); ok {
		roomType = roomType0
	} else {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseUserDo, err := b.baseUserDao.Select(xl, userId)
	if err != nil {
		xl.Errorf("select base_user fail with userId: %s", userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// 使用用户头像作为封面
	if image == "" {
		image = baseUserDo.Avatar
	}
	baseRoomDo := &model.BaseRoomDo{
		Title:   title,
		Image:   image,
		Desc:    desc,
		Status:  model.BaseRoomCreated,
		Creator: userId,
		Type:    roomType,
	}
	attrs := make([]model.BaseEntryDo, 0, 1)
	if attrs0, ok := input["attrs"].([]interface{}); ok {
		for _, val0 := range attrs0 {
			val := val0.(map[string]interface{})
			entry := model.BaseEntryDo{
				Key:    val["key"].(string),
				Value:  val["value"],
				Status: model.BaseEntryAvailable,
			}
			attrs = append(attrs, entry)
		}
	}
	params := make([]model.BaseEntryDo, 0, 1)
	if params0, ok := input["params"].([]interface{}); ok {
		for _, val0 := range params0 {
			val := val0.(map[string]interface{})
			entry := model.BaseEntryDo{
				Key:    val["key"].(string),
				Value:  val["value"],
				Status: model.BaseEntryAvailable,
			}
			params = append(params, entry)
		}
	}
	// 上面一堆都是参数解析
	var invitationCode string
	for {
		invitationCode = utils.GenerateID()[0:6]
		tmp, _ := b.baseRoomDao.SelectByInvitationCode(xl, invitationCode)
		if tmp == nil {
			break
		}
	}
	baseRoomDo.BaseRoomAttrs = attrs
	baseRoomDo.BaseRoomParams = params
	baseRoomDo.InvitationCode = invitationCode
	baseRoomDo.BaseRoomParams = append(baseRoomDo.BaseRoomParams, model.BaseEntryDo{
		Key:    "invitationCode",
		Value:  invitationCode,
		Status: model.BaseEntryAvailable,
	})
	// 创建七牛IM群ID
	qiniuImGroupId, err := b.appConfigService.GetGroupId(xl, baseRoomDo.Id)
	if err != nil {
		xl.Errorf("failed to create qiniu im group, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseRoomDo.QiniuIMGroupId = qiniuImGroupId
	_, err = b.baseRoomDao.Insert(xl, baseRoomDo)
	if err != nil {
		xl.Errorf("insert base_room fail with roomId: %s and userId: %s", baseRoomDo.Id, userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	switch roomType {
	// 麦位数固定的情况
	case model.BaseTypeKtv, model.BaseTypeMovie:
		micNumber := 0
		if roomType == model.BaseTypeKtv {
			micNumber = DefaultKtvMicNumber
		} else if roomType == model.BaseTypeMovie {
			micNumber = DefaultMovieMicNumber
		}
		for i := 1; i < micNumber; i += 1 {
			mic := model.BaseMicDo{
				Name:          fmt.Sprintf("%s-%04d", baseRoomDo.Id, i),
				Status:        model.BaseMicAvailable,
				Type:          model.BaseMicTypeSecondary,
				BaseMicAttrs:  make([]model.BaseEntryDo, 0, 1),
				BaseMicParams: make([]model.BaseEntryDo, 0, 1),
			}
			_, _ = b.baseMicDao.InsertBaseMic(xl, &mic)
			roomMic := model.BaseRoomMicDo{
				RoomId: baseRoomDo.Id,
				MicId:  mic.Id,
				Index:  i,
				Status: model.BaseRoomMicUnused,
			}
			_, _ = b.baseRoomMicDao.Insert(xl, &roomMic)
		}
		// 指定主麦
		mic := model.BaseMicDo{
			Name:          fmt.Sprintf("%s-%04d", baseRoomDo.Id, 0),
			Status:        model.BaseMicAvailable,
			Type:          model.BaseMicTypeMain,
			BaseMicAttrs:  make([]model.BaseEntryDo, 0, 1),
			BaseMicParams: make([]model.BaseEntryDo, 0, 1),
		}
		_, _ = b.baseMicDao.InsertBaseMic(xl, &mic)
		roomMic := model.BaseRoomMicDo{
			RoomId: baseRoomDo.Id,
			MicId:  mic.Id,
			Index:  0,
			Status: model.BaseRoomMicUnused,
		}
		_, _ = b.baseRoomMicDao.Insert(xl, &roomMic)
	// 麦位数按需增长，但是需要设定一个主麦
	case model.BaseTypeClassroom, model.BaseTypeShow, model.BaseTypeExam, model.BaseTypeVoiceChat:
		mic := model.BaseMicDo{
			Name:          fmt.Sprintf("%s-%04d", baseRoomDo.Id, 0),
			Status:        model.BaseMicAvailable,
			Type:          model.BaseMicTypeMain,
			BaseMicAttrs:  make([]model.BaseEntryDo, 0, 1),
			BaseMicParams: make([]model.BaseEntryDo, 0, 1),
		}
		_, _ = b.baseMicDao.InsertBaseMic(xl, &mic)
		roomMic := model.BaseRoomMicDo{
			RoomId: baseRoomDo.Id,
			MicId:  mic.Id,
			Index:  0,
			Status: model.BaseRoomMicUnused,
		}
		_, _ = b.baseRoomMicDao.Insert(xl, &roomMic)
	}
	// 构建返回值
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			RoomInfo *model.RoomInformation `json:"roomInfo"`
			UserInfo *model.BaseUserDo      `json:"userInfo"`
			RtcInfo  *model.RtcInfoResponse `json:"rtcInfo"`
		}{
			RoomInfo: &model.RoomInformation{
				BaseRoomDo: *baseRoomDo,
				TotalUsers: 0,
			},
			UserInfo: baseUserDo,
			RtcInfo: &model.RtcInfoResponse{
				RoomToken:   b.rtcService.GenerateRTCRoomToken(baseRoomDo.Id, userId, ADMIN),
				PublishUrl:  b.rtcService.StreamPubURL(baseRoomDo.Id),
				RtmpPlayUrl: b.rtcService.StreamRtmpPlayURL(baseRoomDo.Id),
				FlvPlayUrl:  b.rtcService.StreamFlvPlayURL(baseRoomDo.Id),
				HlsPlayUrl:  b.rtcService.StreamHlsPlayURL(baseRoomDo.Id),
			},
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseRoomApiHandler) JoinRoom(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	roomId := ""
	invitationCode := ""
	var roomType string
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	}
	if roomType0, ok := input["type"].(string); ok {
		roomType = roomType0
	} else {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	params := make([]model.BaseEntryDo, 0, 1)
	if params0, ok := input["params"].([]interface{}); ok {
		for _, val0 := range params0 {
			val := val0.(map[string]interface{})
			entry := model.BaseEntryDo{
				Key:    val["key"].(string),
				Value:  val["value"],
				Status: model.BaseEntryAvailable,
			}
			params = append(params, entry)
		}
	}
	role := "no-role"
	for _, entry := range params {
		if entry.Key == "role" {
			role = entry.Value.(string)
		}
		if entry.Key == "invitationCode" {
			invitationCode = entry.Value.(string)
		}
	}
	if roomId == "" && invitationCode == "" {
		xl.Infof("miss roomId/invitationCode in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	var baseRoomDo *model.BaseRoomDo
	if roomId != "" {
		baseRoomDo, err = b.baseRoomDao.Select(xl, roomId)
	} else {
		baseRoomDo, err = b.baseRoomDao.SelectByInvitationCode(xl, invitationCode)
	}
	if err != nil {
		if err == mgo.ErrNotFound {
			resp := &model.Response{
				Code:    model.ResponseErrorBadRequest,
				Message: "room not exist.",
				Data: struct {
				}{},
				RequestID: requestId,
			}
			context.JSON(http.StatusOK, resp)
		} else {
			xl.Errorf("select base_room fail with roomId: %s", baseRoomDo.Id)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
			context.JSON(http.StatusOK, resp)
		}
		return
	}
	classType := -1
	if roomType == model.BaseTypeClassroom && len(baseRoomDo.BaseRoomParams) != 0 {
		for _, val := range baseRoomDo.BaseRoomParams {
			if val.Key == "classType" {
				classType = int(val.Value.(float64))
				break
			}
		}
	}
	canNot := false
	// 是小班课
	if classType == 2 {
		l, _ := b.baseRoomUserDao.ListByRoomId(xl, baseRoomDo.Id)
		// 且人数已经到了两人
		if len(l) >= 2 && l[0].UserId != userId && l[1].UserId != userId {
			canNot = true
		}
	}
	if !canNot {
		baseRoomUserDo, err := b.baseRoomUserDao.SelectByRoomIdUserId(xl, baseRoomDo.Id, userId)
		if err == mgo.ErrNotFound {
			baseRoomUserDo = &model.BaseRoomUserDo{
				RoomId:            baseRoomDo.Id,
				UserId:            userId,
				UserRole:          role,
				Status:            model.BaseRoomUserJoin,
				LastHeartbeatTime: time.Now(),
			}
			_, err = b.baseRoomUserDao.Insert(xl, baseRoomUserDo)
			if err != nil {
				xl.Errorf("insert base_room_user fail with userId: %s and roomId: %s", userId, baseRoomDo.Id)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
				context.JSON(http.StatusOK, resp)
				return
			}
		} else {
			baseRoomUserDo.LastHeartbeatTime = time.Now()
			_ = b.baseRoomUserDao.Update(xl, baseRoomUserDo)
		}
	} else {
		resp := &model.Response{
			Code:    model.ResponseErrorTooManyPeople,
			Message: "too many people.",
			Data: struct {
			}{},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	_ = b.baseRoomDao.Update(xl, baseRoomDo)
	baseUserDo, err := b.baseUserDao.Select(xl, userId)
	if err != nil {
		xl.Errorf("select base_user fail with userId: %s", userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseRoomUserDos, err := b.baseRoomUserDao.ListByRoomId(xl, baseRoomDo.Id)
	if err != nil {
		xl.Errorf("select base_room_user fail with userId: %s and roomId: %s", userId, baseRoomDo.Id)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseUserDos := make([]model.BaseUserDo, 0, 1)
	for _, baseRoomUser := range baseRoomUserDos {
		tmp, _ := b.baseUserDao.Select(xl, baseRoomUser.UserId)
		baseUserDos = append(baseUserDos, *tmp)
	}
	list, _ := b.baseRoomUserDao.ListByRoomId(xl, baseRoomDo.Id)
	// 赋值IM群ID
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			RoomInfo         *model.RoomInformation  `json:"roomInfo"`
			UserInfo         *model.BaseUserDo       `json:"userInfo"`
			RtcInfo          *model.RtcInfoResponse  `json:"rtcInfo"`
			AllUserList      []model.BaseUserDo      `json:"allUserList"`
			ImConfigResponse *model.ImConfigResponse `json:"imConfig"`
		}{
			RoomInfo: &model.RoomInformation{
				BaseRoomDo: *baseRoomDo,
				TotalUsers: len(list),
			},
			UserInfo: baseUserDo,
			RtcInfo: &model.RtcInfoResponse{
				RoomToken:   b.rtcService.GenerateRTCRoomToken(baseRoomDo.Id, userId, ADMIN),
				PublishUrl:  b.rtcService.StreamPubURL(baseRoomDo.Id),
				RtmpPlayUrl: b.rtcService.StreamRtmpPlayURL(baseRoomDo.Id),
				FlvPlayUrl:  b.rtcService.StreamFlvPlayURL(baseRoomDo.Id),
				HlsPlayUrl:  b.rtcService.StreamHlsPlayURL(baseRoomDo.Id),
			},
			AllUserList: baseUserDos,
			ImConfigResponse: &model.ImConfigResponse{
				IMGroupId: baseRoomDo.QiniuIMGroupId,
				Type:      int(model.ImTypeQiniu),
			},
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseRoomApiHandler) LeaveRoom(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	var roomId string
	var roomType string
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		xl.Infof("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if roomType0, ok := input["type"].(string); ok {
		roomType = roomType0
	} else {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// 根据业务类型特例化
	if roomType == "" {
	}
	room, err := b.baseRoomDao.Select(xl, roomId)
	if err != nil {
		if err == mgo.ErrNotFound {
			resp := &model.Response{
				Code:    model.ResponseErrorBadRequest,
				Message: "room not exist.",
				Data: struct {
				}{},
				RequestID: requestId,
			}
			context.JSON(http.StatusOK, resp)
		} else {
			xl.Errorf("select base_room fail with roomId:[%s]", userId, roomId)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
			context.JSON(http.StatusOK, resp)
		}
		return
	}
	if room != nil {
		// 主持人离开
		if room.Creator == userId {
			xl.Infof("room creator leave, and the room will be destroyed.")
			room.Status = model.BaseRoomDestroyed
			_ = b.baseRoomDao.Update(xl, room)
			_ = b.appConfigService.DestroyGroupChat(xl, room.QiniuIMGroupId)
			roomUsers, _ := b.baseRoomUserDao.ListByRoomId(xl, roomId)
			for _, val := range roomUsers {
				b.leaveRoom(&val)
			}
		} else {
			roomUser, _ := b.baseRoomUserDao.SelectByRoomIdUserId(xl, roomId, userId)
			if roomUser != nil {
				b.leaveRoom(roomUser)
			}
		}
		if roomType == model.BaseTypeKtv || roomType == model.BaseTypeMovie {
			roomUserSong, _ := b.roomUserSongDao.SelectByRoomIdUserId(xl, roomId, userId)
			if roomUserSong != nil {
				roomUserSong.Status = model.RoomUserSongUnavailable
				_ = b.roomUserSongDao.Update(xl, roomUserSong)
			}
			roomUserMovie, _ := b.roomUserMovieDao.SelectByRoomIdUserId(xl, roomId, userId)
			if roomUserMovie != nil {
				roomUserMovie.Status = model.RoomUserMovieUnavailable
				_ = b.roomUserMovieDao.Update(xl, roomUserMovie)
			}
		}
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      true,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseRoomApiHandler) leaveRoom(roomUser *model.BaseRoomUserDo) {
	userMic, _ := b.baseUserMicDao.SelectByRoomIdUserId(nil, roomUser.RoomId, roomUser.UserId)
	if userMic != nil {
		userMic.Status = model.BaseUserMicNonHold
		_ = b.baseUserMicDao.Update(nil, userMic)
		roomMic, _ := b.baseRoomMicDao.Select(nil, userMic.RoomId, userMic.MicId)
		if roomMic != nil {
			roomMic.Status = model.BaseRoomMicUnused
			_ = b.baseRoomMicDao.Update(nil, roomMic)
		}
	}
	roomUser.Status = model.BaseRoomUserTimeout
	_ = b.baseRoomUserDao.Update(nil, roomUser)
}

func (b *BaseRoomApiHandler) ListRooms(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	roomType := context.DefaultQuery("type", "")
	if roomType == "" {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseRoomDos, total, count, err := b.baseRoomDao.ListByRoomType(xl, roomType, pageNum, pageSize)
	if err != nil {
		xl.Errorf("select base_room all fail with userId: %s", userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	flag := false
	if pageNum*pageSize >= total {
		flag = true
	}
	list := make([]model.RoomInformation, 0, len(baseRoomDos))
	for _, val := range baseRoomDos {
		l, _ := b.baseRoomUserDao.ListByRoomId(xl, val.Id)
		list = append(list, model.RoomInformation{
			BaseRoomDo: val,
			TotalUsers: len(l),
		})
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.ListRooms{
			List:           list,
			Total:          total,
			NextId:         "",
			Cnt:            count,
			CurrentPageNum: pageNum,
			NextPageNum:    pageNum + 1,
			PageSize:       pageSize,
			EndPage:        flag,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseRoomApiHandler) RoomInfo(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	roomId := context.DefaultQuery("roomId", "")
	if roomId == "" {
		xl.Infof("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	roomType := context.DefaultQuery("type", "")
	if roomType == "" {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// 以上都是参数处理
	b.sync(roomId)
	baseRoomDo, err := b.baseRoomDao.Select(xl, roomId)
	if err != nil {
		xl.Errorf("select base_room fail with roomId: %s and userId: %s", roomId, userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseUser, err := b.baseUserDao.Select(xl, userId)
	if err != nil {
		xl.Errorf("select base_user fail with userId: %s", userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseRoomUserDos, err := b.baseRoomUserDao.ListByRoomId(xl, roomId)
	if err != nil {
		xl.Errorf("select base_room_user all fail with roomId: %s", roomId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	baseUserDos := make([]model.BaseUserDo, 0, len(baseRoomUserDos))
	for _, val := range baseRoomUserDos {
		tmp, _ := b.baseUserDao.Select(xl, val.UserId)
		baseUserDos = append(baseUserDos, *tmp)
	}
	userMics, err := b.baseUserMicDao.ListByRoomId(xl, roomId)
	if err != nil {
		xl.Errorf("select base_room_mic all fail with roomId: %s", roomId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	mics := make([]model.MicInfo, 0, len(userMics))
	for _, val := range userMics {
		baseMicDo, _ := b.baseMicDao.Select(xl, val.MicId)
		micInfo := model.MicInfo{
			Uid:           val.UserId,
			UserExtension: val.UserExtension,
		}
		if baseMicDo == nil {
			micInfo.Attrs = make([]model.BaseEntryDo, 0, 1)
			micInfo.Params = make([]model.BaseEntryDo, 0, 1)
		} else {
			micInfo.Attrs = baseMicDo.BaseMicAttrs
			micInfo.Params = baseMicDo.BaseMicParams
		}
		mics = append(mics, micInfo)
	}
	l, _ := b.baseRoomUserDao.ListByRoomId(xl, baseRoomDo.Id)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.RoomInfoAll{
			UserInfo: baseUser,
			RoomInfo: &model.RoomInformation{
				BaseRoomDo: *baseRoomDo,
				TotalUsers: len(l),
			},
			RtcInfo: &model.RtcInfoResponse{
				RoomToken:   b.rtcService.GenerateRTCRoomToken(baseRoomDo.Id, userId, ADMIN),
				PublishUrl:  b.rtcService.StreamPubURL(roomId),
				RtmpPlayUrl: b.rtcService.StreamRtmpPlayURL(roomId),
				FlvPlayUrl:  b.rtcService.StreamFlvPlayURL(roomId),
				HlsPlayUrl:  b.rtcService.StreamHlsPlayURL(roomId),
			},
			Mics:        mics,
			AllUserList: baseUserDos,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseRoomApiHandler) UpdateRoomInfo(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	var roomId string
	var roomType string
	var entries []model.BaseEntryDo
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		xl.Infof("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if roomType0, ok := input["type"].(string); ok {
		roomType = roomType0
	} else {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// 特例化处理
	if roomType == "" {
	}
	if entries0, ok := input["attrs"].([]interface{}); ok {
		entries = make([]model.BaseEntryDo, 0, len(entries0))
		for _, val0 := range entries0 {
			val := val0.(map[string]interface{})
			entry := model.BaseEntryDo{
				Key:    val["key"].(string),
				Value:  val["value"],
				Status: model.BaseEntryAvailable,
			}
			entries = append(entries, entry)
		}
	}
	baseRoomDo, err := b.baseRoomDao.Select(xl, roomId)
	baseRoomDo.BaseRoomAttrs = entries
	err = b.baseRoomDao.Update(xl, baseRoomDo)
	if err != nil {
		return
	}
	baseUserDo, err := b.baseUserDao.Select(xl, userId)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			RoomInfo *model.BaseRoomDo `json:"roomInfo"`
			UserInfo *model.BaseUserDo `json:"userInfo"`
		}{
			RoomInfo: baseRoomDo,
			UserInfo: baseUserDo,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseRoomApiHandler) RoomInfoAttr(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	roomId := context.DefaultQuery("roomId", "")
	if roomId == "" {
		xl.Infof("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	roomType := context.DefaultQuery("type", "")
	if roomType == "" {
		xl.Infof("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	attrKey := context.DefaultQuery("attrKey", "")
	baseRoomDo, err := b.baseRoomDao.Select(xl, roomId)
	if err != nil {
		xl.Errorf("select base_room fail with roomId: %s", roomId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if attrKey == "" {
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				Attrs []model.BaseEntryDo `json:"attrs"`
			}{
				Attrs: baseRoomDo.BaseRoomAttrs,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	} else {
		var val interface{}
		var status int
		for _, v := range baseRoomDo.BaseRoomAttrs {
			if v.Key == attrKey {
				val = v.Value
				status = v.Status
			}
		}
		tmp := make([]model.BaseEntryDo, 0, 1)
		tmp = append(tmp, model.BaseEntryDo{
			Key:    attrKey,
			Value:  val,
			Status: status,
		})
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				Attrs []model.BaseEntryDo `json:"attrs"`
			}{
				Attrs: tmp,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	}
}

func (b *BaseRoomApiHandler) TruncateGroupChat(context *gin.Context) {
	force, _ := b.baseRoomDao.ListAllForce(b.xl)
	for _, val := range force {
		if val.QiniuIMGroupId == 0 {
			continue
		}
		_ = b.appConfigService.DestroyGroupChat(b.xl, val.QiniuIMGroupId)
	}
}

func (b *BaseRoomApiHandler) ListUser(context *gin.Context) {
	roomId := context.Param("roomId")
	list, _ := b.rtcService.ListUser(roomId)
	context.JSON(200, list)
}

func (b *BaseRoomApiHandler) sync(roomId string) {
	list, _ := b.rtcService.ListUser(roomId)
	set := make(map[string]struct{})
	for _, val := range list {
		set[val] = struct{}{}
	}
	userMicDos, _ := b.baseUserMicDao.ListByRoomId(nil, roomId)
	for _, val := range userMicDos {
		if _, ok := set[val.UserId]; !ok {
			roomMicDo, _ := b.baseRoomMicDao.Select(nil, val.RoomId, val.MicId)
			if roomMicDo != nil {
				roomMicDo.Status = model.BaseRoomMicUnused
				_ = b.baseRoomMicDao.Update(nil, roomMicDo)
			}
			val.Status = model.BaseUserMicNonHold
			_ = b.baseUserMicDao.Update(nil, &val)
		}
	}
}
