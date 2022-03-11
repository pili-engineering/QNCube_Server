package handler

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	dao2 "github.com/solutions/niu-cube/internal/service/dao"
)

const (
	DefaultKtvMicNumber   int = 6
	DefaultMovieMicNumber int = 2
)

type BaseMicApi interface {
	UpMic(context *gin.Context)

	DownMic(context *gin.Context)

	UpdateMicAttrs(context *gin.Context)

	MicInfo(context *gin.Context)

	MicAttrs(context *gin.Context)
}

type BaseMicApiHandler struct {
	baseMicDao     dao2.BaseMicDaoInterface
	baseRoomDao    dao2.BaseRoomDaoInterface
	baseUserMicDao dao2.BaseUserMicDaoInterface
	baseRoomMicDao dao2.BaseRoomMicDaoInterface
	rtcService     *cloud.RTCService
}

func NewBaseMicApiHandler(xl *xlog.Logger, conf *utils.Config) *BaseMicApiHandler {
	baseMicDao, err := dao2.NewBaseMicDaoService(xl, conf.Mongo)
	if err != nil {
		xl.Error("create BaseMicDaoService failed.")
		return nil
	}
	baseRoomDao, err := dao2.NewBaseRoomDaoService(xl, conf.Mongo)
	if err != nil {
		xl.Error("create BaseRoomDaoService failed.")
		return nil
	}
	baseUserMicDao, err := dao2.NewBaseUserMicDaoService(xl, conf.Mongo)
	if err != nil {
		xl.Error("create BaseUserMicDaoService failed.")
		return nil
	}
	baseRoomMicDao, err := dao2.NewBaseRoomMicDaoService(xl, conf.Mongo)
	if err != nil {
		xl.Error("create BaseRoomMicDaoService failed.")
		return nil
	}
	rtcService := cloud.NewRtcService(*conf)
	return &BaseMicApiHandler{
		baseMicDao,
		baseRoomDao,
		baseUserMicDao,
		baseRoomMicDao,
		rtcService,
	}
}

