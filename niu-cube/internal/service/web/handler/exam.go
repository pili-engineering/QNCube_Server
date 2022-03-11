package handler

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"github.com/solutions/niu-cube/internal/service/dao"
	"github.com/solutions/niu-cube/internal/service/db"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type ExamApi interface {
	CreateExam(context *gin.Context)

	DeleteExam(context *gin.Context)

	UpdateExam(context *gin.Context)

	GetExamInfo(context *gin.Context)

	GetExamExaminees(context *gin.Context)

	JoinExam(context *gin.Context)

	GetExamPaper(context *gin.Context)

	LeaveExam(context *gin.Context)

	CommitExamAnswer(context *gin.Context)

	GetExamAnswerDetails(context *gin.Context)

	ListExamStudent(context *gin.Context)

	ListExamTeacher(context *gin.Context)

	QuestionList(context *gin.Context)

	AddQuestion(context *gin.Context)

	UpdateQuestion(context *gin.Context)

	DeleteQuestion(context *gin.Context)

	RoomToken(context *gin.Context)

	UploadCheatingEvent(context *gin.Context)

	MoreCheatingEvent(context *gin.Context)

	AiToken(context *gin.Context)

	PandoraToken(context *gin.Context)

	Clear(context *gin.Context)

	RunOnstart()

	SyncExamList(userId string)

	SyncByPhone(context *gin.Context)
}

type ExamApiHandler struct {
	logger                     *xlog.Logger
	examDao                    dao.ExamDao
	questionDao                dao.QuestionDao
	examPaperDao               dao.ExamPaperDao
	userExamDao                dao.UserExamDao
	answerPaperDao             dao.AnswerPaperDao
	baseRoomDao                dao.BaseRoomDaoInterface
	baseUserDao                dao.BaseUserDaoInterface
	rtcService                 *cloud.RTCService
	tokenService               *TokenService
	cheatingExamDao            dao.CheatingEventDao
	cheatingEventLogFileWriter io.Writer
	accountDao                 AccountInterface
	config                     *utils.Config
}

func NewExamApiHandler(config *utils.Config) *ExamApiHandler {
	baseRoomDao, _ := dao.NewBaseRoomDaoService(nil, config.Mongo)
	baseUserDao, _ := dao.NewBaseUserDaoService(nil, config.Mongo)
	file, err := os.OpenFile(config.CheatingEventLogFile, os.O_WRONLY, os.ModeAppend)
	if os.IsNotExist(err) {
		_ = os.Mkdir(path.Dir(config.CheatingEventLogFile), os.ModePerm)
		file, _ = os.Create(config.CheatingEventLogFile)
	}
	accountService, _ := db.NewAccountService(*config.Mongo, nil)
	return &ExamApiHandler{
		logger:                     xlog.New("exam api handler"),
		examDao:                    dao.NewExamDaoService(config.Mongo),
		questionDao:                dao.NewQuestionDaoService(config.Mongo),
		examPaperDao:               dao.NewExamPaperDaoService(config.Mongo),
		userExamDao:                dao.NewUserExamDaoService(config.Mongo),
		answerPaperDao:             dao.NewAnswerPaperDaoService(config.Mongo),
		baseRoomDao:                baseRoomDao,
		baseUserDao:                baseUserDao,
		rtcService:                 cloud.NewRtcService(*config),
		tokenService:               NewTokenService(config),
		cheatingExamDao:            dao.NewCheatingEventDaoService(config.Mongo),
		cheatingEventLogFileWriter: file,
		accountDao:                 accountService,
		config:                     config,
	}
}

type CreateExamInput struct {
	ExamId    string   `json:"examId"`
	Name      string   `json:"name"`
	StartTime int64    `json:"startTime"`
	EndTime   int64    `json:"endTime"`
	Type      string   `json:"type"`
	Desc      string   `json:"desc"`
	Examinees []string `json:"examinees"`
	Paper     struct {
		Name         string   `json:"name"`
		TotalScore   int      `json:"totalScore"`
		QuestionList []string `json:"questionList"`
	} `json:"paper"`
}

