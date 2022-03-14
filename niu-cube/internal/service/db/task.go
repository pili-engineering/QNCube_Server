package db

import (
	"fmt"
	"math/rand"

	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
)

var (
	DefaultTaskService *TaskService
)

type TaskService struct {
	taskCollection *mgo.Collection
	xl             *xlog.Logger
}

func NewTaskService(xl *xlog.Logger, config utils.MongoConfig) *TaskService {
	v := new(TaskService)
	v.xl = xlog.New("task db")
	db, err := mgo.Dial(config.URI)
	if err != nil {
		v.xl.Fatalf("error dialing db error:%v", err)
	}
	err = db.Ping()
	if err != nil {
		v.xl.Fatalf("err ping db error:%v", err)
	}
	v.taskCollection = db.DB(config.Database).C(dao.TaskCollection)
	return v
}

// CRUD

func (v *TaskService) Create(xl *xlog.Logger, task model.TaskResultDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if task.ID == "" {
		switch task.SubjectID {
		case "":
			return fmt.Errorf("subjectID is needed")
		default:
			task.ID = task.SubjectID
		}
	}
	err := v.taskCollection.Insert(task)
	if err != nil {
		logger.Errorf("error create TaskResultDo %v err:%v", task, err)
	}
	return err
}

func (v *TaskService) Upsert(xl *xlog.Logger, task model.TaskResultDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if task.ID == "" {
		switch task.SubjectID {
		case "":
			return fmt.Errorf("subjectID is needed")
		default:
			task.ID = task.SubjectID
		}
	}
	info, err := v.taskCollection.UpsertId(task.ID, task)
	if err != nil {
		logger.Errorf("error upsert TaskResultDo %v err:%v", task, err)
	}
	if info.UpsertedId != nil {
		logger.Infof("successfully create task %v", info.UpsertedId)
	} else {
		logger.Infof("modify task %v ,update status:%v", info.Matched, info.Updated)
	}
	return err
}

func (v *TaskService) Update(xl *xlog.Logger, task model.TaskResultDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.taskCollection.UpdateId(task.ID, task)
	if err != nil {
		logger.Errorf("error update TaskResultDo %v err:%v", task, err)
	}
	return err
}

func (v *TaskService) Delete(xl *xlog.Logger, id string) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.taskCollection.RemoveId(id)
	if err != nil {
		logger.Errorf("error update taskId %v err:%v", id, err)
	}
	return err
}

func (c *TaskService) GetOneByID(xl *xlog.Logger, id string) (model.TaskResultDo, error) {
	return c.GetOneByMap(xl, map[string]interface{}{"_id": id})
}

func (c *TaskService) GetTask(xl *xlog.Logger, subject, action, subjectId string) (model.TaskResultDo, error) {
	condition := bson.M{
		"subject":    subject,
		"action":     action,
		"subject_id": subjectId,
	}
	return c.GetOneByMap(xl, condition)
}

func (v *TaskService) GetOneByMap(xl *xlog.Logger, filter interface{}) (model.TaskResultDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	var task model.TaskResultDo
	err := v.taskCollection.Find(filter).One(&task)
	if err != nil && err != mgo.ErrNotFound {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return task, err
	}
	return task, err
}

func (v *TaskService) GetByMap(xl *xlog.Logger, filter interface{}) ([]model.TaskResultDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	tasks := make([]model.TaskResultDo, 0)
	var err error
	err = v.taskCollection.Find(filter).All(&tasks)
	if err != nil && err != mgo.ErrNotFound {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return tasks, err
	}
	return tasks, err
}

// GetPageByMap
func (v *TaskService) GetPageByMap(xl *xlog.Logger, filter interface{}, pageNum, pageSize int) ([]model.TaskResultDo, int, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	tasks := make([]model.TaskResultDo, 0)
	var err error
	err = v.taskCollection.Find(filter).Skip((pageNum - 1) * pageSize).Limit(pageSize).All(&tasks)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return tasks, 0, err
	}
	cnt, err := v.taskCollection.Find(filter).Count()
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return tasks, 0, err
	}
	return tasks, cnt, err
}

// generateID utils func: for 12-digit random id generation
func (v *TaskService) generateID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	idLength := 12
	id := ""
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		id = id + string(alphaNum[index])
	}
	return id
}

func (v *TaskService) Permit(action string, userId string) bool {
	return true
}
