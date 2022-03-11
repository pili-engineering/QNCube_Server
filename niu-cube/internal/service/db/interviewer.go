package db

import (
	"github.com/solutions/niu-cube/internal/common/utils"
	errors2 "github.com/solutions/niu-cube/internal/protodef/errors"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	dao "github.com/solutions/niu-cube/internal/service/db/dao"
	"time"

	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type InterviewService struct {
	mongoClient       *mgo.Session
	interviewColl     *mgo.Collection
	interviewUserColl *mgo.Collection
	accountTokenColl  *mgo.Collection
	taskColl          *mgo.Collection
	xl                *xlog.Logger
}

const (
	AllowedDelayDuration = time.Second * 3
)

func NewInterviewService(conf utils.MongoConfig, xl *xlog.Logger) (*InterviewService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-room-controller")
	}
	mongoClient, err := mgo.Dial(conf.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error %v", err)
		return nil, err
	}
	interviewColl := mongoClient.DB(conf.Database).C(dao.InterviewCollection)
	accountTokenColl := mongoClient.DB(conf.Database).C(dao.CollectionAccountToken)
	interviewerUserColl := mongoClient.DB(conf.Database).C(dao.InterviewUserCollection)
	taskColl := mongoClient.DB(conf.Database).C(dao.TaskCollection)
	return &InterviewService{
		mongoClient:       mongoClient,
		interviewColl:     interviewColl,
		interviewUserColl: interviewerUserColl,
		accountTokenColl:  accountTokenColl,
		taskColl:          taskColl,
		xl:                xl,
	}, nil
}

func (c *InterviewService) CreateInterview(xl *xlog.Logger, interview *model.InterviewDo) (*model.InterviewDo, error) {
	if xl == nil {
		xl = c.xl
	}
	err := c.interviewColl.Insert(interview)
	if err != nil {
		xl.Errorf("failed to update user status of user %s, error %v", interview.Creator, err)
		return nil, err
	}
	xl.Infof("user %s CreateInterview room %s", interview.Creator, interview.ID)
	createTime := time.Now()
	interviewer := model.InterviewUserDo{
		ID:             interview.ID + "_" + interview.Interviewer,
		InterviewID:    interview.ID,
		UserID:         interview.Interviewer,
		Status:         1,
		LastModifyTime: createTime,
	}
	candidate := model.InterviewUserDo{
		ID:             interview.ID + "_" + interview.Candidate,
		InterviewID:    interview.ID,
		UserID:         interview.Candidate,
		Status:         1,
		LastModifyTime: createTime,
	}
	err = c.interviewUserColl.Insert(interviewer)
	if err != nil {
		xl.Errorf("failed to update user status of user %s, error %v", interview.Creator, err)
		return nil, err
	}
	err = c.interviewUserColl.Insert(candidate)
	if err != nil {
		xl.Errorf("failed to update user status of user %s, error %v", interview.Creator, err)
		return nil, err
	}
	return interview, nil
}

func (c *InterviewService) ListInterviewsByPage(xl *xlog.Logger, userID string, pageNum int, pageSize int) ([]model.InterviewDo, int, error) {
	if xl == nil {
		xl = c.xl
	}
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	interviews := []model.InterviewDo{}
	err := c.interviewColl.Find(bson.M{"$or": []bson.M{bson.M{"candidate": userID}, bson.M{"interviewer": userID}, bson.M{"creator": userID}}}).Sort("-status", "startTime").Skip(skip).Limit(limit).All(&interviews)
	if err != nil {
		xl.Errorf("failed to ListInterviews of userId %s, error %v", userID, err)
		return nil, 0, err
	}
	total, err := c.interviewColl.Find(bson.M{"$or": []bson.M{bson.M{"candidate": userID}, bson.M{"interviewer": userID}, bson.M{"creator": userID}}}).Count()
	if err != nil {
		xl.Errorf("failed to ListInterviews of userId %s, error %v", userID, err)
		return nil, 0, err
	}
	return interviews, total, err
}

// GetRoomByFields 根据一组 key/value 关系查找直播房间。
func (c *InterviewService) GetInterviewByFields(xl *xlog.Logger, fields map[string]interface{}) (*model.InterviewDo, error) {
	if xl == nil {
		xl = c.xl
	}
	interview := model.InterviewDo{}
	err := c.interviewColl.Find(fields).One(&interview)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("no such room for fields %v", fields)
			return nil, &errors2.ServerError{Code: errors2.ServerErrorRoomNotFound}
		}
		xl.Errorf("failed to get room, error %v", fields)
		return nil, err
	}
	return &interview, nil
}

func (c *InterviewService) GetInterviewByID(xl *xlog.Logger, interviewID string) (*model.InterviewDo, error) {
	return c.GetInterviewByFields(xl, map[string]interface{}{"_id": interviewID})
}

func (c *InterviewService) UpdateInterview(xl *xlog.Logger, id string, interview *model.InterviewDo) (*model.InterviewDo, error) {
	if xl == nil {
		xl = c.xl
	}
	err := c.interviewColl.Update(bson.M{"_id": id}, bson.M{"$set": interview})
	if err != nil {
		xl.Errorf("failed to update interview %s,error %v", id, err)
		return nil, err
	}
	return interview, nil
}