func (e *ExamApiHandler) CreateExam(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	inputs := CreateExamInput{}
	inputs.Type = "default"
	err := context.BindJSON(&inputs)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if len(inputs.Paper.QuestionList) == 0 || inputs.Paper.Name == "" {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// 添加所有用户
	if len(inputs.Examinees) == 0 {
		allUsers, _ := e.baseUserDao.ListAll()
		for idx := range allUsers {
			inputs.Examinees = append(inputs.Examinees, allUsers[idx].Id)
		}
	}
	exam := model.ExamDo{
		Name:     inputs.Name,
		BgnTime:  time.UnixMilli(inputs.StartTime),
		EndTime:  time.UnixMilli(inputs.EndTime),
		Duration: inputs.EndTime - inputs.StartTime,
		Type:     inputs.Type,
		Desc:     inputs.Desc,
		Creator:  userId,
		// InvigilatorList: inputs.InvigilatorList,
		Status: model.ExamCreated,
	}
	// TODO 处理err
	_ = e.examDao.Insert(&exam)
	totalScore := 0.0
	for idx := range inputs.Paper.QuestionList {
		questionId := inputs.Paper.QuestionList[idx]
		question, _ := e.questionDao.Select(questionId)
		totalScore += question.Score
	}
	paper := model.ExamPaperDo{
		Name:         inputs.Paper.Name,
		ExamId:       exam.Id,
		QuestionList: inputs.Paper.QuestionList,
		TotalScore:   int(totalScore),
	}
	// TODO 处理err
	_ = e.examPaperDao.Insert(&paper)
	for idx := range inputs.Examinees {
		userExam := model.UserExamDo{
			UserId:      inputs.Examinees[idx],
			ExamId:      exam.Id,
			ExamPaperId: paper.Id,
			RoomId:      "",
		}
		_ = e.userExamDao.Insert(&userExam)
	}
	utils.TimedTask(exam.BgnTime, func() {
		exam.Status = model.ExamInProgress
		_ = e.examDao.Update(&exam)
	})
	utils.TimedTask(exam.EndTime, func() {
		exam.Status = model.ExamFinished
		_ = e.examDao.Update(&exam)
	})
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			ExamId string `json:"examId"`
		}{
			ExamId: exam.Id,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) DeleteExam(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	// userId := context.GetString(model.UserIDContextKey)
	input := make(map[string]interface{})
	err := context.Bind(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	examId := input["examId"].(string)
	_ = e.examDao.Delete(examId)
	userExams, _ := e.userExamDao.ListByExamId0(examId)
	for idx := range userExams {
		userExams[idx].Status = model.UserExamDestroyed
		_ = e.userExamDao.Update(&userExams[idx])
	}
	examPapers, _ := e.examPaperDao.ListByExamId(examId)
	for idx := range examPapers {
		examPapers[idx].Status = model.ExamPaperUnAvailable
		_ = e.examPaperDao.Update(&examPapers[idx])
	}
	answerPapers, _ := e.answerPaperDao.ListByExamId0(examId)
	for idx := range answerPapers {
		answerPapers[idx].Status = model.AnswerPaperUnavailable
		_ = e.answerPaperDao.Update(&answerPapers[idx])
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) UpdateExam(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	inputs := CreateExamInput{}
	inputs.Type = "default"
	err := context.BindJSON(&inputs)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	exam, _ := e.examDao.Select(inputs.ExamId)
	if exam == nil {
		resp := &model.Response{
			Code:    model.ResponseErrorBadRequest,
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				Result bool `json:"result"`
			}{
				Result: false,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	if inputs.Name != "" {
		exam.Name = inputs.Name
	}
	if inputs.Type != "" {
		exam.Type = inputs.Type
	}
	if inputs.StartTime != 0 {
		exam.BgnTime = time.UnixMilli(inputs.StartTime)
	}
	if inputs.EndTime != 0 {
		exam.EndTime = time.UnixMilli(inputs.EndTime)
	}
	if inputs.Desc != "" {
		exam.Desc = inputs.Desc
	}
	if len(inputs.Paper.QuestionList) != 0 {
		examPapers, _ := e.examPaperDao.ListByExamId(exam.Id)
		examPapers[0].QuestionList = inputs.Paper.QuestionList
		examPapers[0].Name = inputs.Paper.Name
		_ = e.examPaperDao.Update(&examPapers[0])
	}
	_ = e.examDao.Update(exam)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type UserExamRoomInputs struct {
	ExamId string `json:"examId"`
	UserId string `json:"userId"`
	RoomId string `json:"roomId"`
}

func (e *ExamApiHandler) GetExamInfo(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	examId := context.Param("examId")
	// TODO err
	exam, _ := e.examDao.Select(examId)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			*ExamResult
		}{
			examConverter(exam),
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type ExamExamineesResult struct {
	UserId          string                `json:"userId"`
	UserName        string                `json:"userName"`
	ExamPaperStatus int                   `json:"examPaperStatus"`
	RtcInfo         model.RtcInfoResponse `json:"rtcInfo"`
	RoomInfo        struct {
		RoomId string `json:"roomId"`
	} `json:"roomInfo"`
}

func (e *ExamApiHandler) GetExamExaminees(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	examId := context.Param("examId")
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	userExams, total, _ := e.userExamDao.ListByExamId(examId, int64(pageNum), int64(pageSize))
	list := make([]ExamExamineesResult, 0, len(userExams))
	for idx := range userExams {
		user, _ := e.baseUserDao.Select(e.logger, userExams[idx].UserId)
		if userExams[idx].RoomId == "" {
			list = append(list, ExamExamineesResult{
				UserId:          user.Id,
				UserName:        user.Name,
				ExamPaperStatus: model.UserExamToBeInvolved,
				RtcInfo: model.RtcInfoResponse{
					RoomToken:   "",
					PublishUrl:  "",
					RtmpPlayUrl: "",
					FlvPlayUrl:  "",
					HlsPlayUrl:  "",
				},
				RoomInfo: struct {
					RoomId string `json:"roomId"`
				}{
					RoomId: "",
				},
			})
		} else {
			result := ExamExamineesResult{
				UserId:          user.Id,
				UserName:        user.Name,
				ExamPaperStatus: userExams[idx].Status,
				RtcInfo: model.RtcInfoResponse{
					RoomToken:   e.rtcService.GenerateRTCRoomToken(userExams[idx].RoomId, user.Id, ADMIN),
					PublishUrl:  e.rtcService.StreamPubURL(userExams[idx].RoomId),
					RtmpPlayUrl: e.rtcService.StreamRtmpPlayURL(userExams[idx].RoomId),
					FlvPlayUrl:  e.rtcService.StreamFlvPlayURL(userExams[idx].RoomId),
					HlsPlayUrl:  e.rtcService.StreamHlsPlayURL(userExams[idx].RoomId),
				},
				RoomInfo: struct {
					RoomId string `json:"roomId"`
				}{
					RoomId: userExams[idx].RoomId,
				},
			}
			list = append(list, result)
		}
	}
	isEnd := false
	if len(userExams) != pageSize {
		isEnd = true
	}
	type TempListResult struct {
		ListResult
		Timestamp int64 `json:"timestamp"`
		Interval  int64 `json:"interval"`
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: TempListResult{
			ListResult: ListResult{
				Total:          total,
				NextId:         "",
				Cnt:            int64(len(userExams)),
				CurrentPageNum: int64(pageNum),
				NextPageNum:    int64(pageNum + 1),
				PageSize:       int64(pageSize),
				EndPage:        isEnd,
				List:           list,
			},
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) JoinExam(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	inputs := UserExamRoomInputs{}
	err := context.BindJSON(&inputs)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	exam, _ := e.examDao.Select(inputs.ExamId)
	if exam == nil {
		// TODO 处理空
		return
	}
	// 时间处理
	if exam.Status != model.ExamInProgress {
		resp := &model.Response{
			Code:    model.ResponseErrorExamTimeNotMatch,
			Message: "考试时间不匹配",
			Data: struct {
				Result bool `json:"result"`
			}{
				Result: false,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	// TODO err
	userExam, _ := e.userExamDao.SelectByExamIdUserId(inputs.ExamId, userId)
	// 禁止重复参加
	if userExam.Status == model.UserExamFinished {
		resp := &model.Response{
			Code:    model.ResponseErrorExamDuplicateEntry,
			Message: "您已结束考试，无法再次参加",
			Data: struct {
				Result bool `json:"result"`
			}{
				Result: false,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	userExam.RoomId = inputs.RoomId
	userExam.Status = model.UserExamInProgress
	// TODO 处理err
	_ = e.userExamDao.Update(userExam)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type ExamPaperResult struct {
	PaperName    string             `json:"paperName"`
	TotalScore   int                `json:"totalScore"`
	QuestionList []model.QuestionDo `json:"questionList"`
}

func (e *ExamApiHandler) GetExamPaper(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	examId := context.Param("examId")
	// TODO err
	exam, _ := e.examDao.Select(examId)
	if exam.Status != model.ExamInProgress {
		resp := &model.Response{
			Code:    model.ResponseErrorExamTimeNotMatch,
			Message: "时间不匹配",
			Data: struct {
			}{},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	// TODO 处理err
	userExam, _ := e.userExamDao.SelectByExamIdUserId(examId, userId)
	// TODO 处理err
	examPaper, _ := e.examPaperDao.Select(userExam.ExamPaperId)
	result := ExamPaperResult{
		PaperName:    examPaper.Name,
		TotalScore:   examPaper.TotalScore,
		QuestionList: make([]model.QuestionDo, len(examPaper.QuestionList), len(examPaper.QuestionList)),
	}
	for idx := range examPaper.QuestionList {
		tmp, _ := e.questionDao.Select(examPaper.QuestionList[idx])
		result.QuestionList[idx] = *tmp
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      result,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) LeaveExam(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	inputs := UserExamRoomInputs{}
	err := context.BindJSON(&inputs)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	userExam, _ := e.userExamDao.SelectByExamIdUserId(inputs.ExamId, userId)
	if userExam == nil {
		// TODO 处理空
		return
	}
	userExam.Status = model.UserExamFinished
	// TODO 处理err
	_ = e.userExamDao.Update(userExam)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type AnswerInput struct {
	ExamId     string `json:"examId"`
	AnswerList []struct {
		QuestionId string   `json:"questionId"`
		TextList   []string `json:"textList"`
	} `json:"answerList"`
}

func (e *ExamApiHandler) CommitExamAnswer(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	inputs := AnswerInput{}
	err := context.BindJSON(&inputs)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	// TODO err
	exam, _ := e.examDao.Select(inputs.ExamId)
	if exam.Status != model.ExamInProgress {
		resp := &model.Response{
			Code:    model.ResponseErrorExamTimeNotMatch,
			Message: "考试时间不匹配",
			Data: struct {
				Result bool `json:"result"`
			}{
				Result: false,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
	answerPaper, _ := e.answerPaperDao.SelectByExamIdUserId(inputs.ExamId, userId)
	if answerPaper == nil {
		answerPaper = &model.AnswerPaperDo{
			UserId:     userId,
			ExamId:     inputs.ExamId,
			AnswerList: make([]model.AnswerDo, 0, len(inputs.AnswerList)),
		}
		// TODO err
		_ = e.answerPaperDao.Insert(answerPaper)
	}
	m := make(map[string]model.AnswerDo)
	for idx := range answerPaper.AnswerList {
		m[answerPaper.AnswerList[idx].QuestionId] = answerPaper.AnswerList[idx]
	}
	for idx := range inputs.AnswerList {
		answer := model.AnswerDo{}
		// TODO err
		question, _ := e.questionDao.Select(inputs.AnswerList[idx].QuestionId)
		// TODO Demo暂时设定为满分
		answer.Score = question.Score
		answer.QuestionId = question.Id
		answer.Type = question.Type
		if question.Type == model.MultiChoice || question.Type == model.SingleChoice {
			answer.ChoiceList = inputs.AnswerList[idx].TextList
		} else {
			switch question.Type {
			case model.Judge:
				answer.Judge, _ = strconv.ParseBool(inputs.AnswerList[idx].TextList[0])
			case model.Text:
				answer.Text = inputs.AnswerList[idx].TextList[0]
			}
		}
		m[question.Id] = answer
	}
	answerPaper.AnswerList = make([]model.AnswerDo, 0, len(m))
	for _, v := range m {
		answerPaper.AnswerList = append(answerPaper.AnswerList, v)
	}
	// TODO err
	_ = e.answerPaperDao.Update(answerPaper)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type QuestionAndAnswerList struct {
	QuestionId string `json:"questionId"`
	Question   struct {
		Type       string   `json:"type"`
		Score      float64  `json:"score"`
		Desc       string   `json:"desc"`
		ChoiceList []string `json:"choiceList"`
		Answer     struct {
			Correct  bool     `json:"correct"`
			TextList []string `json:"textList"`
		} `json:"answer"`
	} `json:"question"`
}

type ExamAnswerDetailsResult struct {
	PaperName  string                  `json:"paperName"`
	TotalScore float64                 `json:"totalScore"`
	List       []QuestionAndAnswerList `json:"list"`
}

func (e *ExamApiHandler) GetExamAnswerDetails(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	examId := context.Param("examId")
	userId0 := context.Param("userId")
	if userId0 != "/" {
		userId = userId0
	}
	// TODO err
	answerPaper, _ := e.answerPaperDao.SelectByExamIdUserId(examId, userId)
	answerPaper.TotalScore = 0
	for idx := range answerPaper.AnswerList {
		answerPaper.TotalScore += answerPaper.AnswerList[idx].Score
	}
	// TODO err
	_ = e.answerPaperDao.Update(answerPaper)
	userExam, _ := e.userExamDao.SelectByExamIdUserId(examId, userId)
	examPaper, _ := e.examPaperDao.Select(userExam.ExamPaperId)
	result := ExamAnswerDetailsResult{
		PaperName:  examPaper.Name,
		TotalScore: answerPaper.TotalScore,
		List:       make([]QuestionAndAnswerList, len(answerPaper.AnswerList), len(answerPaper.AnswerList)),
	}
	for idx := range answerPaper.AnswerList {
		result.List[idx].QuestionId = answerPaper.AnswerList[idx].QuestionId
		questionTmp := &result.List[idx].Question
		question, _ := e.questionDao.Select(answerPaper.AnswerList[idx].QuestionId)
		answerTmp := &result.List[idx].Question.Answer
		answer := &answerPaper.AnswerList[idx]
		questionTmp.Type = question.Type
		questionTmp.Desc = question.Desc
		questionTmp.ChoiceList = question.ChoiceList
		questionTmp.Score = question.Score
		switch answer.Type {
		case model.SingleChoice, model.MultiChoice:
			answerTmp.TextList = answer.ChoiceList
		case model.Text:
			answerTmp.TextList = []string{answer.Text}
		case model.Judge:
			answerTmp.TextList = []string{strconv.FormatBool(answer.Judge)}
		}
		if math.Abs(answer.Score-question.Score) < 0.000001 {
			answerTmp.Correct = true
		} else {
			answerTmp.Correct = false
		}
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      &result,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type ListResult struct {
	Total          int64       `json:"total"`
	NextId         string      `json:"nextId"`
	Cnt            int64       `json:"cnt"`
	CurrentPageNum int64       `json:"currentPageNum"`
	NextPageNum    int64       `json:"nextPageNum"`
	PageSize       int64       `json:"pageSize"`
	EndPage        bool        `json:"endPage"`
	List           interface{} `json:"list"`
}

func (e *ExamApiHandler) ListExamStudent(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	// TODO err
	userExams, total, _ := e.userExamDao.ListByUserId(userId, int64(pageNum), int64(pageSize))
	isEnd := false
	if len(userExams) != pageSize {
		isEnd = true
	}
	exams := make([]ExamResult, 0, len(userExams))
	for idx := range userExams {
		temp, _ := e.examDao.Select(userExams[idx].ExamId)
		if temp == nil {
			continue
		}
		exams = append(exams, *examConverter(temp))
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: ListResult{
			Total:          total,
			NextId:         "",
			Cnt:            int64(len(exams)),
			CurrentPageNum: int64(pageNum),
			NextPageNum:    int64(pageNum + 1),
			PageSize:       int64(pageSize),
			EndPage:        isEnd,
			List:           exams,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type ExamListInner struct {
	PaperName    string   `json:"paperName"`
	QuestionList []string `json:"questionList"`
}

func (e *ExamApiHandler) ListExamTeacher(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	// userId := context.GetString(model.UserIDContextKey)
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	// TODO err
	examList, total, _ := e.examDao.ListAll(int64(pageNum), int64(pageSize))
	isEnd := false
	if len(examList) != pageSize {
		isEnd = true
	}
	exams := make([]ExamResult, 0, len(examList))
	for idx := range examList {
		if examList[idx].Status == model.ExamDestroyed {
			continue
		}
		examPapers, _ := e.examPaperDao.ListByExamId(examList[idx].Id)
		e := *examConverter(&examList[idx])
		e.Paper = &struct {
			PaperName    string   `json:"paperName"`
			QuestionList []string `json:"questionList"`
		}{}
		if len(examPapers) > 0 {
			e.Paper.PaperName = examPapers[0].Name
			e.Paper.QuestionList = examPapers[0].QuestionList
		} else {
			e.Paper.PaperName = "no-paper"
			e.Paper.QuestionList = make([]string, 0, 0)
		}
		exams = append(exams, e)
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: ListResult{
			Total:          total,
			NextId:         "",
			Cnt:            int64(len(exams)),
			CurrentPageNum: int64(pageNum),
			NextPageNum:    int64(pageNum + 1),
			PageSize:       int64(pageSize),
			EndPage:        isEnd,
			List:           exams,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) QuestionList(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	pageNum, _ := strconv.Atoi(context.DefaultQuery("pageNum", "1"))
	pageSize, _ := strconv.Atoi(context.DefaultQuery("pageSize", "10"))
	var questions []model.QuestionDo
	var total int64
	isEnd := false
	if pageNum == -1 && pageSize == -1 {
		questions, total, _ = e.questionDao.ListAll0()
		isEnd = true
	} else {
		questions, total, _ = e.questionDao.ListAll(int64(pageNum), int64(pageSize))
		if len(questions) != pageSize {
			isEnd = true
		}
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: ListResult{
			Total:          total,
			NextId:         "",
			Cnt:            int64(len(questions)),
			CurrentPageNum: int64(pageNum),
			NextPageNum:    int64(pageNum + 1),
			PageSize:       int64(pageSize),
			EndPage:        isEnd,
			List:           questions,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type QuestionInput struct {
	QuestionId string `json:"questionId"`
	Question   struct {
		model.QuestionDo
		Answer struct {
			TextList []string `json:"textList"`
		} `json:"answer"`
	} `json:"question"`
}

func (e *ExamApiHandler) AddQuestion(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	type QuestionList struct {
		Questions []QuestionInput `json:"questions"`
	}
	list := QuestionList{}
	err := context.BindJSON(&list)
	inputs := list.Questions
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	for idx := range inputs {
		question := model.QuestionDo{
			Type:       inputs[idx].Question.Type,
			Score:      inputs[idx].Question.Score,
			Desc:       inputs[idx].Question.Desc,
			ChoiceList: inputs[idx].Question.ChoiceList,
			Answer: model.AnswerDo{
				Type:       inputs[idx].Question.Type,
				Score:      inputs[idx].Question.Score,
				ChoiceList: nil,
				Judge:      false,
				Text:       "",
			},
		}
		if question.Type == model.SingleChoice || question.Type == model.MultiChoice {
			question.Answer.ChoiceList = inputs[idx].Question.Answer.TextList
		} else if question.Type == model.Judge {
			question.Answer.Judge, _ = strconv.ParseBool(inputs[idx].Question.Answer.TextList[0])
		} else if question.Type == model.Text {
			question.Answer.Text = inputs[idx].Question.Answer.TextList[0]
		}
		_ = e.questionDao.Insert(&question)
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) UpdateQuestion(context *gin.Context) {
}

func (e *ExamApiHandler) DeleteQuestion(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	input := QuestionInput{}
	err := context.BindJSON(&input)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	_ = e.questionDao.Delete(input.QuestionId)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) RoomToken(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	roomId := context.DefaultQuery("roomId", "")
	if roomId == "" {
		xl.Infof("invalid args in body, error: %v", "miss roomId")
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	userId := uuid.NewString()
	roomToken := e.rtcService.GenerateRTCRoomToken(roomId, userId, ADMIN)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			RoomToken string `json:"roomToken"`
		}{
			RoomToken: roomToken,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type UploadCheatingEvent struct {
	ExamId string `json:"examId"`
	UserId string `json:"userId"`
	Event  struct {
		Action string `json:"action"`
		Value  string `json:"value"`
	} `json:"event"`
}

func (e *ExamApiHandler) UploadCheatingEvent(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	event := UploadCheatingEvent{}
	err := context.BindJSON(&event)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	cheatingEvent := model.CheatingEvent{
		UserId: event.UserId,
		ExamId: event.ExamId,
		Action: event.Event.Action,
		Value:  event.Event.Value,
	}
	_ = e.cheatingExamDao.Insert(&cheatingEvent)
	bytes, _ := json.Marshal(cheatingEvent)
	bytes = append(bytes, byte('\n'))
	_ = e.appendToLogFile(bytes)
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

type MoreCheatingEvent struct {
	UserList      []string `json:"userList"`
	LastTimestamp int64    `json:"lastTimestamp"`
	ExamId        string   `json:"examId"`
}

type MoreCheatingEventResult struct {
	Timestamp int64 `json:"timestamp"`
	List      []struct {
		UserId    string `json:"userId"`
		EventList []struct {
			Action    string `json:"action"`
			Value     string `json:"value"`
			Timestamp int64  `json:"timestamp"`
		} `json:"eventList"`
	} `json:"list"`
}

func (e *ExamApiHandler) MoreCheatingEvent(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	moreCheatingEvent := MoreCheatingEvent{}
	err := context.BindJSON(&moreCheatingEvent)
	if err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	result := MoreCheatingEventResult{
		Timestamp: time.Now().UnixMilli(),
		List: make([]struct {
			UserId    string `json:"userId"`
			EventList []struct {
				Action    string `json:"action"`
				Value     string `json:"value"`
				Timestamp int64  `json:"timestamp"`
			} `json:"eventList"`
		}, 0, len(moreCheatingEvent.UserList)),
	}
	for idx := range moreCheatingEvent.UserList {
		cheatingEventList, _ := e.cheatingExamDao.ListByExamIdUserId(moreCheatingEvent.ExamId, moreCheatingEvent.UserList[idx], moreCheatingEvent.LastTimestamp, result.Timestamp)
		eventList := make([]struct {
			Action    string `json:"action"`
			Value     string `json:"value"`
			Timestamp int64  `json:"timestamp"`
		}, 0, len(cheatingEventList))
		for i := range cheatingEventList {
			eventList = append(eventList, struct {
				Action    string `json:"action"`
				Value     string `json:"value"`
				Timestamp int64  `json:"timestamp"`
			}{
				Action:    cheatingEventList[i].Action,
				Value:     cheatingEventList[i].Value,
				Timestamp: cheatingEventList[i].Timestamp,
			})
		}
		result.List = append(result.List, struct {
			UserId    string `json:"userId"`
			EventList []struct {
				Action    string `json:"action"`
				Value     string `json:"value"`
				Timestamp int64  `json:"timestamp"`
			} `json:"eventList"`
		}{
			UserId:    moreCheatingEvent.UserList[idx],
			EventList: eventList,
		})
	}
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      result,
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) AiToken(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	url := context.DefaultQuery("url", "")
	src := fmt.Sprintf("%s:%d", e.config.DoraAiAppId, time.Now().UnixMilli()/1000+6*60*60*12)
	encodedSrc := base64.URLEncoding.EncodeToString([]byte(src))
	h := hmac.New(sha1.New, []byte(e.config.DoraAiSk))
	h.Write([]byte(encodedSrc))
	encodedSign := base64.URLEncoding.EncodeToString(h.Sum(nil))
	fmt.Println(encodedSign)
	encodedSign = strings.ReplaceAll(encodedSign, "\\//g", "_")
	encodedSign = strings.ReplaceAll(encodedSign, "\\+/g", "-")
	aiToken := "QD " + e.config.DoraAiAk + ":" + encodedSign + ":" + encodedSrc
	if url != "" {
		h := hmac.New(sha1.New, []byte(e.config.DoraSignSk))
		h.Write([]byte(url))
		skEncodedSign := base64.StdEncoding.EncodeToString(h.Sum(nil))
		skEncodedSign = strings.ReplaceAll(skEncodedSign, "\\//g", "_")
		skEncodedSign = strings.ReplaceAll(skEncodedSign, "\\+/g", "-")
		signToken := e.config.DoraSignAk + ":" + skEncodedSign
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				AiToken   string `json:"aiToken"`
				SignToken string `json:"signToken"`
			}{
				AiToken:   aiToken,
				SignToken: signToken,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	} else {
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: struct {
				AiToken string `json:"aiToken"`
			}{
				AiToken: aiToken,
			},
			RequestID: requestId,
		}
		context.JSON(http.StatusOK, resp)
	}

}

func (e *ExamApiHandler) PandoraToken(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	pandoraToken, err := e.tokenService.GetToken(e.config.PandoraConfig.PandoraUsername, e.config.PandoraConfig.PandoraPass)
	if err != nil {
		e.logger.Errorf("申请PandoraToken失败: %v", err)
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Token string `json:"token"`
		}{Token: pandoraToken.Token},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) Clear(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	e.logger.Infof("user: %s try to clear db.", userId)
	e.examDao.DeleteAll()
	e.userExamDao.DeleteAll()
	e.examPaperDao.DeleteAll()
	e.answerPaperDao.DeleteAll()
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
		}{},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (e *ExamApiHandler) RunOnstart() {
	exams, _ := e.examDao.ListAll0()
	for idx := range exams {
		// 复制
		exam := exams[idx]
		if exam.Status == model.ExamDestroyed {
			continue
		}
		utils.TimedTask(exam.BgnTime, func() {
			if exam.Status == model.ExamDestroyed {
				return
			}
			exam.Status = model.ExamInProgress
			_ = e.examDao.Update(&exam)
		})
		utils.TimedTask(exam.EndTime, func() {
			if exam.Status == model.ExamDestroyed {
				return
			}
			exam.Status = model.ExamFinished
			_ = e.examDao.Update(&exam)
		})
	}
	// e.examDao.DeleteAll0()
	t := "single_choice"
	question := model.QuestionDo{
		Type:  t,
		Score: 10,
		Desc:  "单选一: 1+2*3-4等于几？",
		ChoiceList: []string{
			"A: 2",
			"B: 3",
			"C: 4",
			"D: 5",
		},
		Answer: model.AnswerDo{
			Type:       t,
			Score:      10,
			ChoiceList: []string{"B"},
			Judge:      false,
			Text:       "",
		},
	}
	e.questionDao.Select(question.Id)
	// e.questionDao.Insert(&question)
	question = model.QuestionDo{
		Type:  t,
		Score: 10,
		Desc:  "单选二: 中国国土面积是？",
		ChoiceList: []string{
			"A: 974万平方公里",
			"B: 1024万平方公里",
			"C: 968万平方公里",
			"D: 960万平方公里",
		},
		Answer: model.AnswerDo{
			Type:       t,
			Score:      10,
			ChoiceList: []string{"D"},
			Judge:      false,
			Text:       "",
		},
	}
	// e.questionDao.Insert(&question)
	question = model.QuestionDo{
		Type:  t,
		Score: 10,
		Desc:  "单选三: 新冠疫情起始于多少年？",
		ChoiceList: []string{
			"A: 2018",
			"B: 2019",
			"C: 2020",
			"D: 2021",
		},
		Answer: model.AnswerDo{
			Type:       t,
			Score:      10,
			ChoiceList: []string{"B"},
			Judge:      false,
			Text:       "",
		},
	}
	// e.questionDao.Insert(&question)
	question = model.QuestionDo{
		Type:  t,
		Score: 10,
		Desc:  "单选四: 特朗普下一届美国总统是？",
		ChoiceList: []string{
			"A: 特朗普",
			"B: 希拉里",
			"C: 拜登",
			"D: 摩耶",
		},
		Answer: model.AnswerDo{
			Type:       t,
			Score:      10,
			ChoiceList: []string{"C"},
			Judge:      false,
			Text:       "",
		},
	}
	// e.questionDao.Insert(&question)
	question = model.QuestionDo{
		Type:  t,
		Score: 10,
		Desc:  "单选五: 下面哪一个不是安卓手机？",
		ChoiceList: []string{
			"A: 小米",
			"B: 华为",
			"C: 魅族",
			"D: 苹果",
		},
		Answer: model.AnswerDo{
			Type:       t,
			Score:      10,
			ChoiceList: []string{"D"},
			Judge:      false,
			Text:       "",
		},
	}
	// e.questionDao.Insert(&question)
}

func (e *ExamApiHandler) appendToLogFile(content []byte) error {
	_, err := e.cheatingEventLogFileWriter.Write(content)
	return err
}

func (e *ExamApiHandler) SyncExamList(userId string) {
	examDos, err := e.examDao.ListAll0()
	if err != nil {
		e.logger.Error(err)
		return
	}
	for i := range examDos {
		userExamDo, _ := e.userExamDao.SelectByExamIdUserId(examDos[i].Id, userId)
		if userExamDo == nil {
			examPaperDos, err := e.examPaperDao.ListByExamId(examDos[i].Id)
			if err != nil {
				e.logger.Error(err)
				continue
			}
			if len(examPaperDos) == 0 {
				e.logger.Error("unknown error")
				continue
			}
			userExam := model.UserExamDo{
				UserId:      userId,
				ExamId:      examDos[i].Id,
				ExamPaperId: examPaperDos[0].Id,
				RoomId:      "",
			}
			_ = e.userExamDao.Insert(&userExam)
		}
	}
}

func (e *ExamApiHandler) SyncByPhone(context *gin.Context) {
	phone := context.Param("phone")
	accountDo, err := e.accountDao.GetAccountByPhone(nil, phone)
	if err != nil {
		context.JSON(200, "ok")
	}
	userDo, _ := e.baseUserDao.Select(nil, accountDo.ID)
	if userDo == nil {
		baseUser := model.BaseUserDo{
			Id:            accountDo.ID,
			Name:          accountDo.Nickname,
			Nickname:      accountDo.Nickname,
			Avatar:        accountDo.Avatar,
			Status:        model.BaseUserLogin,
			Profile:       "",
			BaseUserAttrs: make([]model.BaseEntryDo, 0, 0),
		}
		e.baseUserDao.Insert(nil, &baseUser)
	}
	e.SyncExamList(userDo.Id)
}

type ExamResult struct {
	Id       string `json:"examId"`
	Name     string `json:"examName"`
	BgnTime  int64  `json:"startTime"`
	EndTime  int64  `json:"endTime"`
	Duration int64  `json:"duration"`
	Type     string `json:"type"`
	Desc     string `json:"desc"`
	Creator  string `json:"creator"`
	Status   int    `json:"status"`
	Paper    *struct {
		PaperName    string   `json:"paperName"`
		QuestionList []string `json:"questionList"`
	} `json:"paper,omitempty"`
}

func examConverter(exam *model.ExamDo) *ExamResult {
	return &ExamResult{
		Id:       exam.Id,
		Name:     exam.Name,
		BgnTime:  exam.BgnTime.UnixMilli(),
		EndTime:  exam.EndTime.UnixMilli(),
		Duration: exam.Duration,
		Type:     exam.Type,
		Desc:     exam.Desc,
		Creator:  exam.Creator,
		Status:   exam.Status,
		Paper:    nil,
	}
}
