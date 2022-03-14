package handler

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	form "github.com/solutions/niu-cube/internal/protodef/form"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	service "github.com/solutions/niu-cube/internal/service/db"
	"gopkg.in/mgo.v2"
	"net/http"
	"time"
)

type BoardHandlerApi struct {
	boardService     *service.BoardService
	interviewService *service.InterviewService
	rtcService       *cloud.RTCService
	xl               *xlog.Logger
}

// TODO
func NewBoardHandlerApi(config utils.Config) *BoardHandlerApi {
	var err error
	v := new(BoardHandlerApi)
	v.xl = xlog.New("board Handler")
	v.boardService = service.NewBoardService(v.xl, *config.Mongo)
	v.interviewService, err = service.NewInterviewService(*config.Mongo, v.xl)
	v.rtcService = cloud.NewRtcService(config)
	if err != nil {
		panic(err)
	}
	return v
}

func (v *BoardHandlerApi) CreateOrUpdateBoard(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	userId := c.MustGet(model.UserIDContextKey).(string)
	interviewId := c.Param("interviewId")
	var boardForm form.BoardCreateForm
	xl.Debugf("create or update board")
	err := c.Bind(&boardForm)
	if err != nil {
		xl.Errorf("form binding error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	err = boardForm.Validate()
	if err != nil {
		xl.Errorf("form valdiation error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	var board model.BoardDo
	board, err = v.boardService.GetOneByID(v.xl, interviewId)
	interview, _ := v.interviewService.GetInterviewByID(v.xl, interviewId)
	switch {
	case interview == nil:
		xl.Errorf("no such interview:%v", interviewId)
		respErr := model.NewResponseErrorNoSuchInterview()
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	case err == nil:
		// 已存在board 更新状态
		err := v.boardStateTransition(&board, boardForm.Cmd, userId, interviewId)
		if err != nil {
			xl.Errorf("board transit state error:%v", err)
			respErr := model.NewResponseErrorValidation(err)
			model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
			return
		}
	default:
		// 创建board
		board = model.BoardDo{
			InterviewID: interviewId,
			//StatusCode:    protocol.BoardStatusCodeOpen,
			Status:        model.BoardStatusCodeOpen.String(),
			CurrentUserID: userId,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
	}
	err = v.boardService.Upsert(xl, board)
	if err != nil {
		xl.Errorf("board service upsert error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	resp := model.NewSuccessResponse(board)
	c.JSON(http.StatusOK, resp)
}

func (v *BoardHandlerApi) GetBoard(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	interviewId := c.Param("interviewId")
	xl.Debugf("get board")
	if interviewId == "" {
		xl.Errorf("interviewId is missing")
		respErr := model.NewResponseErrorValidation(form.ErrInterviewIdNeeded)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	var res interface{}
	var err error
	res, err = v.boardService.GetOneByID(v.xl, interviewId)
	if err != nil {
		var respErr *model.ResponseError
		xl.Errorf("board service get error:%v", err)
		switch {
		case errors.Is(err, mgo.ErrNotFound):
			respErr = model.NewResponseErrorNoSuchBoard()
		default:
			respErr = model.NewResponseErrorValidation(err)
		}
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	resp := model.NewSuccessResponse(res)
	c.JSON(http.StatusOK, resp)
}

func (v *BoardHandlerApi) boardStateTransition(b *model.BoardDo, action model.BoardCmd, userId string, interviewId string) error {
	switch {
	case userId == b.CurrentUserID:
		v.xl.Debugf("permit cmd %v from owner %v", action, userId)
		switch action {
		case model.BoardCmdOpen:
			b.Status = model.BoardStatusOpen
			return nil
		case model.BoardCmdClose, model.BoardCmdReset:
			b.Status = model.BoardStatusClose
			return nil
		}
		return fmt.Errorf("未知命令")
	case userId != b.CurrentUserID:
		// reset命令，暂无权限校验
		if action == model.BoardCmdReset {
			b.CurrentUserID = ""
			b.Status = model.BoardStatusClose
			v.xl.Debugf("board %v reset by %v", b.ID, userId)
			return nil
		}
		// db中 不在线是可靠的，在线者中可能包含不在线者
		// rtc中 未知
		owner := b.CurrentUserID
		onlineExistence := v.rtcService.Online(interviewId, owner)
		dbExistence := v.interviewService.Online(v.xl, interviewId, owner)
		v.xl.Debugf("db existence:%v, online existence:%v", dbExistence, onlineExistence)
		switch {
		case dbExistence == true && onlineExistence == false:
			b.CurrentUserID = userId
			v.xl.Debugf("board %v current user change to %v", b.ID, b.CurrentUserID)
			return v.boardStateTransition(b, action, userId, interviewId)
		case dbExistence == true && onlineExistence == true:
			v.xl.Debugf("borad occupied by user %v", b.CurrentUserID)
			return form.ErrBoardLocked
		case dbExistence == false:
			b.CurrentUserID = userId
			v.xl.Debugf("board %v current user change to %v", b.ID, b.CurrentUserID)
			return v.boardStateTransition(b, action, userId, interviewId)
		}
		return fmt.Errorf("逻辑错误")
	}
	return fmt.Errorf("逻辑错误")
}
