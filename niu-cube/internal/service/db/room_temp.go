package db

import (
	"encoding/json"
	"fmt"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
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
	ListRoom(accountId string, sort []string, pageNum, pageSize int) ([]model.Room, int, int, error)
	ListRoomUser(roomName string) ([]map[string]interface{}, error)
	//ModifyRoomAccountStatus(accountId string,roomName string,status model.AccountRoomStatus)error
	// Biz
	GetBizExtra(roomName string) (*gjson.Result, error)
	SetBizExtra(roomName string, extra map[string]interface{}) error
	SetBizExtraKV(roomName string, k string, v interface{}) error
	//GetRoom(roomName string)
	GetCreatorId(roomName string) (string, error)
	GetRoomWithExtra(roomName string) (model.Room, error)
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
		return mgo.ErrNotFound //TODO: add to error
	}
	// 考虑重复进入
	{
		// 曾进过
		filter := bson.M{
			"accountId": rAccountId,
			"roomId":    roomId,
		}
		var record model.RoomAccount
		err := r.roomAccountColl.Find(filter).One(&record)
		switch err {
		case nil:
			// 已存在
			record.AccountRoomStatus = model.AccountRoomStatusIn
			err = r.roomAccountColl.Update(filter, record)
			if err != nil {
				return err // TODO 像上面一样处理
			}
			return nil
		case mgo.ErrNotFound:
			// 不存在
			now := time.Now()
			{
				// tx1:
				roomAccount := model.RoomAccount{
					ID:                utils.GenerateID(), // APP_ID + AccountID
					AccountId:         rAccountId,         // APP_ID + AccountID
					RoomRole:          roomRole,
					AccountRoomStatus: model.AccountRoomStatusIn,
					RoomId:            roomId, // TODO: roomID = AppID + roomID
					CreateAt:          now,
					UpdateAt:          now,
				}
				return r.roomAccountColl.Insert(roomAccount) // TODO: db error handle 错误处理
			}
		default:
			// 数据库其他错误
			return err
		}
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
		"accountId": rAccountId,
		"roomId":    roomId,
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
	// TODO: 指定了roomId
	bizExtra["_id"] = bizId
	bizExtra["roomId"] = roomName

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
		ValidBefore: model.MaxTime, // TODO: 暂时不用 ValidBefore
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
	return r.roomColl.UpdateId(roomName, bson.M{"$currentDate": bson.M{"valid_before": false}})
}

// ListRoom if accountId is empty,return all user's room,return result,totalPage,nextPage,error
func (r RoomServiceImpl) ListRoom(accountId string, sort []string, pageNum, pageSize int) ([]model.Room, int, int, error) {
	//pipeline:=[]bson.M{
	//	bson.M{"$skip":pageNum*(pageSize-1)},
	//	bson.M{"$limit":pageSize},
	//	bson.M{"$lookup":bson.M{
	//		"from":dao.CollectionBizExtra,
	//		"localField":"biz_extra_id",
	//		"foreignField":"_id",
	//		"as":"biz_extra",
	//	}},
	//}
	//p:=r.roomColl.Pipe(pipeline)
	//mRes:=make([]model.FlattenMap,0)
	//err:=p.All(&mRes)
	//res:=make([]*model.Room,len(mRes))
	//for i,m:=range mRes{
	//	res[i] = model.NewRoomFromFlattenMap(m)
	//}
	//return res,err
	var filter = model.MakeFlattenMap("valid_before", bson.M{"$gt": time.Now()})
	if accountId != "" {
		filter.Merge(bson.M{"creator_id": accountId})
	} // TODO add APP ID into condition
	res := []model.Room{}
	err := r.roomColl.Find(filter).Sort(sort...).Skip((pageNum - 1) * pageSize).Limit(pageSize).All(&res)
	if err != nil {
		return nil, 0, 0, err
	}
	cnt, err := r.roomColl.Find(filter).Count()
	if err != nil {
		return nil, 0, 0, err
	}
	totalPage := cnt / pageSize
	if cnt%pageSize != 0 {
		totalPage = totalPage + 1
	}
	nextPage := pageNum + 1
	if nextPage > totalPage {
		nextPage = pageNum
	}
	return res, totalPage, nextPage, nil
}

func (r RoomServiceImpl) ListRoomUser(roomName string) ([]map[string]interface{}, error) {
	filter := bson.M{"roomId": roomName, "accountRoomStatus": "in"}
	sort := []string{"create_at"}
	result := make([]map[string]interface{}, 0)
	err := r.roomAccountColl.Find(filter).Sort(sort...).All(&result)
	return result, err
}

func ModifyRoomAccountStatus(accountId string, roomName string, status model.AccountRoomStatus) error {
	return nil
}

func (r RoomServiceImpl) GetBizExtra(roomName string) (*gjson.Result, error) {
	filter := bson.M{"roomId": roomName}
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

func (r RoomServiceImpl) GetRoomWithExtra(roomName string) (model.Room, error) {
	pipeline := []bson.M{
		bson.M{"$match": bson.M{"_id": roomName}},
		bson.M{"$lookup": bson.M{
			"from":         dao.CollectionBizExtra,
			"localField":   "biz_extra_id",
			"foreignField": "_id",
			"as":           "biz_extra",
		}},
	}
	p := r.roomColl.Pipe(pipeline)
	res := model.FlattenMap{}
	err := p.One(&res)
	return *model.NewRoomFromFlattenMap(res), err
}

func (r RoomServiceImpl) KickAllUser(roomName string) error {
	panic("implement me")
}

func NewRoomService() *RoomServiceImpl {
	client, err := mgo.Dial(utils.DefaultConf.Mongo.URI)
	if err != nil {
		panic(err)
	}
	roomColl := client.DB(utils.DefaultConf.Mongo.Database).C(dao.CollectionRoom)
	roomAccountColl := client.DB(utils.DefaultConf.Mongo.Database).C(dao.CollectionRoomAccount)
	bizExtraColl := client.DB(utils.DefaultConf.Mongo.Database).C(dao.CollectionBizExtra)
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

func (r RoomServiceImpl) GetCreatorId(roomName string) (string, error) {
	var room model.Room
	err := r.roomColl.FindId(roomName).One(&room)
	if err != nil {
		return "", err
	}
	return room.CreatorId, err
}
