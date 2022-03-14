package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/form"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db"
	"gopkg.in/mgo.v2"
	"net/http"
)

// short for interactive entertainment 互动娱乐

type IEHandler interface {
	Create(c *gin.Context)
	Join(c *gin.Context)
	Leave(c *gin.Context)
	//Cancel(c *gin.Context)
	End(c *gin.Context) // 房主退房
	Share(c *gin.Context)
	Update(c *gin.Context)
	List(c *gin.Context)

	RegisterRoute(group *gin.RouterGroup)
}

type IEHandlerImpl struct {
	ieService db.IEService
	xl        *xlog.Logger
}

func NewIEHandler(ieService db.IEService) IEHandler {
	return NewIEHandlerImpl(ieService)
}

func NewIEHandlerImpl(ieService db.IEService) *IEHandlerImpl {
	xl := xlog.New("ie service")
	return &IEHandlerImpl{ieService: ieService, xl: xl}
}

// Create ie/create
func (I IEHandlerImpl) Create(c *gin.Context) {
	//userId:=c.MustGet(model.UserIDContextKey)
	user := c.MustGet(model.UserContextKey).(model.AccountDo)
	args := form.IECreateForm{}
	err := c.Bind(&args)
	if err != nil {
		I.logger(c).Errorf("err bind form:%v", err)
		return
	}
	err = args.Validate()
	if err != nil {
		I.logger(c).Errorf("err validate form:%v form:%v", err, args)
		c.JSON(http.StatusOK, model.NewResponseErrorValidation(err))
		return
	}
	room_extra := map[string]interface{}{"roomAvatar": user.Avatar}
	err = I.ieService.CreateRoom(user, utils.GenerateID(), args.Map().Merge(room_extra))
	if err != nil {
		I.logger(c).Errorf("err create room:%v", err)
		c.JSON(http.StatusOK, model.NewResponseErrorNoSuchRoom())
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse("房间创建成功"))
}

// Join ie/:roomId/join
// 需返回麦序上的人 + 房间内的其他人 重点麦序
func (I IEHandlerImpl) Join(c *gin.Context) {
	roomId := c.Param("roomId")
	user := c.MustGet(model.UserContextKey).(model.AccountDo)
	extra, err := I.ieService.EnterRoom(user, roomId)
	if err != nil {
		switch {
		case err == mgo.ErrNotFound:
			I.logger(c).Errorf("err enter room err:%v", err)
			c.JSON(http.StatusOK, model.NewResponseErrorNoSuchRoom())
			return
		case err != nil && err != mgo.ErrNotFound:
			I.logger(c).Errorf("err enter room err:%v", err)
			c.JSON(http.StatusOK, model.NewResponseErrorExternalService())
			return
		}
	}
	creator, err := I.ieService.GetCreator(roomId)
	if err != nil {
		switch {
		case err == mgo.ErrNotFound:
			I.logger(c).Errorf("err enter room err:%v", err)
			c.JSON(http.StatusOK, model.NewResponseErrorNoSuchUser())
			return
		case err != nil && err != mgo.ErrNotFound:
			I.logger(c).Errorf("err enter room err:%v", err)
			c.JSON(http.StatusOK, model.NewResponseErrorExternalService())
			return
		}
	}
	m := model.IERoomResponse{
		Title:      extra.Title,
		Notice:     extra.Notice,
		RoomAvatar: extra.RoomAvatar,
		RoomID:     extra.RoomID,
	}
	userInfo := model.UserInfoResponse{
		ID:       creator.ID,
		Nickname: creator.Nickname,
		Avatar:   creator.Avatar,
		Phone:    creator.Phone,
		Profile:  model.DefaultAccountProfile,
	}
	res := model.JoinIERoomResponse{
		Room:       m,
		Creator:    userInfo,
		RoomToken:  "mock room Token",
		PublishURL: "mock publish url",
		PullURL:    "mock pull url",
		ImConfig:   model.ImConfigResponse{},
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse(res))
}

// Leave ie/:roomId/leave
func (I IEHandlerImpl) Leave(c *gin.Context) {
	roomId := c.Param("roomId") // TODO add to const
	user := c.MustGet(model.UserContextKey).(model.AccountDo)
	if err := I.ieService.LeaveRoom(user, roomId); err != nil {
		I.logger(c).Errorf("error leave room err:%v", err)
		c.JSON(http.StatusOK, model.NewResponseErrorNoSuchRoom())
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse("离开成功"))
}

// End ie/:roomId/end
func (I IEHandlerImpl) End(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")

}

//
func (I IEHandlerImpl) Share(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

func (I IEHandlerImpl) Update(c *gin.Context) {
	user := c.MustGet(model.UserContextKey).(model.AccountDo)
	roomId := c.Param("roomId")
	args := form.IEUpadteForm{}
	err := c.Bind(&args)
	if err != nil {
		I.logger(c).Errorf("err bind form:%v", err)
		return
	}
	err = args.Validate()
	if err != nil {
		I.logger(c).Errorf("err validate form:%v form:%v", err, args)
		c.JSON(http.StatusOK, model.NewResponseErrorValidation(err))
		return
	}
	err = I.ieService.UpdateRoomNoticeAndTitle(user, roomId, args.Notice, args.Title)
	if err != nil {
		I.logger(c).Errorf("err create room:%v", err)
		c.JSON(http.StatusOK, model.NewResponseErrorNoSuchRoom())
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse("房间修改成功"))
}

// List return List room
// may not need login
func (I IEHandlerImpl) List(c *gin.Context) {
	roomId := c.Param("roomId") // TODO: add to const
	pageSize := c.GetInt("pageSize")
	pageNum := c.GetInt("pageNum")
	data, total, next, err := I.ieService.ListRoom(roomId, pageNum, pageSize)
	if err != nil {
		I.logger(c).Errorf("error get list response err:%v", err)
		c.JSON(http.StatusOK, model.NewResponseErrorExternalService())
	}
	val := data.([]model.FlattenMap)
	listData := make([]interface{}, 0)
	for _, item := range val {
		listData = append(listData, item)
	}
	page := model.Pagination{
		Total:          total,
		CurrentPageNum: next - 1,
		NextPageNum:    next,
		PageSize:       pageSize,
		EndPage:        pageNum == next,
		List:           listData,
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse(page))
}

func (I IEHandlerImpl) RegisterRoute(group *gin.RouterGroup) {
	group.POST("create", I.Create)
	group.POST(":roomId/update", I.Update)
	group.GET("list", I.List)
	group.POST(":roomId/leave", I.Leave)
	group.POST(":roomId/join", I.Join)
}

// logger retrieve logger from request,use handler 's logger if none in req
func (i IEHandlerImpl) logger(c *gin.Context) *xlog.Logger {
	logger := i.xl
	val, ok := c.Get(model.XLogKey)
	if ok {
		logger = val.(*xlog.Logger)
	}
	return logger
}