func (b *BaseMicApiHandler) UpMic(context *gin.Context) {
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
	var userExtension string
	attrs := make([]model.BaseEntryDo, 0, 1)
	params := make([]model.BaseEntryDo, 0, 1)
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
	if userExtension0, ok := input["userExtension"].(string); ok {
		userExtension = userExtension0
	} else {
	}
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
	if params0, ok := input["params"].([]interface{}); ok {
		for _, val0 := range params0 {
			val := val0.(map[string]interface{})
			entry := model.BaseEntryDo{
				Key:    val["key"].(string),
				Value:  val["value"],
				Status: 0,
			}
			params = append(params, entry)
		}
	}
	color.Blue("用户: %s 上 %s 的麦位", userId, roomId)
	// 以上都是参数处理
	b.sync(roomId)
	userMics, err := b.baseUserMicDao.ListByRoomId(xl, roomId)
	if err != nil {
		xl.Errorf("select base_user_mic all fail with userId: %s", userId)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	alreadyUpMic := false
	for _, val := range userMics {
		// 已上麦
		if val.UserId == userId && val.Status == model.BaseUserMicHold {
			alreadyUpMic = true
		}
	}
	if !alreadyUpMic {
		roomTmp, err := b.baseRoomDao.Select(xl, roomId)
		if err != nil {
			xl.Errorf("select base_room fail with roomId: %s", roomId)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
			context.JSON(http.StatusOK, resp)
			return
		}
		switch roomType {
		case model.BaseTypeKtv, model.BaseTypeMovie:
			var roomMic model.BaseRoomMicDo
			flag := false
			roomMics, err := b.baseRoomMicDao.ListByRoomId(xl, roomId)
			if err != nil {
				xl.Errorf("select base_room_mic all fail with roomId: %s", roomId)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
				context.JSON(http.StatusOK, resp)
				return
			}
			// 上主麦
			if roomTmp.Creator == userId {
				for _, val := range roomMics {
					if val.Status == model.BaseRoomMicUnused {
						mic, _ := b.baseMicDao.Select(xl, val.MicId)
						if mic.Type == model.BaseMicTypeMain {
							roomMic = val
							flag = true
							break
						}
					}
				}
				// 上副麦
			} else {
				for _, val := range roomMics {
					if val.Status == model.BaseRoomMicUnused {
						mic, _ := b.baseMicDao.Select(xl, val.MicId)
						if mic.Type == model.BaseMicTypeSecondary {
							roomMic = val
							flag = true
							break
						}
					}
				}
			}
			// 无麦可用
			if !flag {
				resp := &model.Response{
					Code:    int(model.ResponseStatusCodeSuccess),
					Message: string(model.ResponseStatusMessageSuccess),
					Data: struct {
						Mics []model.MicInfo
					}{
						Mics: make([]model.MicInfo, 0, 1),
					},
					RequestID: requestId,
				}
				context.JSON(http.StatusOK, resp)
				return
			} else {
				roomMic.Status = model.BaseRoomMicUsed
				err = b.baseRoomMicDao.Update(xl, &roomMic)
				if err != nil {
					// TODO
					return
				}
				userMic := model.BaseUserMicDo{
					RoomId:        roomId,
					UserId:        userId,
					MicId:         roomMic.MicId,
					Status:        model.BaseUserMicHold,
					UserExtension: userExtension,
				}
				_, err = b.baseUserMicDao.Insert(xl, &userMic)
				if err != nil {
					// TODO
					return
				}
				mic, _ := b.baseMicDao.Select(xl, roomMic.MicId)
				mic.BaseMicAttrs = attrs
				mic.BaseMicParams = params
				_ = b.baseMicDao.Update(xl, mic)
			}
		case model.BaseTypeClassroom, model.BaseTypeShow, model.BaseTypeExam, model.BaseTypeVoiceChat:
			if roomTmp.Creator == userId {
				var roomMic model.BaseRoomMicDo
				flag := false
				roomMics, _ := b.baseRoomMicDao.ListByRoomId(xl, roomId)
				for _, val := range roomMics {
					if val.Status == model.BaseRoomMicUnused {
						mic, _ := b.baseMicDao.Select(xl, val.MicId)
						if mic.Type == model.BaseMicTypeMain {
							roomMic = val
							flag = true
							break
						}
					}
				}
				// 无麦可用
				if !flag {
					resp := &model.Response{
						Code:    int(model.ResponseStatusCodeSuccess),
						Message: string(model.ResponseStatusMessageSuccess),
						Data: struct {
							Mics []model.MicInfo
						}{
							Mics: make([]model.MicInfo, 0, 1),
						},
						RequestID: requestId,
					}
					context.JSON(http.StatusOK, resp)
					return
				} else {
					roomMic.Status = model.BaseRoomMicUsed
					err = b.baseRoomMicDao.Update(xl, &roomMic)
					if err != nil {
						// TODO
						return
					}
					userMic := model.BaseUserMicDo{
						RoomId:        roomId,
						UserId:        userId,
						MicId:         roomMic.MicId,
						Status:        model.BaseUserMicHold,
						UserExtension: userExtension,
					}
					_, err = b.baseUserMicDao.Insert(xl, &userMic)
					if err != nil {
						// TODO
						return
					}
					mic, _ := b.baseMicDao.Select(xl, roomMic.MicId)
					mic.BaseMicAttrs = attrs
					mic.BaseMicParams = params
					_ = b.baseMicDao.Update(xl, mic)
				}
			} else {
				mic := model.BaseMicDo{
					Name:          roomId + "-" + fmt.Sprintf("%20d", time.Now().Unix()),
					Status:        0,
					Type:          model.BaseMicTypeSecondary,
					BaseMicAttrs:  attrs,
					BaseMicParams: params,
				}
				_, err := b.baseMicDao.InsertBaseMic(xl, &mic)
				if err != nil {
					return
				}
				userMic := model.BaseUserMicDo{
					RoomId:        roomId,
					UserId:        userId,
					MicId:         mic.Id,
					Status:        model.BaseUserMicHold,
					UserExtension: userExtension,
				}
				_, err = b.baseUserMicDao.Insert(xl, &userMic)
				if err != nil {
					// TODO
					return
				}
				roomMic := model.BaseRoomMicDo{
					RoomId: roomId,
					MicId:  mic.Id,
					Index:  -1,
					Status: model.BaseRoomMicUsed,
				}
				_, err = b.baseRoomMicDao.Insert(xl, &roomMic)
				if err != nil {
					xl.Errorf("insert base_room_mic fail with roomId: %s", roomId)
					responseErr := model.NewResponseErrorInternal()
					resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
					context.JSON(http.StatusOK, resp)
					return
				}
			}
		}
	}
	// 构建返回值
	userMics, err = b.baseUserMicDao.ListByRoomId(xl, roomId)
	if err != nil {
		xl.Errorf("select base_user_mic all fail with userId: %s", userId)
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
			if baseMicDo.BaseMicAttrs == nil || len(baseMicDo.BaseMicAttrs) == 0 {
				baseMicDo.BaseMicAttrs = make([]model.BaseEntryDo, 0, 1)
			}
			if baseMicDo.BaseMicParams == nil || len(baseMicDo.BaseMicParams) == 0 {
				baseMicDo.BaseMicParams = make([]model.BaseEntryDo, 0, 1)
			}
			micInfo.Attrs = baseMicDo.BaseMicAttrs
			micInfo.Params = baseMicDo.BaseMicParams
		}
		mics = append(mics, micInfo)
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Mics []model.MicInfo `json:"mics"`
		}{
			Mics: mics,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseMicApiHandler) DownMic(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	// userId := context.GetString(model.UserIDContextKey)
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
	var userId string
	var roomType string
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		xl.Info("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if micId0, ok := input["uid"].(string); ok {
		userId = micId0
	} else {
		xl.Info("miss userId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if roomType0, ok := input["type"].(string); ok {
		roomType = roomType0
	} else {
		xl.Info("miss roomType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// 以上都是参数处理
	b.sync(roomId)
	// 特例化处理
	if roomType == "" {
	}
	userMic, _ := b.baseUserMicDao.SelectByRoomIdUserId(xl, roomId, userId)
	if userMic != nil {
		roomMic, _ := b.baseRoomMicDao.Select(xl, roomId, userMic.MicId)
		userMic.Status = model.BaseUserMicNonHold
		_ = b.baseUserMicDao.Update(xl, userMic)
		roomMic.Status = model.BaseRoomMicUnused
		_ = b.baseRoomMicDao.Update(xl, roomMic)
	} else {
		xl.Error("未找到相关user_mic")
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      true,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseMicApiHandler) UpdateMicAttrs(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	// userId := context.GetString(model.UserIDContextKey)
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
	var userId string
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
	if micId0, ok := input["uid"].(string); ok {
		userId = micId0
	} else {
		xl.Infof("miss userId in body.")
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
	// 以上都是参数处理
	b.sync(roomId)
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
	v0, err := b.baseUserMicDao.SelectByRoomIdUserId(xl, roomId, userId)
	baseMicDo, _ := b.baseMicDao.Select(xl, v0.MicId)
	baseMicDo.BaseMicAttrs = entries
	err = b.baseMicDao.Update(xl, baseMicDo)
	if err != nil {
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
			}{},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      true,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseMicApiHandler) MicInfo(context *gin.Context) {
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
	// 特例化处理
	if roomType == "" {
	}
	color.Blue("用户 %s 获取 %s 的麦位", userId, roomId)
	// 以上都是参数处理
	b.sync(roomId)
	baseRoomDo, _ := b.baseRoomDao.Select(xl, roomId)
	userMics, _ := b.baseUserMicDao.ListByRoomId(xl, roomId)
	mics := make([]model.MicInfo, 0, len(userMics))
	for _, userMic := range userMics {
		mic, _ := b.baseMicDao.Select(xl, userMic.MicId)
		if mic.BaseMicAttrs == nil || len(mic.BaseMicAttrs) == 0 {
			mic.BaseMicAttrs = make([]model.BaseEntryDo, 0, 1)
		}
		if mic.BaseMicParams == nil || len(mic.BaseMicParams) == 0 {
			mic.BaseMicParams = make([]model.BaseEntryDo, 0, 1)
		}
		mics = append(mics, model.MicInfo{
			Uid:           userMic.UserId,
			UserExtension: userMic.UserExtension,
			Attrs:         mic.BaseMicAttrs,
			Params:        mic.BaseMicParams,
		})
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			RoomInfo *struct {
				RoomId string              `json:"roomId"`
				Attrs  []model.BaseEntryDo `json:"attrs"`
			} `json:"roomInfo"`
			Mics []model.MicInfo `json:"mics"`
		}{
			RoomInfo: &struct {
				RoomId string              `json:"roomId"`
				Attrs  []model.BaseEntryDo `json:"attrs"`
			}{
				RoomId: baseRoomDo.Id,
				Attrs:  baseRoomDo.BaseRoomAttrs,
			},
			Mics: mics,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseMicApiHandler) MicAttrs(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	// userId := context.GetString(model.UserIDContextKey)
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
	userId := context.DefaultQuery("uid", "")
	if userId == "" {
		xl.Infof("miss userId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	color.Blue("用户 %s 尝试获取 %s 里的麦位", userId, roomId)
	// 以上都是参数处理
	b.sync(roomId)
	attrKey := context.DefaultQuery("attrKey", "")
	if roomType == "" {
	}
	roomMic, _ := b.baseUserMicDao.SelectByRoomIdUserId(xl, roomId, userId)
	color.Blue("用户 %s 尝试获取 %s 的麦位 %s 的属性", userId, roomId, roomMic.MicId)
	baseMicDo, _ := b.baseMicDao.Select(xl, roomMic.MicId)
	if attrKey != "" {
		var value interface{}
		for _, entry := range baseMicDo.BaseMicAttrs {
			if entry.Key == attrKey {
				value = entry.Value
				break
			}
		}
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				Attrs []model.BaseEntryDo
			}{
				Attrs: []model.BaseEntryDo{{
					Key:    attrKey,
					Value:  value,
					Status: model.BaseEntryAvailable,
				}},
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	} else {
		if baseMicDo.BaseMicAttrs == nil || len(baseMicDo.BaseMicAttrs) == 0 {
			baseMicDo.BaseMicAttrs = make([]model.BaseEntryDo, 0, 1)
		}
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				Attrs []model.BaseEntryDo `json:"attrs"`
			}{
				Attrs: baseMicDo.BaseMicAttrs,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	}
}

func (b *BaseMicApiHandler) sync(roomId string) {
	list, _ := b.rtcService.ListUser(roomId)
	set := make(map[string]struct{})
	for _, val := range list {
		set[val] = struct{}{}
	}
	color.Yellow("开始同步麦位")
	userMicDos, _ := b.baseUserMicDao.ListByRoomId(nil, roomId)
	for _, val := range userMicDos {
		if _, ok := set[val.UserId]; !ok {
			roomMicDo, _ := b.baseRoomMicDao.Select(nil, val.RoomId, val.MicId)
			color.Red("房间 %s, 麦位 %s, 用户 %s 不一致", val.RoomId, val.MicId, val.UserId)
			if roomMicDo != nil {
				roomMicDo.Status = model.BaseRoomMicUnused
				_ = b.baseRoomMicDao.Update(nil, roomMicDo)
			}
			val.Status = model.BaseUserMicNonHold
			_ = b.baseUserMicDao.Update(nil, &val)
		}
	}
}
