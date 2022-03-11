package task

import (
	"fmt"

	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	"github.com/solutions/niu-cube/internal/service/db/dao"
)

type RecordTask struct {
	Rtc           *cloud.RTCService
	conf          utils.Config
	interviewColl *mgo.Collection
	taskColl      *mgo.Collection
	client        *mgo.Session
	xl            *xlog.Logger
}

func NewRecordTask(conf utils.Config) *RecordTask {
	n := new(RecordTask)
	n.Rtc = cloud.NewRtcService(conf)
	n.xl = xlog.New("record task manager")
	var err error
	n.client, err = mgo.Dial(conf.Mongo.URI)
	if err != nil {
		n.xl.Fatalf("error fetching service client err:%v", err)
	}
	n.interviewColl = n.client.DB(conf.Mongo.Database).C(dao.InterviewCollection)
	n.taskColl = n.client.DB(conf.Mongo.Database).C(dao.TaskCollection)
	n.conf = conf
	return n
}

// Start 同步任务
// 面试状态已结束 && 开启录制
func (r RecordTask) Start() {
	tasks, err := r.listTasks()
	if err != nil {
		r.xl.Errorf("error fetching task err:%v", err)
		return
	}
	for _, interview := range tasks {
		model.NewTask(interview.ID, "interview", "record").Handle(r.genHandleFunc(interview)).Start(r.taskColl, r.xl)
	}
}
func (r *RecordTask) genHandleFunc(interview model.InterviewDo) func() (string, error) {
	return func() (string, error) {
		var result string
		var callback = func(resp map[string]string, ok bool) error {
			var err error
			if ok {
				result = fmt.Sprintf("%s/%s", r.conf.RTC.PlayBackURL, resp["fname"])
			} else {
				err = fmt.Errorf("failed stream name:%v, resp:%v", r.streamName(interview.ID), resp)
			}
			return err
		}
		err := r.Rtc.RecordPlayBackM3u8(r.streamName(interview.ID), 0, 0, callback)
		if err == nil {
			var newInterview model.InterviewDo
			_ = r.interviewColl.FindId(interview.ID).One(&newInterview)
			newInterview.Recorded = true
			_ = r.interviewColl.UpdateId(interview.ID, newInterview)
			return result, err
		} else {
			return "", err
		}
	}
}

func (r *RecordTask) listTasks() ([]model.InterviewDo, error) {
	condition := map[string]interface{}{
		"status":   model.InterviewStatusCodeEnd,
		"isRecord": true,
		"recorded": false,
	}
	interviews := make([]model.InterviewDo, 0)
	err := r.interviewColl.Find(condition).Limit(10).All(&interviews)
	if err != nil {
		r.xl.Errorf("fetch interview list err:%v", err)
		return interviews, err
	}
	return interviews, nil
}

func (r *RecordTask) streamName(interviewId string) string {
	return fmt.Sprintf(r.conf.RTC.StreamPattern, interviewId)
}