func (c *InterviewService) JoinInterview(xl *xlog.Logger, userID string, interviewID string) ([]model.InterviewUserDo, []model.InterviewUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	interviewUserDo := &model.InterviewUserDo{
		ID:             interviewID + "_" + userID,
		InterviewID:    interviewID,
		UserID:         userID,
		Status:         2,
		LastModifyTime: time.Now(),
	}
	err := c.interviewUserColl.Update(bson.M{"userId": userID, "interviewId": interviewID}, bson.M{"$set": interviewUserDo})
	if err != nil {
		xl.Errorf("failed to update user status of user %s, error %v", userID, err)
		return nil, nil, err
	}
	xl.Infof("user %s JoinInterview %s", userID, interviewID)
	onlineInterviewUserDos, onlineUsersQueryErr := c.OnlineInterviewUsers(xl, userID, interviewID)
	if onlineUsersQueryErr != nil {
		xl.Errorf("failed to update user status of user %s, error %v", userID, err)
		return nil, nil, onlineUsersQueryErr
	}
	allInterviewUserDos, allUsersQueryErr := c.AllInterviewUsers(xl, userID, interviewID)
	if allUsersQueryErr != nil {
		xl.Errorf("failed to update user status of user %s, error %v", userID, err)
		return nil, nil, allUsersQueryErr
	}
	if len(onlineInterviewUserDos) > 1 {
		interview, err := c.GetInterviewByID(xl, interviewID)
		if err != nil {
			// TODO: 这里直接返回错误？
			if err == mgo.ErrNotFound {
				xl.Infof("room %s not found", interviewID)
			}
			xl.Errorf("failed to get room %s, error %v", interviewID, err)
		}
		interview.Status = int(model.InterviewStatusCodeStart)
		updateErr := c.interviewColl.Update(bson.M{"_id": interviewID}, bson.M{"$set": interview})
		if updateErr != nil {
			xl.Errorf("failed to update interview %s,error %v", interviewID, updateErr)
			return nil, nil, err
		}
	}
	return onlineInterviewUserDos, allInterviewUserDos, nil
}

func (c *InterviewService) LeaveInterview(xl *xlog.Logger, userID string, interviewID string) error {
	if xl == nil {
		xl = c.xl
	}
	_, err := c.GetInterviewByID(xl, interviewID)
	if err != nil {
		// TODO: 这里直接返回错误？
		if err == mgo.ErrNotFound {
			xl.Infof("room %s not found", interviewID)
		}
		xl.Errorf("failed to get room %s, error %v", interviewID, err)
	}

	// 修改用户状态为空闲。
	interviewUserDo := &model.InterviewUserDo{
		ID:             interviewID + "_" + userID,
		InterviewID:    interviewID,
		UserID:         userID,
		Status:         3,
		LastModifyTime: time.Now(),
	}
	interviewUserUpdateErr := c.interviewUserColl.Update(bson.M{"userId": userID, "interviewId": interviewID}, bson.M{"$set": interviewUserDo})
	if interviewUserUpdateErr != nil {
		xl.Errorf("failed to update user status of user %s, error %v", userID, interviewUserUpdateErr)
	}
	xl.Infof("user %s left room %s", userID, interviewID)
	return nil
}

func (c *InterviewService) OnlineInterviewUsers(xl *xlog.Logger, userID string, interviewID string) ([]model.InterviewUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	interviewUserDos := []model.InterviewUserDo{}
	err := c.interviewUserColl.Find(bson.M{"interviewId": interviewID, "$or": []bson.M{bson.M{"status": 2}}}).All(&interviewUserDos)
	return interviewUserDos, err
}

func (c *InterviewService) AllInterviewUsers(xl *xlog.Logger, userID string, interviewID string) ([]model.InterviewUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	interviewUserDos := []model.InterviewUserDo{}
	err := c.interviewUserColl.Find(bson.M{"interviewId": interviewID}).All(&interviewUserDos)
	return interviewUserDos, err
}

// HeartBeat mark user LastHeartBeat moment
func (c *InterviewService) HeartBeat(xl *xlog.Logger, userId, interviewID string) {
	if xl == nil {
		xl = c.xl
	}
	interview, err := c.GetInterviewByID(xl, interviewID)
	if err != nil || interview.Status == int(model.InterviewStatusCodeEnd) {
		return
	}
	var record model.InterviewUserDo
	condition := bson.M{"_id": interviewID + "_" + userId}
	err = c.interviewUserColl.Find(condition).One(&record)
	if err != nil {
		return
	}
	record.LastHeartBeatTime = time.Now()
	err = c.interviewUserColl.UpdateId(record.ID, record)
	if err != nil {
		xl.Errorf("update interview user do err:%v", err)
	}
	return
}

// ListHeartBeatTimeOutUser
func (c *InterviewService) ListHeartBeatTimeOutUser(xl *xlog.Logger) ([]model.InterviewUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	ddl := time.Now().Add((time.Duration(model.HeartBeatInterval) + 5) * time.Second * (-1))
	condition := bson.M{
		"lastHeartBeatTime": bson.M{
			"$lt": ddl,
		},
		"status": 2,
	}
	records := make([]model.InterviewUserDo, 0)
	err := c.interviewUserColl.Find(condition).All(&records)
	return records, err
}

func (c *InterviewService) Online(xl *xlog.Logger, interviewId, userId string) bool {
	if xl == nil {
		xl = c.xl
	}
	_, err := c.GetInterviewByID(xl, interviewId)
	if err != nil {
		return false
	}
	users, err := c.OnlineInterviewUsers(xl, userId, interviewId)
	if err != nil {
		return false
	}
	for _, u := range users {
		if u.UserID == userId {
			return true
		}
	}
	return false
}

func (c *InterviewService) GetRecordURL(xl *xlog.Logger, interviewId string) string {
	if xl == nil {
		xl = c.xl
	}
	condition := bson.M{
		"subject":    "interview",
		"action":     "record",
		"subject_id": interviewId,
	}
	var record model.TaskResultDo
	err := c.taskColl.Find(condition).One(&record)
	if err != nil {
		xl.Debugf("get interview record task err:%v", err)
		return ""
	}
	return record.Result
}
