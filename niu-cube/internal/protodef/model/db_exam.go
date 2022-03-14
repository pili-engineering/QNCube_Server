package model

import "time"

type ExamDo struct {
	Id   string `bson:"_id" json:"examId"`
	Name string `bson:"name" json:"examName"`
	// TODO 后端记得校验提交时间是否超时
	BgnTime  time.Time `bson:"bgn_time" json:"startTime"`
	EndTime  time.Time `bson:"end_time" json:"endTime"`
	Duration int64     `bson:"duration" json:"duration"`
	Type     string    `bson:"type" json:"type"`
	Desc     string    `bson:"desc" json:"desc"`
	Creator  string    `bson:"creator" json:"creator"`
	// InvigilatorList []string  `bson:"invigilator_list" json:"invigilatorList"`
	Status      int       `bson:"status" json:"status"`
	CreatedTime time.Time `bson:"created_time" json:"-"`
	UpdatedTime time.Time `bson:"updated_time" json:"-"`
}

// AnswerDo 每个Answer都是全新的答案，所以每次使用都是全新的实例
// TODO 未来会增加题目分类
type AnswerDo struct {
	QuestionId  string    `bson:"question_id" json:"questionId"`
	Type        string    `bson:"type" json:"type"`
	Score       float64   `bson:"score" json:"score"`
	ChoiceList  []string  `bson:"choice_list" json:"choiceList"`
	Judge       bool      `bson:"judge" json:"judge"`
	Text        string    `bson:"text" json:"text"`
	Status      int       `bson:"status" json:"status"`
	CreatedTime time.Time `bson:"created_time" json:"-"`
	UpdatedTime time.Time `bson:"updated_time" json:"-"`
}

// QuestionDo Question可以实现复用，所以有ID，方便查找
type QuestionDo struct {
	Id          string    `bson:"_id" json:"questionId"`
	Type        string    `bson:"type" json:"type"`
	Score       float64   `bson:"score" json:"score"`
	Desc        string    `bson:"desc" json:"desc"`
	ChoiceList  []string  `bson:"choice_list" json:"choiceList"`
	Answer      AnswerDo  `bson:"answer" json:"-"`
	Status      int       `bson:"status" json:"status"`
	CreatedTime time.Time `bson:"created_time" json:"-"`
	UpdatedTime time.Time `bson:"updated_time" json:"-"`
}

type ExamPaperDo struct {
	Id           string    `bson:"_id" json:"examPaperId"`
	Name         string    `bson:"name" json:"paperName"`
	ExamId       string    `bson:"exam_id" json:"examId"`
	QuestionList []string  `bson:"question_list" json:"questionList"`
	TotalScore   int       `bson:"total_score" json:"totalScore"`
	Status       int       `bson:"status" json:"status"`
	CreatedTime  time.Time `bson:"created_time" json:"-"`
	UpdatedTime  time.Time `bson:"updated_time" json:"-"`
}

type UserExamDo struct {
	Id          string    `bson:"_id" json:"-"`
	UserId      string    `bson:"user_id" json:"userId"`
	ExamId      string    `bson:"exam_id" json:"examId"`
	ExamPaperId string    `bson:"exam_paper_id" json:"examPaperId"`
	RoomId      string    `bson:"room_id" json:"roomId"`
	Status      int       `bson:"status" json:"status"`
	CreatedTime time.Time `bson:"created_time" json:"-"`
	UpdatedTime time.Time `bson:"updated_time" json:"-"`
}

type AnswerPaperDo struct {
	Id          string     `bson:"_id" json:"id"`
	UserId      string     `bson:"user_id" json:"userId"`
	ExamId      string     `bson:"exam_id" json:"examId"`
	AnswerList  []AnswerDo `bson:"answer_list" json:"answerList"`
	TotalScore  float64    `bson:"total_score" json:"totalScore"`
	Status      int        `bson:"status" json:"status"`
	CreatedTime time.Time  `bson:"created_time" json:"createdTime"`
	UpdatedTime time.Time  `bson:"updated_time" json:"updatedTime"`
}

type CheatingEvent struct {
	Id        string `bson:"_id" json:"-"`
	UserId    string `bson:"user_id" json:"userId"`
	ExamId    string `bson:"exam_id" json:"examId"`
	Action    string `bson:"action" json:"action"`
	Value     string `bson:"value" json:"value"`
	Timestamp int64  `bson:"timestamp" json:"timestamp"`
}

const (
	_ = iota
	ExamCreated
	ExamInProgress
	ExamFinished
	ExamDestroyed
)

const (
	_ = iota
	AnswerAvailable
	AnswerUnavailable
)

const (
	_ = iota
	QuestionAvailable
	QuestionUnavailable
)

const (
	_ = iota
	ExamPaperAvailable
	ExamPaperUnAvailable
)

const (
	_ = iota
	UserExamToBeInvolved
	UserExamInProgress
	UserExamFinished
	UserExamDestroyed
)

const (
	_ = iota
	AnswerPaperAvailable
	AnswerPaperUnavailable
)

const (
	SingleChoice = "single_choice"
	MultiChoice  = "multi_choice"
	Judge        = "judge"
	Text         = "text"
)

const (
	_ = iota
	UserStatusOk
	UserStatusNotExaminee
	UserStatusNotSelf
	UserStatusOtherBrowser
)
