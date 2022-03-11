package task

import (
	model "github.com/solutions/niu-cube/internal/protodef/model"
	dao "github.com/solutions/niu-cube/internal/service/db/dao"
	"time"

	"github.com/qiniu/x/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type InterviewTask struct {
	mongoClient   *mgo.Session
	interviewColl *mgo.Collection
}

func NewInterviewTask(mongoURI string, database string) (*InterviewTask, error) {
	mongoClient, err := mgo.Dial(mongoURI + "/" + database)
	if err != nil {
		log.Errorf("failed to create mongo client, error %v", err)
		return nil, err
	}
	interviewColl := mongoClient.DB(database).C(dao.InterviewCollection)
	return &InterviewTask{
		mongoClient:   mongoClient,
		interviewColl: interviewColl,
	}, nil
}

func (c *InterviewTask) ListTaskInterviews(dataSize int) ([]model.InterviewDo, error) {
	if dataSize <= 0 {
		dataSize = 10
	}
	interviews := []model.InterviewDo{}
	err := c.interviewColl.Find(bson.M{"$or": []bson.M{bson.M{"status": model.InterviewStatusCodeInit}, bson.M{"status": model.InterviewStatusCodeStart}}}).Sort("startTime").Limit(dataSize).All(&interviews)
	if err != nil {
		log.Errorf("failed to ListTaskInterviews , error %v", err)
		return nil, err
	}
	return interviews, err
}

func (c *InterviewTask) UpdateInterview(interview *model.InterviewDo) (*model.InterviewDo, error) {
	err := c.interviewColl.Update(bson.M{"_id": interview.ID}, bson.M{"$set": interview})
	if err != nil {
		log.Errorf("failed to update interview %s,error %v", interview.ID, err)
		return nil, err
	}
	return interview, nil
}

func (t *InterviewTask) TaskForModifyInterviewStatus() {
	log.Infof("taskForModifyInterviewStatus run at %s", time.Now().String())

	interviews, err := t.ListTaskInterviews(10)
	if err != nil {
		log.Errorf("TaskForModifyInterviewStatus find interviews, error: %v", err)
		return
	}
	if len(interviews) <= 0 {
		log.Infof("taskForModifyInterviewStatus find no interviews")
	}
	for _, interview := range interviews {
		d, _ := time.ParseDuration("-24h")
		if time.Now().Add(d).After(interview.CreateTime) {
			log.Infof("TaskForModifyInterviewStatus modify status for interview %s, status: %d, startTime: %s", interview.ID, interview.Status, interview.StartTime)
			interview.Status = int(model.InterviewStatusCodeEnd)
			_, err := t.UpdateInterview(&interview)
			if err != nil {
				log.Errorf("TaskForModifyInterviewStatus modify err, %v", err)
			}
		}
	}

}
