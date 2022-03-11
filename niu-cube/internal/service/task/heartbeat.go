package task

import (
	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"github.com/solutions/niu-cube/internal/service/db"
	"github.com/solutions/niu-cube/internal/service/db/dao"
)

type HeartBeatCheckTask struct {
	interviewService *db.InterviewService
	rtc              *cloud.RTCService
	taskColl         *mgo.Collection
	xl               *xlog.Logger
}

func NewHeartBeatTask(conf utils.Config) *HeartBeatCheckTask {
	xl := xlog.New("heartbeat kick task")
	var err error
	interviewService, err := db.NewInterviewService(*conf.Mongo, xl)
	if err != nil {
		panic(err)
	}
	client, err := mgo.Dial(conf.Mongo.URI)
	if err != nil {
		panic(err)
	}
	taskColl := client.DB(utils.DefaultConf.Mongo.Database).C(dao.TaskCollection)
	return &HeartBeatCheckTask{
		interviewService: interviewService,
		rtc:              cloud.NewRtcService(utils.DefaultConf),
		taskColl:         taskColl,
		xl:               xl,
	}
}

func (h *HeartBeatCheckTask) Start() {
	users, err := h.interviewService.ListHeartBeatTimeOutUser(h.xl)
	if err != nil {
		h.xl.Errorf("error list heartbeat timeout user:%v", err)
		return
	}
	for _, user := range users {
		var handleFunc = func() (result string, err error) {
			err = h.kickIfTimeout(user.InterviewID, user.UserID)
			if err == nil {
				result = "success kick timeout user"
				h.xl.Infof(result)
			} else {
				h.xl.Errorf("failed kick user %v err:%v", user.UserID, err)
			}
			return
		}
		model.NewTask(user.ID, "user", "kickOut").Handle(handleFunc).Start(h.taskColl, h.xl)
	}
}

// kickIfTimeout kick and update interview_user table, should be atomic op
func (h *HeartBeatCheckTask) kickIfTimeout(roomId, userId string) error {
	err := h.rtc.KickUser(roomId, userId)
	if err != nil {
		// rtc踢人失败 但是也缺少了心跳 认为离开
		h.xl.Errorf("err kick rtc user %v err:%v", userId, err)
	}
	err = h.interviewService.LeaveInterview(h.xl, userId, roomId)
	if err != nil {
		return err
	}
	return nil
}
