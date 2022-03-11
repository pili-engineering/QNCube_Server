package db

import (
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/cloud"
	"gopkg.in/mgo.v2"
	"sync"
	"time"
)

type InterviewService interface {

	// Logic Action
	Create(creator model.Account, candidate model.Account, interviewId string, bizExtra map[string]interface{}) error
	Join(account model.Account, interviewId string) error
	Leave(account model.Account, interviewId string) error
	Cancel(account model.Account, interviewId string) error
	End(account model.Account, interviewId string) error
	Share(account model.Account, interviewId string, ua string) (interface{}, error)
	Update(account model.Account, interviewId string) error
	List(account model.Account, pageNum, pageSize int) (interface{}, error)

	// CRUD
	//	ListOnlineUser(interviewId string)([]*model.RoomAccount,error)

}

type InterviewServiceImpl struct {
	roomColl        *mgo.Collection
	roomAccountColl *mgo.Collection
	roomService     cloud.RoomService

	JoinCheck func() error
}

func NewInterviewService(roomService cloud.RoomService) InterviewService {
	//client,err:=mgo.Dial(common.GetConf().Mongo.URI)
	i := &InterviewServiceImpl{
		roomService: roomService,
	}
	return i
}

func (i InterviewServiceImpl) Leave(account model.Account, interviewId string) error {
	switch {
	case i.IsCreatorOrCandidate(account.ID, interviewId):
		return i.roomService.Leave(account.ID, interviewId)
	default:
		return nil //permission error
	}
}

func (i InterviewServiceImpl) Cancel(account model.Account, interviewId string) error {
	// TODO： replace estimate with function to fasten
	switch {
	case i.IsCreatorOrCandidate(account.ID, interviewId):
		err := i.roomService.KickAllUser(interviewId)
		return err
	default:
		return nil // permission err
	}
}

func (i InterviewServiceImpl) End(account model.Account, interviewId string) error {
	isCreator, err := i.roomService.IsCreator(account.ID, interviewId)
	switch {
	case err != nil:
		return err
	case isCreator && err == nil:
		_ = i.roomService.SetBizExtraKV(interviewId, "endTime", time.Now())
		_ = i.roomService.InValidNow(interviewId)
		return nil // err handle
	default:
		return nil // permission
	}
}

func (i InterviewServiceImpl) Share(account model.Account, interviewId string, ua string) (interface{}, error) {
	view, err := i.roomService.GetBizExtra(interviewId)
	switch {
	case err != nil:
		return nil, err // err handle
	case ua == "mobile" && err == nil:
		return view, nil
	}
	panic("implement me")
}

func (i InterviewServiceImpl) Update(account model.Account, interviewId string) error {
	//bizExtra:=i.roomService.GetBizExtra()
	return nil
}

func (i InterviewServiceImpl) List(account model.Account, pageNum, pageSize int) (interface{}, error) {
	//interviewList,err:=i.roomService.ListRoomWithExtra(account.ID,nil,pageNum,pageSize)
	return i.roomService.ListRoomWithExtra(account.ID, nil, pageNum, pageSize) // return type remain modify

}

func (i InterviewServiceImpl) Join(account model.Account, interviewId string) error {
	switch {
	case i.IsCreatorOrCandidate(account.ID, interviewId):
		{ // tx 1
			return i.roomService.Enter(account.ID, "nil", interviewId)
		}
	default:
		{ // tx 1
			return i.roomService.Enter(account.ID, "nil", interviewId) // TODO: err handle
		}
		// permission check failed
		//return pderr.ErrPermissionFailed
	}
}

func (i InterviewServiceImpl) Create(creator model.Account, candidate model.Account, interviewId string, bizExtra map[string]interface{}) error {
	// TODO: 如果留空 设为随机
	if interviewId == "" {
		interviewId = common.GenerateID()
	}
	err := i.roomService.Create(creator.ID, interviewId, bizExtra)              // may use random generated name
	err = i.roomService.SetBizExtraKV(interviewId, "candidateId", candidate.ID) // TODO: add key to doc
	// TODO wrap db err: duplicate
	return err
}

// tool func TODO: rewrite
func (i InterviewServiceImpl) IsCreatorOrCandidate(accountId string, roomId string) bool {
	isCreator := make(chan bool, 1)
	isCandidate := make(chan bool, 1)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		res, _ := i.roomService.IsCreator(accountId, roomId)
		// TODO: err handle
		isCreator <- res
		wg.Done()
	}()
	go func() {
		extra, err := i.roomService.GetBizExtra(roomId)
		if err != nil {
			isCandidate <- false
			wg.Done()
			return
		}
		if extra.Get("candidateId").String() == accountId {
			isCandidate <- true
		} else {
			isCandidate <- false
		}
		wg.Done()
	}()
	var res1, res2 bool
	select {
	case res1 = <-isCreator:
		if res1 {
			return res1
		}
	case res2 = <-isCandidate:
		if res2 {
			return res2
		}
	default:
		wg.Wait()
		return res1 || res2
	}
	return false
}
