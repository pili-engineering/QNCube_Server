package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/dao"
)

type KtvApi interface {
	ListSong(context *gin.Context)

	SongDemanded(context *gin.Context)

	SongOperation(context *gin.Context)

	SongInfo(context *gin.Context)

	AddSongs(context *gin.Context)

	UpdateSong(context *gin.Context)

	DeleteSong(context *gin.Context)

	ListAllSong(context *gin.Context)
}

type KtvApiHandler struct {
	songDao         dao.SongDaoInterface
	roomUserSongDao dao.RoomUserSongDaoInterface
}

func NewKtvApiHandler(xl *xlog.Logger, conf *utils.MongoConfig) *KtvApiHandler {
	songDao, err := dao.NewSongDaoService(xl, conf)
	if err != nil {
		xl.Error("create SongDaoService failed.")
		return nil
	}
	roomUserSongDao, err := dao.NewRoomUserSongDaoService(xl, conf)
	if err != nil {
		xl.Error("create RoomUserSongDaoService failed.")
	}
	return &KtvApiHandler{
		songDao,
		roomUserSongDao,
	}
}

func (k *KtvApiHandler) ListSong(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
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
	var pageNum int
	var pageSize int
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		roomId = "-1"
		xl.Infof("miss roomId: %s in the request parameters.", roomId)
	}
	if pageNum0, ok := input["pageNum"].(int); ok {
		pageNum = pageNum0
	} else {
		pageNum = 1
	}
	if pageSize0, ok := input["pageSize"].(int); ok {
		pageSize = pageSize0
	} else {
		pageSize = 10
	}
	list, total, count, err := k.songDao.ListAll(xl, pageNum, pageSize)
	flag := false
	if pageNum*pageSize >= total {
		flag = true
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Total          int            `json:"total"`
			NextId         string         `json:"nextId"`
			Cnt            int            `json:"cnt"`
			CurrentPageNum int            `json:"currentPageNum"`
			NextPageNum    int            `json:"nextPageNum"`
			PageSize       int            `json:"pageSize"`
			EndPage        bool           `json:"endPage"`
			List           []model.SongDo `json:"list"`
		}{
			Total:          total,
			NextId:         "",
			Cnt:            count,
			CurrentPageNum: pageNum,
			NextPageNum:    pageNum + 1,
			PageSize:       pageSize,
			EndPage:        flag,
			List:           list,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) SongDemanded(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
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
	var pageNum int
	var pageSize int
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		roomId = "-1"
		xl.Infof("miss roomId: %s in the request parameters.", roomId)
	}
	if pageNum0, ok := input["pageNum"].(int); ok {
		pageNum = pageNum0
	} else {
		pageNum = 1
	}
	if pageSize0, ok := input["pageSize"].(int); ok {
		pageSize = pageSize0
	} else {
		pageSize = 10
	}
	list, total, count, err := k.roomUserSongDao.ListByRoomId(xl, roomId, pageNum, pageSize)
	flag := false
	if pageNum*pageSize >= total {
		flag = true
	}
	type TmpResponse struct {
		model.SongDo
		Demander string `json:"demander"`
	}
	l := make([]TmpResponse, 0, count)
	for _, val := range list {
		song, _ := k.songDao.Select(xl, val.SongId)
		l = append(l, TmpResponse{
			SongDo:   *song,
			Demander: val.UserId,
		})
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Total          int           `json:"total"`
			NextId         string        `json:"nextId"`
			Cnt            int           `json:"cnt"`
			CurrentPageNum int           `json:"currentPageNum"`
			NextPageNum    int           `json:"nextPageNum"`
			PageSize       int           `json:"pageSize"`
			EndPage        bool          `json:"endPage"`
			List           []TmpResponse `json:"list"`
		}{
			Total:          total,
			NextId:         "",
			Cnt:            count,
			CurrentPageNum: pageNum,
			NextPageNum:    pageNum + 1,
			PageSize:       pageSize,
			EndPage:        flag,
			List:           l,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) SongOperation(context *gin.Context) {
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
	var songId string
	var op string
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		xl.Infof("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if songId0, ok := input["songId"].(string); ok {
		songId = songId0
	} else {
		xl.Infof("miss songId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if op0, ok := input["operateType"].(string); ok {
		op = op0
	} else {
		xl.Infof("miss operateType in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	roomUserSong, err := k.roomUserSongDao.SelectByRoomIdSongId(xl, roomId, songId)
	if err != nil && err == mgo.ErrNotFound {
		roomUserSong = &model.RoomUserSongDo{
			RoomId: roomId,
			UserId: userId,
			SongId: songId,
			Status: model.SongAvailable,
		}
		_, _ = k.roomUserSongDao.Insert(xl, roomUserSong)
	}
	if roomUserSong.UserId != userId {
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
			}{},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	if op == "select" {
		roomUserSong.Status = model.RoomUserSongAvailable
	} else if op == "delete" {
		roomUserSong.Status = model.RoomUserSongUnavailable
	}
	_ = k.roomUserSongDao.Update(xl, roomUserSong)
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      true,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) SongInfo(context *gin.Context) {
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
	var songId string
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		roomId = "-1"
		xl.Infof("miss roomId: %s in body.", roomId)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if songId0, ok := input["songId"].(string); ok {
		songId = songId0
	} else {
		xl.Infof("miss songId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	song, err := k.songDao.Select(xl, songId)
	if err != nil && err == mgo.ErrNotFound {
		song = nil
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Song *model.SongDo `json:"song"`
		}{
			Song: song,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) AddSongs(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to add song.", userId)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if songs0, ok := input["songs"].([]interface{}); ok {
		for _, song0 := range songs0 {
			if song, ok := song0.(map[string]interface{}); ok {
				songDo := model.SongDo{
					Name:             song["name"].(string),
					Album:            song["album"].(string),
					Image:            song["image"].(string),
					Author:           song["author"].(string),
					Kind:             song["kind"].(string),
					OriginUrl:        song["originUrl"].(string),
					AccompanimentUrl: song["accompanimentUrl"].(string),
					Lyrics:           song["lyrics"].(string),
					Status:           model.SongAvailable,
				}
				_, err := k.songDao.SelectByNameAndAuthor(xl, songDo.Name, songDo.Author)
				if err == mgo.ErrNotFound {
					_, _ = k.songDao.Insert(xl, &songDo)
				}
			}
		}
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      nil,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) UpdateSong(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to update song.", userId)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if song0, ok := input["song"].(map[string]interface{}); ok {
		songId := song0["songId"].(string)
		songDo, err := k.songDao.Select(xl, songId)
		if err != nil {
			if err == mgo.ErrNotFound {
				resp := &model.Response{
					Code:      model.ResponseErrorBadRequest,
					Message:   string(model.ResponseStatusMessageSuccess),
					Data:      nil,
					RequestID: requestId,
				}
				context.JSON(http.StatusOK, resp)
				return
			} else {
				resp := &model.Response{
					Code:      model.ResponseErrorInternal,
					Message:   string(model.ResponseStatusMessageSuccess),
					Data:      nil,
					RequestID: requestId,
				}
				context.JSON(http.StatusOK, resp)
				return
			}
		}
		if song0["name"] != nil {
			songDo.Name = song0["name"].(string)
		}
		if song0["album"] != nil {
			songDo.Album = song0["album"].(string)
		}
		if song0["image"] != nil {
			songDo.Image = song0["image"].(string)
		}
		if song0["author"] != nil {
			songDo.Author = song0["author"].(string)
		}
		if song0["kind"] != nil {
			songDo.Kind = song0["kind"].(string)
		}
		if song0["originUrl"] != nil {
			songDo.OriginUrl = song0["originUrl"].(string)
		}
		if song0["accompanimentUrl"] != nil {
			songDo.AccompanimentUrl = song0["accompanimentUrl"].(string)
		}
		if song0["lyrics"] != nil {
			songDo.Lyrics = song0["lyrics"].(string)
		}
		_ = k.songDao.Update(xl, songDo)
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      nil,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) DeleteSong(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to delete song.", userId)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if songId, ok := input["songId"].(string); ok {
		songDo, _ := k.songDao.Select(xl, songId)
		if songDo != nil {
			songDo.Status = model.SongUnavailable
			_ = k.songDao.Update(xl, songDo)
		}
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      nil,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (k *KtvApiHandler) ListAllSong(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to list songs.", userId)
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	songDos, _, _, _ := k.songDao.ListAll(xl, pageNum, pageSize)
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      songDos,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}
