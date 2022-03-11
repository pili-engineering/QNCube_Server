package db

import (
	"fmt"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"gopkg.in/mgo.v2"
)

type IEService interface {
	// crud
	ListRoom(roomId string, pageNum, pageSize int) (interface{}, int, int, error) // return data,totalPage,nextPage,error
	UpdateRoomNoticeAndTitle(user model.AccountDo, roomId string, notice, title string) error

	countUserInRoom(roomId string) // cnt room status
	// action
	LeaveRoom(user model.AccountDo, roomId string) error
	EnterRoom(user model.AccountDo, roomId string) (room model.IeExtra, err error)
	CreateRoom(user model.AccountDo, roomID string, extra model.FlattenMap) error
	GetCreator(roomId string) (user model.AccountDo, err error)

	// 业务需要 隐藏 或者 添加计算属性
	MakeIERoomResponse(roomId string) (model.FlattenMap, error)
}

func NewIEService() IEService {
	r := NewRoomService()
	return NewIEServiceImpl(r)
}

type IEServiceImpl struct {
	roomService    RoomService
	accountService *AccountService
}

func NewIEServiceImpl(roomService RoomService) *IEServiceImpl {
	a, err := NewAccountService(*utils.DefaultConf.Mongo, nil)
	if err != nil {
		panic(err)
	}
	return &IEServiceImpl{roomService: roomService, accountService: a}
}

// ListRoom return data,totalNum,NextNum,error
func (i *IEServiceImpl) ListRoom(roomId string, pageNum, pageSize int) (interface{}, int, int, error) {
	sort := []string{"-create_at"}
	data, totalPage, nextPage, err := i.roomService.ListRoom("", sort, pageNum, pageSize)
	if err != nil {
		return nil, 0, 0, err
	}
	// make response
	res := make([]model.FlattenMap, 0)
	for _, room := range data {
		bizData, err := i.MakeIERoomResponse(room.ID)
		if err != nil {
			continue
		}
		res = append(res, bizData.Exclude("_id"))
		//res[index] = room.Map()
	}
	return res, totalPage, nextPage, err
}

func (i *IEServiceImpl) UpdateRoomNoticeAndTitle(user model.AccountDo, roomId string, notice string, title string) error {
	master, err := i.roomService.IsCreator(user.ID, roomId)
	if err != nil {
		return err
	}
	switch {
	case master:
		err := i.roomService.SetBizExtraKV(roomId, "notice", notice)
		if err != nil {
			return err
		}
		err = i.roomService.SetBizExtraKV(roomId, "title", title)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("权限不足") // TODO: 权限不足错误加入文档
	}
}

func (i *IEServiceImpl) GetRoomWithExtra(roomId string) {
	panic("implement me")
}

func (i *IEServiceImpl) countUserInRoom(roomId string) {
	panic("implement me")
}

func (i *IEServiceImpl) LeaveRoom(user model.AccountDo, roomId string) error {
	master, err := i.roomService.IsCreator(user.ID, roomId)
	if err != nil && err != mgo.ErrNotFound {
		return err
	}
	switch {
	case master:
		// kick all user in the room
		err := i.roomService.InValidNow(roomId)
		if err != nil {
			return err
		}
		return nil
	default:
		// 仅仅修改 roomAccount 记录
		err := i.roomService.Leave(user.ID, roomId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *IEServiceImpl) EnterRoom(user model.AccountDo, roomId string) (extra model.IeExtra, err error) {
	//if i.roomService.
	master, err := i.roomService.IsCreator(user.ID, roomId)
	if err != nil {
		return extra, err
	}
	switch {
	case master:
		err = i.roomService.Enter(user.ID, "master", roomId)
	default:
		err = i.roomService.Enter(user.ID, "guest", roomId)
	}
	extraData, err := i.roomService.GetBizExtra(roomId)
	if err != nil {
		return extra, err
	}
	extra = model.IeExtra{
		ID:         extraData.Get("_id").String(),
		RoomID:     extraData.Get("roomId").String(),
		Title:      extraData.Get("title").String(),
		Notice:     extraData.Get("notice").String(),
		RoomAvatar: extraData.Get("roomAvatar").String(),
	}
	return extra, err
}

func (i *IEServiceImpl) CreateRoom(user model.AccountDo, roomID string, extra model.FlattenMap) error {
	err := i.roomService.Create(user.ID, roomID, extra)
	return err
}

func (i *IEServiceImpl) MakeIERoomResponse(roomId string) (model.FlattenMap, error) {
	biz, err := i.roomService.GetBizExtra(roomId)
	if err != nil {
		return nil, err
	}
	activeUsers, err := i.roomService.ListRoomUser(roomId)
	if err != nil {
		return nil, err
	}
	bizMap := biz.Value().(map[string]interface{})
	bizMap["user_num"] = len(activeUsers)
	return bizMap, nil
}

func (i *IEServiceImpl) GetCreator(roomId string) (user model.AccountDo, err error) {
	id, err := i.roomService.GetCreatorId(roomId)
	if err != nil {
		return user, err
	}
	u, err := i.accountService.GetAccountByID(nil, id)
	if err != nil {
		return user, err
	}
	return *u, err
}
