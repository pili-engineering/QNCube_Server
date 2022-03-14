package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/dao"
)

type MovieApi interface {
	ListMovie(context *gin.Context)

	MovieDemanded(context *gin.Context)

	MovieOperation(context *gin.Context)

	MovieInfo(context *gin.Context)

	MovieSwitch(context *gin.Context)

	AddMovies(context *gin.Context)

	UpdateMovie(context *gin.Context)

	DeleteMovie(context *gin.Context)

	ListAllMovie(context *gin.Context)
}

type MovieApiHandler struct {
	baseRoomDao      dao.BaseRoomDaoInterface
	movieDao         dao.MovieDaoInterface
	roomUserMovieDao dao.RoomUserMovieInterface
}

func NewMovieApiHandler(xl *xlog.Logger, config *utils.MongoConfig) *MovieApiHandler {
	baseRoomDao, err := dao.NewBaseRoomDaoService(xl, config)
	if err != nil {
		xl.Error("create BaseRoomDaoService failed.")
	}
	movieDao, err := dao.NewMovieDaoService(xl, config)
	if err != nil {
		xl.Error("create MovieDaoService failed.")
		return nil
	}
	roomUserMovieDao, err := dao.NewRoomUserMovieService(xl, config)
	if err != nil {
		xl.Error("create RoomUserMovieDaoService failed.")
		return nil
	}
	return &MovieApiHandler{
		baseRoomDao,
		movieDao,
		roomUserMovieDao,
	}
}

