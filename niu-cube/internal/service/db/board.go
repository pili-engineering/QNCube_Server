package db

import (
	"fmt"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"gopkg.in/mgo.v2"
	"math/rand"
)

var (
	DefaultBoardService *BoardService
)

type BoardService struct {
	boardCollection *mgo.Collection
	xl              *xlog.Logger
}

func NewBoardService(xl *xlog.Logger, config utils.MongoConfig) *BoardService {
	v := new(BoardService)
	v.xl = xlog.New("board service")
	db, err := mgo.Dial(config.URI)
	if err != nil {
		v.xl.Fatalf("error dialing service error:%v", err)
	}
	err = db.Ping()
	if err != nil {
		v.xl.Fatalf("err ping db error:%v", err)
	}
	v.boardCollection = db.DB(config.Database).C("boards")
	return v
}

// CRUD

func (v *BoardService) Create(xl *xlog.Logger, board model.BoardDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if board.ID == "" {
		switch board.InterviewID {
		case "":
			return fmt.Errorf("interviewId is needed")
		default:
			board.ID = board.InterviewID
		}
	}
	err := v.boardCollection.Insert(board)
	if err != nil {
		logger.Errorf("error create boardDo %v err:%v", board, err)
	}
	return err
}

func (v *BoardService) Upsert(xl *xlog.Logger, board model.BoardDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if board.ID == "" {
		switch board.InterviewID {
		case "":
			return fmt.Errorf("interviewId is needed")
		default:
			board.ID = board.InterviewID
		}
	}
	info, err := v.boardCollection.UpsertId(board.ID, board)
	if err != nil {
		logger.Errorf("error upsert boardDo %v err:%v", board, err)
	}
	if info.UpsertedId != nil {
		logger.Infof("successfully create board %v", info.UpsertedId)
	} else {
		logger.Infof("modify board %v ,update status:%v", info.Matched, info.Updated)
	}
	return err
}

func (v *BoardService) Update(xl *xlog.Logger, board model.BoardDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.boardCollection.UpdateId(board.ID, board)
	if err != nil {
		logger.Errorf("error update boardDo %v err:%v", board, err)
	}
	return err
}

func (v *BoardService) Delete(xl *xlog.Logger, id string) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.boardCollection.RemoveId(id)
	if err != nil {
		logger.Errorf("error update boardId %v err:%v", id, err)
	}
	return err
}

func (c *BoardService) GetOneByID(xl *xlog.Logger, id string) (model.BoardDo, error) {
	return c.GetOneByMap(xl, map[string]interface{}{"_id": id})
}

func (v *BoardService) GetOneByMap(xl *xlog.Logger, filter interface{}) (model.BoardDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	var board model.BoardDo
	err := v.boardCollection.Find(filter).One(&board)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return board, err
	}
	return board, err
}

// GetByMap
func (v *BoardService) GetByMap(xl *xlog.Logger, filter interface{}) ([]model.BoardDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	boards := make([]model.BoardDo, 0)
	var err error
	err = v.boardCollection.Find(filter).All(&boards)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return boards, err
	}
	return boards, err
}

// GetPageByMap
func (v *BoardService) GetPageByMap(xl *xlog.Logger, filter interface{}, pageNum, pageSize int) ([]model.BoardDo, int, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	boards := make([]model.BoardDo, 0)
	var err error
	err = v.boardCollection.Find(filter).Skip((pageNum - 1) * pageSize).Limit(pageSize).All(&boards)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return boards, 0, err
	}
	cnt, err := v.boardCollection.Find(filter).Count()
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return boards, 0, err
	}
	return boards, cnt, err
}

// generateID utils func: for 12-digit random id generation
func (v *BoardService) generateID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	idLength := 12
	id := ""
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		id = id + string(alphaNum[index])
	}
	return id
}

func (v *BoardService) Permit(action string, userId string) bool {
	return true
}
