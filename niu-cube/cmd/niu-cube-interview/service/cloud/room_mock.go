package cloud

import (
	"encoding/json"
	"fmt"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"github.com/tidwall/gjson"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type RoomService interface {
	// action
	Enter(rAccountId string, roomRole string, roomId string) error
	Leave(rAccountId string, roomId string) error
	Create(creatorId string, roomName string, bizExtra map[string]interface{}) error
	// crud
	IsCreator(accountId string, roomName string) (bool, error)
	InValidNow(roomName string) error // set validBefore to now and kick all users
	ListRoomWithExtra(accountId string, sort []string, pageNum, pageSize int) ([]gjson.Result, error)
	// Biz
	GetBizExtra(roomName string) (*gjson.Result, error)
	SetBizExtra(roomName string, extra map[string]interface{}) error
	SetBizExtraKV(roomName string, k string, v interface{}) error
	GetRoomWithExtra(roomName string) (*model.Room, error)
	KickAllUser(roomName string) error
}

type RoomServiceImpl struct {
	roomColl        *mgo.Collection
	roomAccountColl *mgo.Collection
	bizExtraColl    *mgo.Collection
}

func (r RoomServiceImpl) Enter(rAccountId string, roomRole string, roomId string) error {
	// validate roomRole before xxxx
	exist, err := r.roomExists(roomId) // TODO: 业务层鉴权 暂不考虑
	switch {
	case err != nil:
		return err
	case !exist:
		return fmt.Errorf("room:%v not exist", roomId) //TODO: add to error
	}
	now := time.Now()
	{
		// tx1:
		roomAccount := model.RoomAccount{
			ID:                common.GenerateID(),
			AccountId:         rAccountId,
			RoomRole:          roomRole,
			AccountRoomStatus: model.AccountRoomStatusNil,
			RoomId:            roomId, // TODO: roomID = AppID + roomID
			CreateAt:          now,
			UpdateAt:          now,
		}
		return r.roomAccountColl.Insert(roomAccount) // TODO: db error handle 错误处理
	}
}

func (r RoomServiceImpl) Leave(rAccountId string, roomId string) error {
	exist, err := r.roomExists(roomId) // TODO: 业务层鉴权 暂不考虑
	switch {
	case err != nil:
		return err
	case !exist:
		return fmt.Errorf("room:%v not exist", roomId) //TODO: add to error
	}
	filter := bson.M{
		"account_id": rAccountId,
		"room_id":    roomId,
	}
	var record model.RoomAccount
	err = r.roomAccountColl.Find(filter).One(&record)
	switch {
	case err == mgo.ErrNotFound:
		// 无用户记录 也视为成功
		return nil
	case err != nil && err == mgo.ErrNotFound:
		return err // TODO: db error handle 错误处理
	default:
		record.AccountRoomStatus = model.AccountRoomStatusOut
		return r.roomAccountColl.UpdateId(record.ID, record) // TODO: db error handle 错误处理
	}
}

func (r RoomServiceImpl) Create(creatorId string, roomName string, bizExtra map[string]interface{}) error {
	// TODO: biz ID 是随机生成的，不是客户业务服务器指定的
	bizId := bson.NewObjectId().Hex()
	now := time.Now()
	// TODO: 指定了room_id
	bizExtra["_id"] = bizId
	bizExtra["room_id"] = roomName

	err := r.bizExtraColl.Insert(bizExtra)
	if err != nil {
		return err // TODO: db error handle 错误处理
	}
	room := model.Room{
		ID:          roomName, // TODO: appId + roomName
		Name:        roomName,
		AppId:       "",
		CreatorId:   creatorId,
		BizExtraId:  string(bizId),
		ValidBefore: time.Time{}, // TODO: 暂时不用 ValidBefore
		CreateAt:    now,
		UpdateAt:    now,
	}
	return r.roomColl.Insert(room) // TODO: handle dup id inside, db error handle 错误处理
}

func (r RoomServiceImpl) IsCreator(accountId string, roomName string) (bool, error) {
	var room model.Room
	err := r.roomColl.FindId(roomName).One(&room)
	if err != nil {
		return false, err
	} else {
		return room.CreatorId == accountId, err
	}
}

func (r RoomServiceImpl) InValidNow(roomName string) error {
	panic("implement me")
}

func (r RoomServiceImpl) ListRoomWithExtra(accountId string, sort []string, pageNum, pageSize int) ([]gjson.Result, error) {
	r.roomColl.Find(nil).Sort()
	return nil, nil
	//TODO implement delay
}

func (r RoomServiceImpl) GetBizExtra(roomName string) (*gjson.Result, error) {
	filter := bson.M{"room_id": roomName}
	var extra interface{}
	err := r.bizExtraColl.Find(filter).One(&extra)
	switch {
	case err != nil:
		return nil, err //TODO: wrap db error 错误处理
	default:
		val, _ := json.Marshal(&extra)
		result := gjson.ParseBytes(val)
		return &result, err
	}
}

func (r RoomServiceImpl) SetBizExtra(roomName string, extra map[string]interface{}) error {
	old, err := r.GetBizExtra(roomName)
	if err != nil {
		return err //TODO: wrap db error 错误处理
	}
	pk := old.Get("_id") // TODO: json的tag是这个吗？
	extra["_id"] = pk
	err = r.bizExtraColl.Insert(extra)
	return err //TODO: wrap db error 错误处理
}

func (r RoomServiceImpl) SetBizExtraKV(roomName string, k string, v interface{}) error {
	old, err := r.GetBizExtra(roomName)
	if err != nil {
		return err //TODO: wrap db error 错误处理
	}
	oldMap := old.Value().(map[string]interface{})
	oldMap[k] = v
	id := oldMap["_id"]
	return r.bizExtraColl.UpdateId(id, oldMap)
}

func (r RoomServiceImpl) GetRoomWithExtra(roomName string) (*model.Room, error) {
	pipeline := []bson.M{
		bson.M{"$match": bson.M{"_id": roomName}},
		bson.M{"$lookup": bson.M{
			"from":         model.CollectionBizExtra,
			"localField":   "biz_extra_id",
			"foreignField": "_id",
			"as":           "biz_extra",
		}},
	}
	p := r.roomColl.Pipe(pipeline)
	res := model.FlattenMap{}
	err := p.One(&res)
	return model.NewRoomFromFlattenMap(res), err
}

func (r RoomServiceImpl) KickAllUser(roomName string) error {
	panic("implement me")
}

func NewRoomService() *RoomServiceImpl {
	client, err := mgo.Dial(common.GetConf().Mongo.URI)
	if err != nil {
		panic(err)
	}
	roomColl := client.DB(common.GetConf().Mongo.Database).C(model.CollectionRoom)
	roomAccountColl := client.DB(common.GetConf().Mongo.Database).C(model.CollectionRoomAccount)
	bizExtraColl := client.DB(common.GetConf().Mongo.Database).C(model.CollectionBizExtra)
	return &RoomServiceImpl{roomColl: roomColl, roomAccountColl: roomAccountColl, bizExtraColl: bizExtraColl}
}

func (r RoomServiceImpl) roomExists(roomId string) (bool, error) {
	count, err := r.roomColl.FindId(roomId).Count()
	if err != nil {
		return false, err
	} else {
		return count >= 1, nil
	}
}
