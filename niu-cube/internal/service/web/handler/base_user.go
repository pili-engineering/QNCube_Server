package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"net/http"
	"time"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/dao"
)

type BaseUserApi interface {
	Heartbeat(context *gin.Context)

	UpdateUserInfo(context *gin.Context)
}

type BaseUserApiHandler struct {
	baseUserDao     dao.BaseUserDaoInterface
	baseRoomUserDao dao.BaseRoomUserDaoInterface
}

func NewBaseUserApiHandler(xl *xlog.Logger, conf *utils.Config) *BaseUserApiHandler {
	baseRoomUserDao, err := dao.NewBaseRoomUserDaoService(xl, conf.Mongo)
	if err != nil {
		xl.Error("create BaseRoomUserDaoService failed.")
		return nil
	}
	baseUserDao, err := dao.NewBaseUserDaoService(xl, conf.Mongo)
	if err != nil {
		xl.Error("create BaseUserDaoService failed.")
		return nil
	}
	return &BaseUserApiHandler{
		baseUserDao,
		baseRoomUserDao,
	}
}

func (b *BaseUserApiHandler) Heartbeat(context *gin.Context) {
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
	roomUser, err := b.baseRoomUserDao.SelectByRoomIdUserId(xl, roomId, userId)
	if err != nil {
		// 用户已下线
		if err == mgo.ErrNotFound {
			xl.Infof("user:[%s] already logout.", userId)
		} else {
			xl.Errorf("select base_room_user all fail with roomId:[%s], userId:[%s]", roomId, userId)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
			context.JSON(http.StatusOK, resp)
			return
		}
	}
	if roomUser != nil {
		roomUser.LastHeartbeatTime = time.Now()
		err = b.baseRoomUserDao.Update(xl, roomUser)
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Interval uint64 `json:"interval"`
		}{
			Interval: 30000,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (b *BaseUserApiHandler) UpdateUserInfo(context *gin.Context) {
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
	baseUser, _ := b.baseUserDao.Select(xl, userId)
	if baseUser == nil {
		xl.Errorf("用户不存在")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if tmp, ok := input["name"].(string); ok {
		baseUser.Name = tmp
	}
	if tmp, ok := input["nickname"].(string); ok {
		baseUser.Nickname = tmp
	}
	if tmp, ok := input["avatar"].(string); ok {
		baseUser.Avatar = tmp
	}
	if tmp, ok := input["profile"].(string); ok {
		baseUser.Profile = tmp
	}
	_ = b.baseUserDao.Update(xl, baseUser)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
		}{},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}
