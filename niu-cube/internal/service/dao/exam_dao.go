package dao

import (
	"context"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type ExamDao interface {
	Insert(exam *model.ExamDo) error

	Select(id string) (*model.ExamDo, error)

	ListByCreator0(creator string) ([]model.ExamDo, error)

	ListAll0() ([]model.ExamDo, error)

	ListAll(pgNum, pgSize int64) ([]model.ExamDo, int64, error)

	Update(exam *model.ExamDo) error

	Delete0(id string) error

	Delete(id string) error

	DeleteAll0() error

	DeleteAll() error
}

type QuestionDao interface {
	Insert(question *model.QuestionDo) error

	Select(id string) (*model.QuestionDo, error)

	Update(question *model.QuestionDo) error

	ListByType0(t string) ([]model.QuestionDo, error)

	ListAll0() ([]model.QuestionDo, int64, error)

	ListAll(pgNum, pgSize int64) ([]model.QuestionDo, int64, error)

	Delete0(id string) error

	Delete(id string) error
}

type ExamPaperDao interface {
	Insert(examPaper *model.ExamPaperDo) error

	Select(id string) (*model.ExamPaperDo, error)

	ListByExamId(examId string) ([]model.ExamPaperDo, error)

	Update(examPaper *model.ExamPaperDo) error

	Delete0(id string) error

	Delete(id string) error

	DeleteAll() error
}

type UserExamDao interface {
	Insert(userExam *model.UserExamDo) error

	Select(id string) (*model.UserExamDo, error)

	SelectByExamIdUserId(examId, userId string) (*model.UserExamDo, error)

	ListByExamId0(examId string) ([]model.UserExamDo, error)

	ListByExamId(examId string, pgNum, pgSize int64) ([]model.UserExamDo, int64, error)

	ListByUserId0(userId string) ([]model.UserExamDo, error)

	ListByUserId(userId string, pgNum, pgSize int64) ([]model.UserExamDo, int64, error)

	ListAll0() ([]model.UserExamDo, error)

	ListAll(pgNum, pgSize int64) ([]model.UserExamDo, int64, error)

	Update(userExam *model.UserExamDo) error

	Delete0(id string) error

	Delete(id string) error

	DeleteAll() error
}

type AnswerPaperDao interface {
	Insert(answerPaper *model.AnswerPaperDo) error

	Select(id string) (*model.AnswerPaperDo, error)

	SelectByExamIdUserId(examId, userId string) (*model.AnswerPaperDo, error)

	ListByExamId0(examId string) ([]model.AnswerPaperDo, error)

	Update(answerPaper *model.AnswerPaperDo) error

	Delete0(id string) error

	Delete(id string) error

	DeleteAll() error
}

type CheatingEventDao interface {
	Insert(cheatingEvent *model.CheatingEvent) error

	ListByExamIdUserId(examId, userId string, afterTimestamp, beforeTimestamp int64) ([]model.CheatingEvent, error)
}

type ExamDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewExamDaoService(config *utils.MongoConfig) *ExamDaoService {
	logger := xlog.New("exam dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionExam)
	return &ExamDaoService{
		collection,
		logger,
	}
}

func (e *ExamDaoService) Insert(exam *model.ExamDo) error {
	exam.Id = primitive.NewObjectID().Hex()
	exam.Status = model.ExamCreated
	exam.CreatedTime = time.Now()
	exam.UpdatedTime = time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.InsertOne(timeout, exam)
	if err != nil {
		e.logger.Errorf("插入数据失败: %v", err)
		return err
	}
	return nil
}

func (e *ExamDaoService) Select(id string) (*model.ExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := e.collection.FindOne(timeout, primitive.M{"_id": id, "status": primitive.M{"$ne": model.ExamDestroyed}})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		e.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.ExamDo{}
	err := one.Decode(&result)
	if err != nil {
		e.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (e *ExamDaoService) ListByCreator0(creator string) ([]model.ExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := e.collection.Find(timeout, primitive.M{"creator": creator}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		e.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			e.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.ExamDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.ExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			e.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		e.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (e *ExamDaoService) ListAll0() ([]model.ExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := e.collection.Find(timeout, primitive.M{"status": primitive.M{"$ne": model.ExamDestroyed}}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		e.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			e.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.ExamDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.ExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			e.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		e.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (e *ExamDaoService) ListAll(pgNum, pgSize int64) ([]model.ExamDo, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	skip := (pgNum - 1) * pgSize
	cursor, err := e.collection.Find(ctx, primitive.M{"status": primitive.M{"$ne": model.ExamDestroyed}}, &options.FindOptions{
		Limit: &pgSize,
		Skip:  &skip,
		Sort:  primitive.M{"created_time": -1},
	})
	if err != nil {
		e.logger.Error(err)
		return nil, 0, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			e.logger.Error(err)
		}
	}(cursor, ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]model.ExamDo, 0, pgSize)
	for cursor.Next(ctx) {
		tmp := model.ExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			e.logger.Error(err)
			return nil, 0, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		e.logger.Error(err)
		return nil, 0, err
	}
	total, err := e.collection.CountDocuments(ctx, primitive.M{"status": primitive.M{"$ne": model.ExamDestroyed}})
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (e *ExamDaoService) Update(exam *model.ExamDo) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.UpdateByID(timeout, exam.Id, primitive.M{"$set": exam})
	return err
}

func (e *ExamDaoService) Delete0(id string) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.DeleteOne(timeout, primitive.M{"_id": id})
	return err
}

func (e *ExamDaoService) Delete(id string) error {
	exam, err := e.Select(id)
	if err != nil {
		return err
	}
	exam.UpdatedTime = time.Now()
	exam.Status = model.ExamDestroyed
	return e.Update(exam)
}

func (e *ExamDaoService) DeleteAll0() error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.DeleteMany(timeout, primitive.M{})
	return err
}

func (e *ExamDaoService) DeleteAll() error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()
	_, err := e.collection.UpdateMany(timeout, primitive.M{}, primitive.M{"$set": primitive.M{"status": model.ExamDestroyed}})
	return err
}

type QuestionDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewQuestionDaoService(config *utils.MongoConfig) *QuestionDaoService {
	logger := xlog.New("question dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionQuestion)
	return &QuestionDaoService{
		collection,
		logger,
	}
}

func (q *QuestionDaoService) Insert(question *model.QuestionDo) error {
	question.Id = primitive.NewObjectID().Hex()
	question.Status = model.QuestionAvailable
	question.CreatedTime = time.Now()
	question.UpdatedTime = time.Now()
	question.Answer.QuestionId = question.Id
	question.Answer.Status = model.AnswerAvailable
	question.Answer.CreatedTime = time.Now()
	question.Answer.UpdatedTime = time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := q.collection.InsertOne(timeout, question)
	if err != nil {
		q.logger.Errorf("插入数据失败: %v", err)
		return err
	}
	return nil
}

func (q *QuestionDaoService) Select(id string) (*model.QuestionDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := q.collection.FindOne(timeout, primitive.M{"_id": id})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		q.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.QuestionDo{}
	err := one.Decode(&result)
	if err != nil {
		q.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (q *QuestionDaoService) Update(question *model.QuestionDo) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := q.collection.UpdateByID(timeout, question.Id, primitive.M{"$set": question})
	return err
}

func (q *QuestionDaoService) ListByType0(t string) ([]model.QuestionDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := q.collection.Find(timeout, primitive.M{"type": t}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		q.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			q.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.QuestionDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.QuestionDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			q.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		q.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (q *QuestionDaoService) ListAll0() ([]model.QuestionDo, int64, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := q.collection.Find(timeout, primitive.M{"status": model.QuestionAvailable}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, 0, nil
		}
		q.logger.Error(err)
		return nil, 0, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			q.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.QuestionDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.QuestionDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			q.logger.Error(err)
			return nil, 0, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		q.logger.Error(err)
		return nil, 0, err
	}
	total, err := q.collection.CountDocuments(timeout, primitive.M{})
	if err != nil {
		q.logger.Error(err)
		return nil, 0, err
	}
	return results, total, nil
}

func (q *QuestionDaoService) ListAll(pgNum, pgSize int64) ([]model.QuestionDo, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	skip := (pgNum - 1) * pgSize
	cursor, err := q.collection.Find(ctx, primitive.M{"status": model.QuestionAvailable}, &options.FindOptions{
		Limit: &pgSize,
		Skip:  &skip,
		Sort:  primitive.M{"created_time": -1},
	})
	if err != nil {
		q.logger.Error(err)
		return nil, 0, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			q.logger.Error(err)
		}
	}(cursor, ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]model.QuestionDo, 0, pgSize)
	for cursor.Next(ctx) {
		tmp := model.QuestionDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			q.logger.Error(err)
			return nil, 0, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		q.logger.Error(err)
		return nil, 0, err
	}
	total, err := q.collection.CountDocuments(ctx, primitive.M{})
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (q *QuestionDaoService) Delete0(id string) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := q.collection.DeleteOne(timeout, primitive.M{"_id": id})
	return err
}

func (q *QuestionDaoService) Delete(id string) error {
	question, err := q.Select(id)
	if err != nil {
		return err
	}
	question.UpdatedTime = time.Now()
	question.Status = model.QuestionUnavailable
	return q.Update(question)
}

type ExamPaperDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewExamPaperDaoService(config *utils.MongoConfig) *ExamPaperDaoService {
	logger := xlog.New("exam dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionExamPaper)
	return &ExamPaperDaoService{
		collection,
		logger,
	}
}

func (e *ExamPaperDaoService) Insert(examPaper *model.ExamPaperDo) error {
	examPaper.Id = primitive.NewObjectID().Hex()
	examPaper.Status = model.ExamPaperAvailable
	examPaper.CreatedTime = time.Now()
	examPaper.UpdatedTime = time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.InsertOne(timeout, examPaper)
	if err != nil {
		e.logger.Errorf("插入数据失败: %v", err)
		return err
	}
	return nil
}

func (e *ExamPaperDaoService) Select(id string) (*model.ExamPaperDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := e.collection.FindOne(timeout, primitive.M{"_id": id})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		e.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.ExamPaperDo{}
	err := one.Decode(&result)
	if err != nil {
		e.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (e *ExamPaperDaoService) ListByExamId(examId string) ([]model.ExamPaperDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := e.collection.Find(timeout, primitive.M{"exam_id": examId, "status": model.ExamPaperAvailable}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		e.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			e.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.ExamPaperDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.ExamPaperDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			e.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		e.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (e *ExamPaperDaoService) Update(examPaper *model.ExamPaperDo) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.UpdateByID(timeout, examPaper.Id, primitive.M{"$set": examPaper})
	return err
}

func (e *ExamPaperDaoService) Delete0(id string) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := e.collection.DeleteOne(timeout, primitive.M{"_id": id})
	return err
}

func (e *ExamPaperDaoService) Delete(id string) error {
	examPaper, err := e.Select(id)
	if err != nil {
		return err
	}
	examPaper.UpdatedTime = time.Now()
	examPaper.Status = model.QuestionUnavailable
	return e.Update(examPaper)
}

func (e *ExamPaperDaoService) DeleteAll() error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()
	_, err := e.collection.UpdateMany(timeout, primitive.M{}, primitive.M{"$set": primitive.M{"status": model.ExamPaperUnAvailable}})
	return err
}

type UserExamDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewUserExamDaoService(config *utils.MongoConfig) *UserExamDaoService {
	logger := xlog.New("exam dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionUserExam)
	return &UserExamDaoService{
		collection,
		logger,
	}
}

func (u *UserExamDaoService) Insert(userExam *model.UserExamDo) error {
	userExam.Id = primitive.NewObjectID().Hex()
	userExam.Status = model.UserExamToBeInvolved
	userExam.CreatedTime = time.Now()
	userExam.UpdatedTime = time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := u.collection.InsertOne(timeout, userExam)
	if err != nil {
		u.logger.Errorf("插入数据失败: %v", err)
		return err
	}
	return nil
}

func (u *UserExamDaoService) Select(id string) (*model.UserExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := u.collection.FindOne(timeout, primitive.M{"_id": id})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		u.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.UserExamDo{}
	err := one.Decode(&result)
	if err != nil {
		u.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (u *UserExamDaoService) SelectByExamIdUserId(examId, userId string) (*model.UserExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := u.collection.FindOne(timeout, primitive.M{"exam_id": examId, "user_id": userId, "status": primitive.M{"$ne": model.ExamDestroyed}})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		u.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.UserExamDo{}
	err := one.Decode(&result)
	if err != nil {
		u.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (u *UserExamDaoService) ListByExamId0(examId string) ([]model.UserExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := u.collection.Find(timeout, primitive.M{"exam_id": examId}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		u.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			u.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.UserExamDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.UserExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			u.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		u.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (u *UserExamDaoService) ListByExamId(examId string, pgNum, pgSize int64) ([]model.UserExamDo, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	skip := (pgNum - 1) * pgSize
	cursor, err := u.collection.Find(ctx, primitive.M{"exam_id": examId, "status": model.UserExamInProgress}, &options.FindOptions{
		Limit: &pgSize,
		Skip:  &skip,
		Sort:  primitive.M{"created_time": -1},
	})
	if err != nil {
		u.logger.Error(err)
		return nil, 0, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			u.logger.Error(err)
		}
	}(cursor, ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]model.UserExamDo, 0, pgSize)
	for cursor.Next(ctx) {
		tmp := model.UserExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			u.logger.Error(err)
			return nil, 0, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		u.logger.Error(err)
		return nil, 0, err
	}
	total, err := u.collection.CountDocuments(ctx, primitive.M{"exam_id": examId, "status": model.UserExamInProgress})
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (u *UserExamDaoService) ListByUserId0(userId string) ([]model.UserExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := u.collection.Find(timeout, primitive.M{"user_id": userId}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		u.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			u.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.UserExamDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.UserExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			u.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		u.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (u *UserExamDaoService) ListByUserId(userId string, pgNum, pgSize int64) ([]model.UserExamDo, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	skip := (pgNum - 1) * pgSize
	cursor, err := u.collection.Find(ctx, primitive.M{"user_id": userId, "status": primitive.M{"$ne": model.ExamDestroyed}}, &options.FindOptions{
		Limit: &pgSize,
		Skip:  &skip,
		Sort:  primitive.M{"created_time": -1},
	})
	if err != nil {
		u.logger.Error(err)
		return nil, 0, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			u.logger.Error(err)
		}
	}(cursor, ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]model.UserExamDo, 0, pgSize)
	for cursor.Next(ctx) {
		tmp := model.UserExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			u.logger.Error(err)
			return nil, 0, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		u.logger.Error(err)
		return nil, 0, err
	}
	total, err := u.collection.CountDocuments(ctx, primitive.M{"user_id": userId, "status": primitive.M{"$ne": model.ExamDestroyed}})
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (u *UserExamDaoService) ListAll0() ([]model.UserExamDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := u.collection.Find(timeout, primitive.M{}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		u.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			u.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.UserExamDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.UserExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			u.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		u.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (u *UserExamDaoService) ListAll(pgNum, pgSize int64) ([]model.UserExamDo, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	skip := (pgNum - 1) * pgSize
	cursor, err := u.collection.Find(ctx, primitive.M{"status": primitive.M{"$ne": model.ExamDestroyed}}, &options.FindOptions{
		Limit: &pgSize,
		Skip:  &skip,
		Sort:  primitive.M{"created_time": -1},
	})
	if err != nil {
		u.logger.Error(err)
		return nil, 0, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			u.logger.Error(err)
		}
	}(cursor, ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]model.UserExamDo, 0, pgSize)
	for cursor.Next(ctx) {
		tmp := model.UserExamDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			u.logger.Error(err)
			return nil, 0, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		u.logger.Error(err)
		return nil, 0, err
	}
	total, err := u.collection.CountDocuments(ctx, primitive.M{"status": primitive.M{"$ne": model.ExamDestroyed}})
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (u *UserExamDaoService) Update(userExam *model.UserExamDo) error {
	userExam.UpdatedTime = time.Now()
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := u.collection.UpdateByID(timeout, userExam.Id, primitive.M{"$set": userExam})
	return err
}

func (u *UserExamDaoService) Delete0(id string) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := u.collection.DeleteOne(timeout, primitive.M{"_id": id})
	return err
}

func (u *UserExamDaoService) Delete(id string) error {
	userExam, err := u.Select(id)
	if err != nil {
		return err
	}
	userExam.UpdatedTime = time.Now()
	userExam.Status = model.QuestionUnavailable
	return u.Update(userExam)
}

func (u *UserExamDaoService) DeleteAll() error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()
	_, err := u.collection.UpdateMany(timeout, primitive.M{}, primitive.M{"$set": primitive.M{"status": model.UserExamDestroyed}})
	return err
}

type AnswerPaperDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewAnswerPaperDaoService(config *utils.MongoConfig) *AnswerPaperDaoService {
	logger := xlog.New("exam dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionAnswerPaper)
	return &AnswerPaperDaoService{
		collection,
		logger,
	}
}

func (a *AnswerPaperDaoService) Insert(answerPaper *model.AnswerPaperDo) error {
	answerPaper.Id = primitive.NewObjectID().Hex()
	answerPaper.Status = model.AnswerPaperAvailable
	answerPaper.CreatedTime = time.Now()
	answerPaper.UpdatedTime = time.Now()
	for idx := range answerPaper.AnswerList {
		answerPaper.AnswerList[idx].Status = model.AnswerAvailable
		answerPaper.AnswerList[idx].CreatedTime = time.Now()
		answerPaper.AnswerList[idx].UpdatedTime = time.Now()
	}
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := a.collection.InsertOne(timeout, answerPaper)
	if err != nil {
		a.logger.Errorf("插入数据失败: %v", err)
		return err
	}
	return nil
}

func (a *AnswerPaperDaoService) Select(id string) (*model.AnswerPaperDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := a.collection.FindOne(timeout, primitive.M{"_id": id})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		a.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.AnswerPaperDo{}
	err := one.Decode(&result)
	if err != nil {
		a.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (a *AnswerPaperDaoService) SelectByExamIdUserId(examId, userId string) (*model.AnswerPaperDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	one := a.collection.FindOne(timeout, primitive.M{"exam_id": examId, "user_id": userId})
	if err := one.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		a.logger.Errorf("查询数据表失败: %v", err)
		return nil, err
	}
	result := model.AnswerPaperDo{}
	err := one.Decode(&result)
	if err != nil {
		a.logger.Error(err)
		return nil, err
	}
	return &result, nil
}

func (a *AnswerPaperDaoService) ListByExamId0(examId string) ([]model.AnswerPaperDo, error) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	cursor, err := a.collection.Find(timeout, primitive.M{"exam_id": examId}, &options.FindOptions{
		Sort: primitive.M{"created_time": -1},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		a.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			a.logger.Error(err)
		}
	}(cursor, timeout)
	results := make([]model.AnswerPaperDo, 0, 10)
	for cursor.Next(timeout) {
		tmp := model.AnswerPaperDo{}
		err := cursor.Decode(&tmp)
		if err != nil {
			a.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		a.logger.Error(err)
		return nil, err
	}
	return results, nil
}

func (a *AnswerPaperDaoService) Update(answerPaper *model.AnswerPaperDo) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := a.collection.UpdateByID(timeout, answerPaper.Id, primitive.M{"$set": answerPaper})
	return err
}

func (a *AnswerPaperDaoService) Delete0(id string) error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := a.collection.DeleteOne(timeout, primitive.M{"_id": id})
	return err
}

func (a *AnswerPaperDaoService) Delete(id string) error {
	answerPaper, err := a.Select(id)
	if err != nil {
		return err
	}
	answerPaper.UpdatedTime = time.Now()
	answerPaper.Status = model.QuestionUnavailable
	return a.Update(answerPaper)
}

func (a *AnswerPaperDaoService) DeleteAll() error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()
	_, err := a.collection.UpdateMany(timeout, primitive.M{}, primitive.M{"$set": primitive.M{"status": model.AnswerPaperUnavailable}})
	return err
}

type CheatingEventDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewCheatingEventDaoService(config *utils.MongoConfig) *CheatingEventDaoService {
	logger := xlog.New("cheating event dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionCheatingExam)
	return &CheatingEventDaoService{
		collection,
		logger,
	}
}

func (c *CheatingEventDaoService) Insert(cheatingEvent *model.CheatingEvent) error {
	cheatingEvent.Id = primitive.NewObjectID().Hex()
	cheatingEvent.Timestamp = time.Now().UnixMilli()
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := c.collection.InsertOne(timeout, cheatingEvent)
	if err != nil {
		c.logger.Errorf("插入数据失败: %v", err)
		return err
	}
	return nil
}

func (c *CheatingEventDaoService) ListByExamIdUserId(examId, userId string, afterTimestamp, beforeTimestamp int64) ([]model.CheatingEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	match := primitive.M{"$match": primitive.D{
		primitive.E{Key: "exam_id", Value: examId},
		primitive.E{Key: "user_id", Value: userId},
		primitive.E{Key: "timestamp", Value: primitive.M{"$gt": afterTimestamp}},
		primitive.E{Key: "timestamp", Value: primitive.M{"$lte": beforeTimestamp}},
	}}
	group := primitive.M{"$group": primitive.M{"_id": "$action",
		"user_id":   primitive.M{"$last": "$user_id"},
		"exam_id":   primitive.M{"$last": "$exam_id"},
		"action":    primitive.M{"$last": "$action"},
		"value":     primitive.M{"$last": "$value"},
		"timestamp": primitive.M{"$last": "$timestamp"},
	}}
	sort := primitive.M{"$sort": primitive.M{"timestamp": -1}}
	cursor, err := c.collection.Aggregate(ctx, []primitive.M{match, group, sort})
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			c.logger.Error(err)
		}
	}(cursor, ctx)
	if err != nil {
		return nil, err
	}
	results := make([]model.CheatingEvent, 0, 10)
	for cursor.Next(ctx) {
		tmp := model.CheatingEvent{}
		err := cursor.Decode(&tmp)
		if err != nil {
			c.logger.Error(err)
			return nil, err
		}
		results = append(results, tmp)
	}
	if err := cursor.Err(); err != nil {
		c.logger.Error(err)
		return nil, err
	}
	return results, nil
}
