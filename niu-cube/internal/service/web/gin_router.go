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

package web

import (
	"github.com/solutions/niu-cube/internal/service/dao"
	"net/http"
	"time"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"github.com/solutions/niu-cube/internal/service/db"
	"github.com/solutions/niu-cube/internal/service/web/handler"
	"github.com/solutions/niu-cube/internal/service/web/middleware"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
)

// NewRouter @title 互动直播API
// @version 0.0.1
// @description  http apis
// @BasePath /v1
// NewRouter 返回gin router，分流API。
func NewRouter(config *utils.Config) (*gin.Engine, error) {
	// 1. 初始化GIN
	router := gin.New()
	router.Use(gin.Recovery())
	// 1.1. 全局CORS配置
	router.Use(corsMiddleware())

	// 2. 声明Service
	// 2.1 全局配置Service
	appConfigService, err := db.NewAppConfigService(config.IM, nil)
	if err != nil {
		return nil, err
	}
	appConfigApiHandler := &handler.AppConfigApiHandler{}

	// 2.2 账号Service
	smsCodeService, err := cloud.NewSmsCodeService(config.Mongo.URI, config.Mongo.Database, config, nil)
	if err != nil {
		return nil, err
	}

	accountService, err := db.NewAccountService(*config.Mongo, nil)
	if err != nil {
		return nil, err
	}
	baseUserDao, err := dao.NewBaseUserDaoService(nil, config.Mongo)
	if err != nil {
		return nil, err
	}

	fileApiHandler := handler.NewFileApiHandler(*config)
	interviewApiHandler := handler.NewInterviewApiHandler(*config)
	repairApiHandler := handler.NewRepairApiHandler(*config)
	ieService := db.NewIEService()
	ieHandler := handler.NewIEHandler(ieService)

	//// 3.2 声明分页参数填充拦截
	//pageApiHandler := &handler.PageHandlerApi{}

	versionApiHandler := handler.NewVersionHandlerApi(*config)
	boardApiHandler := handler.NewBoardHandlerApi(*config)

	// 通用相关
	baseRoom := handler.NewBaseRoomApiHandler(xlog.New("base-room-api"), config)
	baseUser := handler.NewBaseUserApiHandler(xlog.New("base-user-api"), config)
	baseMic := handler.NewBaseMicApiHandler(xlog.New("base-mic-api"), config)

	// KT相关
	ktv := handler.NewKtvApiHandler(xlog.New("ktv-api"), config.Mongo)

	// 在线看电影相关
	movie := handler.NewMovieApiHandler(xlog.New("movie-api"), config.Mongo)

	// 在线考试相关
	exam := handler.NewExamApiHandler(config)
	exam.RunOnstart()

	accountApiHandler := &handler.AccountApiHandler{
		Account:           accountService,
		SmsCode:           smsCodeService,
		AppConfigService:  appConfigService,
		DefaultAvatarURLs: config.DefaultAvatars,
		BaseUserDao:       baseUserDao,
		ExamService:       exam,
	}

	middleware.InitMiddleware(*config)

	// 4. 配置V1路径
	v1 := router.Group("/v1", addApiVersion(model.ApiVersionV1), addRequestID, middleware.FetchPageInfo, middleware.ActionLogMiddleware())
	{
		// 3.1 通用|获取APP全局配置
		v1.GET("appConfig", appConfigApiHandler.GetAppConfig)
		v1.GET("appConfig/", appConfigApiHandler.GetAppConfig)
		// TODO： 增加鉴权
		v1.GET("token/kodo", appConfigApiHandler.KodoToken)
		v1.GET("token/kodo/", appConfigApiHandler.KodoToken)
		// 3.2 发送验证码
		v1.POST("getSmsCode", accountApiHandler.SendSmsCode)
		v1.POST("getSmsCode/", accountApiHandler.SendSmsCode)
		// 3.3 登录/注册
		v1.POST("signUpOrIn", accountApiHandler.SignUpOrIn)
		v1.POST("signUpOrIn/", accountApiHandler.SignUpOrIn)

		v1.POST("token/getToken", appConfigApiHandler.GetToken)

		v1.GET("test/:interviewId", interviewApiHandler.InterviewUrlFromId)

		// 3.4 文件上传下载相关
		v1.POST("upload", fileApiHandler.Upload)
		v1.GET("recentImage", fileApiHandler.RecentImage)

		v1.GET("exam/roomToken", exam.RoomToken)

		v1.GET("exam/aiToken", exam.AiToken)

		v1.GET("/pandora/token", exam.PandoraToken)

	}
	baseAuth := v1.Group("", middleware.Authenticate)
	{
		// 3.3 登录/注册
		baseAuth.POST("signInWithToken", accountApiHandler.SignInWithToken)
		baseAuth.POST("signInWithToken/", accountApiHandler.SignInWithToken)

		// 3.4 登出
		baseAuth.POST("signOut", accountApiHandler.SignOut)
		baseAuth.POST("signOut/", accountApiHandler.SignOut)
		// 3.5 场景列表
		baseAuth.GET("solution", appConfigApiHandler.SolutionList)
		baseAuth.GET("solution/", appConfigApiHandler.SolutionList)
		// 3.6 用户信息获取
		baseAuth.GET("accountInfo", accountApiHandler.GetAccountInfo)
		baseAuth.GET("accountInfo/", accountApiHandler.GetAccountInfo)
		baseAuth.GET("accountInfo/:accountId", accountApiHandler.GetAccountInfo)
		// 3.7 用户信息更新
		baseAuth.POST("accountInfo", accountApiHandler.UpdateAccountInfo)
		baseAuth.POST("accountInfo/", accountApiHandler.UpdateAccountInfo)
		baseAuth.POST("accountInfo/:accountId", accountApiHandler.UpdateAccountInfo)
		baseAuth.DELETE("account/delete/:phone", accountApiHandler.DeleteAccount)
		baseAuth.GET("account/sync", accountApiHandler.Sync)

		// 3.8 面试场景-面试列表
		baseAuth.GET("interview", interviewApiHandler.ListAllInterviews)
		baseAuth.GET("interview/", interviewApiHandler.ListAllInterviews)

		// 3.10 面试场景-取消面试
		baseAuth.POST("cancelInterview/:interviewId", interviewApiHandler.CancelInterview)

		// 3.14 面试场景-创建面试
		baseAuth.POST("interview", interviewApiHandler.CreatInterview)
		baseAuth.POST("interview/", interviewApiHandler.CreatInterview)
		// 3.15 面试场景-修改面试
		baseAuth.POST("interview/:interviewId", interviewApiHandler.UpdateInterview)

		// 4.1 检修场景-创建房间
		baseAuth.POST("repair/createRoom", repairApiHandler.CreateRoom)
		// 4.2 检修场景-加入房间
		baseAuth.POST("repair/joinRoom", repairApiHandler.JoinRoom)
		// 4.3 检修场景-离开房间
		baseAuth.GET("repair/leaveRoom/:roomId", repairApiHandler.LeaveRoom)
		// 4.4 检修场景-房间列表
		baseAuth.GET("repair/listRoom/", repairApiHandler.ListRoom)
		baseAuth.GET("repair/listRoom", repairApiHandler.ListRoom)
		baseAuth.POST("repair/listRoom/", repairApiHandler.ListRoom)
		baseAuth.POST("repair/listRoom", repairApiHandler.ListRoom)
		// 4.5 检修场景-心跳接口
		baseAuth.GET("repair/heartBeat/:roomId", repairApiHandler.HeartBeat)

		// 4.6 检修场景-获取房间信息
		baseAuth.GET("repair/getRoomInfo/:roomId", repairApiHandler.GetRoomInfo)

		// 通用创建房间
		baseAuth.POST("base/createRoom", baseRoom.CreateRoom)
		// 通用加入房间
		baseAuth.POST("base/joinRoom", baseRoom.JoinRoom)
		// 通用离开房间
		baseAuth.POST("base/leaveRoom", baseRoom.LeaveRoom)
		// 通用列举房间
		baseAuth.GET("base/listRoom", baseRoom.ListRooms)
		// 通用心跳保活
		baseAuth.GET("base/heartBeat", baseUser.Heartbeat)
		// 更新用户信息
		baseAuth.POST("/base/userInfo", baseUser.UpdateUserInfo)
		// 通用房间信息
		baseAuth.GET("base/getRoomInfo", baseRoom.RoomInfo)
		baseAuth.DELETE("base/room/groupChat/truncate", baseRoom.TruncateGroupChat)
		// 通用上麦接口
		baseAuth.POST("base/upMic", baseMic.UpMic)
		// 通用更新房间
		baseAuth.POST("base/updateRoomAttr", baseRoom.UpdateRoomInfo)
		// 通用更新麦位
		baseAuth.POST("base/updateMicAttr", baseMic.UpdateMicAttrs)
		// 通用下麦接口
		baseAuth.POST("base/downMic", baseMic.DownMic)
		// 通用房间麦位扩展信息
		baseAuth.GET("base/getRoomMicInfo", baseMic.MicInfo)
		// 通用房间属性
		baseAuth.GET("base/getRoomAttr", baseRoom.RoomInfoAttr)
		// 通用麦位属性
		baseAuth.GET("base/getMicAttr", baseMic.MicAttrs)

		baseAuth.GET("listUser/:roomId", baseRoom.ListUser)

		// 歌曲列表
		baseAuth.POST("ktv/songList", ktv.ListSong)
		// 当前用户已选歌曲
		baseAuth.POST("ktv/selectedSongList", ktv.SongDemanded)
		// 点歌/取消点歌
		baseAuth.POST("ktv/operateSong", ktv.SongOperation)
		// 歌曲信息
		baseAuth.POST("ktv/songInfo", ktv.SongInfo)
		// 添加歌曲
		baseAuth.POST("ktv/addSongs", ktv.AddSongs)
		// 更新歌曲
		baseAuth.POST("ktv/updateSong", ktv.UpdateSong)
		// 删除歌曲
		baseAuth.POST("ktv/deleteSong", ktv.DeleteSong)
		// 列举所有歌曲
		baseAuth.GET("ktv/listSongs", ktv.ListAllSong)

		// 在线看电影相关
		baseAuth.GET("watchMoviesTogether/movieList", movie.ListMovie)
		baseAuth.GET("watchMoviesTogether/selectedMovieList", movie.MovieDemanded)
		baseAuth.POST("watchMoviesTogether/movieOperation", movie.MovieOperation)
		baseAuth.GET("watchMoviesTogether/movieInfo", movie.MovieInfo)
		baseAuth.POST("watchMoviesTogether/switchMovie", movie.MovieSwitch)
		baseAuth.POST("movie/addMovies", movie.AddMovies)
		baseAuth.POST("movie/updateMovies", movie.UpdateMovie)
		baseAuth.POST("movie/deleteMovies", movie.DeleteMovie)
		baseAuth.GET("movie/listMovies", movie.ListAllMovie)

		// 在线考试相关
		baseAuth.POST("exam/create", exam.CreateExam)
		baseAuth.POST("exam/update", exam.UpdateExam)
		baseAuth.POST("exam/delete", exam.DeleteExam)
		baseAuth.POST("exam/join", exam.JoinExam)
		baseAuth.POST("exam/leave", exam.LeaveExam)
		baseAuth.GET("exam/info/:examId", exam.GetExamInfo)
		baseAuth.GET("exam/examinees/:examId", exam.GetExamExaminees)
		baseAuth.GET("exam/paper/:examId", exam.GetExamPaper)
		baseAuth.POST("exam/answer", exam.CommitExamAnswer)
		baseAuth.GET("exam/answer/details/:examId/*userId", exam.GetExamAnswerDetails)
		baseAuth.GET("exam/list/student", exam.ListExamStudent)
		baseAuth.GET("exam/list/teacher", exam.ListExamTeacher)
		baseAuth.GET("exam/questionList/*type", exam.QuestionList)
		baseAuth.POST("exam/questions/add", exam.AddQuestion)
		baseAuth.POST("exam/questions/update", exam.UpdateQuestion)
		baseAuth.POST("exam/question/delete", exam.DeleteQuestion)
		baseAuth.POST("exam/eventLog", exam.UploadCheatingEvent)
		baseAuth.POST("exam/eventLog/more", exam.MoreCheatingEvent)
		baseAuth.GET("exam/clear", exam.Clear)
		baseAuth.GET("exam/sync/:phone")
	}
	// 无状态登录
	stateLessAuth := v1.Use(middleware.AfapAuthenticate)
	{

		// 3.9 面试场景-结束面试
		stateLessAuth.POST("endInterview/:interviewId", interviewApiHandler.EndInterview)
		// 3.11 面试场景-进入面试
		stateLessAuth.POST("joinInterview/:interviewId", interviewApiHandler.JoinInterview)
		// 3.12 面试场景-离开面试
		stateLessAuth.POST("leaveInterview/:interviewId", interviewApiHandler.LeaveInterview)
		// 3.13 面试场景-心跳
		stateLessAuth.GET("heartBeat/:interviewId", interviewApiHandler.HeartBeat)
		// 3.16 面试场景-面试详情
		stateLessAuth.GET("interview/:interviewId", interviewApiHandler.GetInterview)

	}

	version := v1.Group("", middleware.Authenticate, middleware.VersionGate())
	{
		version.GET("version", versionApiHandler.GetOrListVersion)
		version.GET("version/", versionApiHandler.GetOrListVersion)

		version.POST("version", versionApiHandler.CreateVersion)
		version.POST("version/", versionApiHandler.CreateVersion)

		version.GET("version/:versionId", versionApiHandler.GetOrListVersion)
		version.DELETE("version/:versionId", versionApiHandler.DeleteVersion)
	}

	board := v1.Group("", middleware.AfapAuthenticate)
	{
		board.GET("board/:interviewId", boardApiHandler.GetBoard)
		//board.POST("board",boardApiHandler.CreateOrUpdateBoard)
		board.POST("board/:interviewId", boardApiHandler.CreateOrUpdateBoard)
		board.PUT("board/:interviewId", boardApiHandler.CreateOrUpdateBoard)
	}
	ie := v1.Group("ie", middleware.Authenticate)
	{
		ieHandler.RegisterRoute(ie)
	}

	appVersion := handler.NewAppVersionApiHandler(config.Mongo)

	// 5. 配置V1路径
	v2 := router.Group("/v2", addApiVersion(model.ApiVersionV2), addRequestID, middleware.FetchPageInfo, middleware.ActionLogMiddleware())
	{
		v2.GET("solution", appConfigApiHandler.SolutionList)
		v2.GET("solution/", appConfigApiHandler.SolutionList)
		v2.POST("/app/updates", appVersion.UpdateAppVersion)
		v2.GET("/app/updates", appVersion.GetNewestAppVersion)
	}

	router.NoRoute(addRequestID, returnNotFound)
	router.RedirectTrailingSlash = false

	return router, nil
}

// 增加当前接口调用版本
func addApiVersion(version model.ApiVersion) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(model.RequestApiVersion, version)
	}
}

func addRequestID(c *gin.Context) {
	requestID := ""
	if requestID = c.Request.Header.Get(model.RequestIDHeader); requestID == "" {
		requestID = utils.NewReqID()
		c.Request.Header.Set(model.RequestIDHeader, requestID)
	}
	xl := xlog.New(requestID)
	xl.Debugf("request: %s %s", c.Request.Method, c.Request.URL.Path)
	c.Set(model.XLogKey, xl)
	c.Set(model.RequestStartKey, time.Now())
}

func returnNotFound(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	xl.Debugf("%s %s: not found", c.Request.Method, c.Request.URL.Path)
	responseErr := model.NewResponseErrorNotFound()
	resp := model.NewFailResponse(*responseErr)
	c.JSON(http.StatusOK, resp)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, HEAD")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