func (m *MovieApiHandler) ListMovie(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	list, total, _ := m.movieDao.ListAll(xl, pageNum, pageSize)
	flag := false
	if pageNum*pageSize >= total {
		flag = true
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Total          int             `json:"total"`
			NextId         string          `json:"nextId"`
			Cnt            int             `json:"cnt"`
			CurrentPageNum int             `json:"currentPageNum"`
			NextPageNum    int             `json:"nextPageNum"`
			PageSize       int             `json:"pageSize"`
			EndPage        bool            `json:"endPage"`
			List           []model.MovieDo `json:"list"`
		}{
			Total:          total,
			NextId:         "",
			Cnt:            len(list),
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

func (m *MovieApiHandler) MovieDemanded(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	roomId := context.DefaultQuery("roomId", "no-room-id")
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	list, total, _ := m.roomUserMovieDao.ListByRoomId(xl, roomId, pageNum, pageSize)
	flag := false
	if pageNum*pageSize >= total {
		flag = true
	}
	type TmpResponse struct {
		model.MovieDo
		Demander string `json:"demander"`
	}
	l := make([]TmpResponse, 0, len(list))
	for _, val := range list {
		song, _ := m.movieDao.Select(xl, val.MovieId)
		l = append(l, TmpResponse{
			MovieDo:  *song,
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
			Cnt:            len(l),
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

func (m *MovieApiHandler) MovieOperation(context *gin.Context) {
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
	var movieId string
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
	if movieId0, ok := input["movieId"].(string); ok {
		movieId = movieId0
	} else {
		xl.Infof("miss movieId in body.")
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
	isRoomMaster := false
	roomDo, err := m.baseRoomDao.Select(xl, roomId)
	if roomDo.Creator == userId {
		isRoomMaster = true
	}
	roomUserMovieDo, err := m.roomUserMovieDao.SelectByRoomIdMovieId(xl, roomId, movieId)
	if err != nil && err == mgo.ErrNotFound {
		roomUserMovieDo = &model.RoomUserMovieDo{
			RoomId:          roomId,
			UserId:          userId,
			MovieId:         movieId,
			RoomMaster:      isRoomMaster,
			Playing:         false,
			CurrentSchedule: 0,
			Status:          model.RoomUserMovieAvailable,
		}
		_ = m.roomUserMovieDao.Insert(xl, roomUserMovieDo)
	}
	if roomUserMovieDo.UserId != userId {
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
		roomUserMovieDo.Status = model.RoomUserSongAvailable
	} else if op == "delete" {
		roomUserMovieDo.Status = model.RoomUserSongUnavailable
	}
	_ = m.roomUserMovieDao.Update(xl, roomUserMovieDo)
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      true,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (m *MovieApiHandler) MovieInfo(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	movieId := context.DefaultQuery("movieId", "no-movie-id")
	movieDo, err := m.movieDao.Select(xl, movieId)
	if err != nil && err == mgo.ErrNotFound {
		movieDo = nil
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Movie *model.MovieDo `json:"movie"`
		}{
			Movie: movieDo,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (m *MovieApiHandler) MovieSwitch(context *gin.Context) {
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
	var movieId string
	if roomId0, ok := input["roomId"].(string); ok {
		roomId = roomId0
	} else {
		xl.Infof("miss roomId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if movieId0, ok := input["movieId"].(string); ok {
		movieId = movieId0
	} else {
		xl.Infof("miss movieId in body.")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	roomUserMovieDo, _ := m.roomUserMovieDao.SelectByRoomIdPlaying(xl, roomId)
	if roomUserMovieDo != nil {
		roomUserMovieDo.Playing = false
		_ = m.roomUserMovieDao.Update(xl, roomUserMovieDo)
	}
	roomUserMovieDo, _ = m.roomUserMovieDao.SelectByRoomIdMovieId(xl, roomId, movieId)
	if roomUserMovieDo == nil {
		isRoomMaster := false
		roomDo, _ := m.baseRoomDao.Select(xl, roomId)
		if roomDo.Creator == userId {
			isRoomMaster = true
		}
		roomUserMovieDo = &model.RoomUserMovieDo{
			RoomId:          roomId,
			UserId:          userId,
			MovieId:         movieId,
			RoomMaster:      isRoomMaster,
			CurrentSchedule: 0,
			Playing:         true,
			Status:          model.RoomUserMovieAvailable,
		}
		_ = m.roomUserMovieDao.Insert(xl, roomUserMovieDo)
	} else {
		roomUserMovieDo.Playing = true
		_ = m.roomUserMovieDao.Update(xl, roomUserMovieDo)
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      true,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (m *MovieApiHandler) AddMovies(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to add movie.", userId)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if movies0, ok := input["movies"].([]interface{}); ok {
		for _, movie0 := range movies0 {
			if movie, ok := movie0.(map[string]interface{}); ok {
				var list1 []string
				var list2 []string
				if list, ok := movie["actorList"].([]interface{}); ok {
					for _, val := range list {
						list1 = append(list1, val.(string))
					}
				}
				if list, ok := movie["kindList"].([]interface{}); ok {
					for _, val := range list {
						list2 = append(list2, val.(string))
					}
				}
				movieDo := model.MovieDo{
					Name:        movie["name"].(string),
					Director:    movie["director"].(string),
					Image:       movie["image"].(string),
					ActorList:   list1,
					KindList:    list2,
					Duration:    uint64(movie["duration"].(float64)),
					PlayUrl:     movie["playUrl"].(string),
					Lyrics:      movie["lyrics"].(string),
					Desc:        movie["desc"].(string),
					DoubanScore: movie["doubanScore"].(float64),
					ImdbScore:   movie["imdbScore"].(float64),
					ReleaseTime: time.UnixMilli(int64(movie["releaseTime"].(float64))),
					Status:      model.MovieAvailable,
				}
				_, err := m.movieDao.SelectByNameDirector(xl, movieDo.Name, movieDo.Director)
				if err == mgo.ErrNotFound {
					_ = m.movieDao.Insert(xl, &movieDo)
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

func (m *MovieApiHandler) UpdateMovie(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to update movie.", userId)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if movie0, ok := input["movie"].(map[string]interface{}); ok {
		movieId := movie0["movieId"].(string)
		movieDo, err := m.movieDao.Select(xl, movieId)
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
		if movie0["name"] != nil {
			movieDo.Name = movie0["name"].(string)
		}
		if movie0["director"] != nil {
			movieDo.Director = movie0["director"].(string)
		}
		if movie0["image"] != nil {
			movieDo.Image = movie0["image"].(string)
		}
		if movie0["actorList"] != nil {
			if list, ok := movie0["actorList"].([]interface{}); ok {
				for _, val := range list {
					movieDo.ActorList = append(movieDo.ActorList, val.(string))
				}
			}
		}
		if movie0["kindList"] != nil {
			if list, ok := movie0["kindList"].([]interface{}); ok {
				for _, val := range list {
					movieDo.KindList = append(movieDo.KindList, val.(string))
				}
			}
		}
		if movie0["duration"] != nil {
			movieDo.Duration = uint64(movie0["duration"].(float64))
		}
		if movie0["playUrl"] != nil {
			movieDo.PlayUrl = movie0["playUrl"].(string)
		}
		if movie0["lyrics"] != nil {
			movieDo.Lyrics = movie0["lyrics"].(string)
		}
		if movie0["desc"] != nil {
			movieDo.Desc = movie0["desc"].(string)
		}
		if movie0["doubanScore"] != nil {
			movieDo.DoubanScore = movie0["doubanScore"].(float64)
		}
		if movie0["imdbScore"] != nil {
			movieDo.ImdbScore = movie0["imdbScore"].(float64)
		}
		if movie0["releaseTime"] != nil {
			movieDo.ReleaseTime = time.UnixMilli(int64(movie0["releaseTime"].(float64)))
		}
		_ = m.movieDao.Update(xl, movieDo)
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      nil,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (m *MovieApiHandler) DeleteMovie(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to delete movie.", userId)
	var input map[string]interface{}
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if movieId, ok := input["movieId"].(string); ok {
		movieDo, _ := m.movieDao.Select(xl, movieId)
		if movieDo != nil {
			movieDo.Status = model.MovieUnavailable
			_ = m.movieDao.Update(xl, movieDo)
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

func (m *MovieApiHandler) ListAllMovie(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	xl.Info("user:[%s] try to list movies.", userId)
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	movieDos, _, _ := m.movieDao.ListAll(xl, pageNum, pageSize)
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      movieDos,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}
