package db

import (
	"fmt"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	errors2 "github.com/solutions/niu-cube/internal/protodef/errors"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type RepairInterface interface {
	// 创建修理房间
	CreateRoom(xl *xlog.Logger, repairRoom *model.RepairRoomDo) (*model.RepairRoomDo, error)
	// 创建房间和用户的关系
	CreateRoomUser(xl *xlog.Logger, repairRoomUser *model.RepairRoomUserDo) (*model.RepairRoomUserDo, error)

	// 加入房间
	JoinRoom(xl *xlog.Logger, userID string, roomID string, role string) (*model.RepairRoomDo, []model.RepairRoomUserDo, error)

	// LimitStaff 限制检修员数量
	LimitStaff(role, roomId string) (bool, error)

	// 按房间号查询房间
	GetRoomByID(xl *xlog.Logger, roomID string) (*model.RepairRoomDo, error)

	LeaveRoom(xl *xlog.Logger, userID string, roomID string) error

	// 房间列表查询
	ListRoomsByPage(xl *xlog.Logger, userID string, pageNum int, pageSize int) ([]model.RepairRoomDo, int, error)

	// 心跳
	HeartBeat(xl *xlog.Logger, userID string, roomID string) error

	// 获取房间信息
	GetRoomContent(xl *xlog.Logger, userID string, roomID string) (*model.RepairRoomDo, []model.RepairRoomUserDo, error)

	// 获取超时用户
	ListHeartBeatTimeOutUser(xl *xlog.Logger) ([]model.RepairRoomUserDo, error)

	// 房间里是的包含有效的检修员
	ContainStaff(roomID string) (bool, error)
}

type RepairService struct {
	mongoClient        *mgo.Session
	repairRoomColl     *mgo.Collection
	repairRoomUserColl *mgo.Collection
	xl                 *xlog.Logger
}

func NewRepairService(conf utils.MongoConfig, xl *xlog.Logger) (*RepairService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-repair")
	}
	mongoClient, err := mgo.Dial(conf.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error %v", err)
		return nil, err
	}
	repairRoomColl := mongoClient.DB(conf.Database).C(dao.CollectionRepairRoom)
	repairRoomUserColl := mongoClient.DB(conf.Database).C(dao.CollectionRepairRoomUser)
	return &RepairService{
		mongoClient:        mongoClient,
		repairRoomColl:     repairRoomColl,
		repairRoomUserColl: repairRoomUserColl,
		xl:                 xl,
	}, nil
}

func (c *RepairService) CreateRoom(xl *xlog.Logger, repairRoom *model.RepairRoomDo) (*model.RepairRoomDo, error) {
	if xl == nil {
		xl = c.xl
	}
	err := c.repairRoomColl.Insert(repairRoom)
	if err != nil {
		xl.Errorf("failed to Insert repairRoom  repairRoom: %v, error %v", repairRoom, err)
		return nil, err
	}
	xl.Infof("CreateRepairRoom room success!, repairRoom : %v", repairRoom)
	return repairRoom, nil
}

func (c *RepairService) CreateRoomUser(xl *xlog.Logger, repairRoomUser *model.RepairRoomUserDo) (*model.RepairRoomUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	err := c.repairRoomUserColl.Insert(repairRoomUser)
	if err != nil {
		xl.Errorf("failed to Insert CreateRoomUser  repairRoomUser: %v, error %v", repairRoomUser, err)
		return nil, err
	}
	xl.Infof("CreateRoomUser room success!, repairRoomUser : %v", repairRoomUser)
	return repairRoomUser, nil
}

func (c *RepairService) JoinRoom(xl *xlog.Logger, userID string, roomId string, role string) (*model.RepairRoomDo, []model.RepairRoomUserDo, error) {
	if xl == nil {
		xl = c.xl
	}

	// 查看是否存在房间
	room, err := c.GetRoomByID(xl, roomId)
	if err != nil || room.Status == int(model.RepairRoomStatusCodeClose) {
		xl.Infof("room is not exit or close ,roomId: %s", roomId)
		return nil, nil, fmt.Errorf("房间异常")
	}

	// 限制检修员数量
	if role == string(model.RepairRoomRoleStaff) {
		if ok, _ := c.LimitStaff(userID, roomId); !ok {
			xl.Infof("there is already a staff in the room[%s]", roomId)
			return nil, nil, fmt.Errorf("仅限一名检修员")
		}
	}

	// 处理房间人员关系
	var repairRoomUser model.RepairRoomUserDo
	condition := bson.M{"_id": roomId + "_" + userID}

	err = c.repairRoomUserColl.Find(condition).One(&repairRoomUser)
	if err != nil {
		// add
		repairRoomUser := &model.RepairRoomUserDo{
			ID:                roomId + "_" + userID,
			RoomId:            roomId,
			UserID:            userID,
			Role:              role,
			Status:            int(model.RepairRoomUserStatusCodeNormal),
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
			LastHeartBeatTime: time.Now(),
		}
		_, err = c.CreateRoomUser(xl, repairRoomUser)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// update
		repairRoomUser.Status = int(model.RepairRoomUserStatusCodeNormal)
		repairRoomUser.Role = role
		repairRoomUser.UpdateTime = time.Now()
		repairRoomUser.LastHeartBeatTime = time.Now()
		err = c.repairRoomUserColl.UpdateId(repairRoomUser.ID, repairRoomUser)
		if err != nil {
			xl.Errorf("repairRoomUserColl.UpdateId err:%v", err)
			return nil, nil, err
		}
	}
	// 加入房间
	allRoomUserDos, allUsersQueryErr := c.AllRoomUsers(xl, roomId)
	if allUsersQueryErr != nil {
		xl.Errorf("allRoomUsers failed  roomId %s, error %v", roomId, err)
		return nil, nil, allUsersQueryErr
	}
	return room, allRoomUserDos, nil
}

func (c *RepairService) LimitStaff(userId, roomId string) (bool, error) {
	var repairRoomUserDos []model.RepairRoomUserDo
	err := c.repairRoomUserColl.Find(bson.M{"roomId": roomId, "role": model.RepairRoomRoleStaff, "status": model.RepairRoomUserStatusCodeNormal}).All(&repairRoomUserDos)
	if err != nil {
		return false, err
	}
	if len(repairRoomUserDos) > 0 {
		if userId != repairRoomUserDos[0].UserID {
			return false, nil
		}
	}
	return true, nil
}

func (c *RepairService) LeaveRoom(xl *xlog.Logger, userID string, roomID string) error {
	if xl == nil {
		xl = c.xl
	}

	// 查看是否存在房间
	room, err := c.GetRoomByID(xl, roomID)
	if err != nil {
		xl.Infof("room is not exit ,roomId: %s", roomID)
		return err
	}

	// 查看room_user
	var repairRoomUser model.RepairRoomUserDo
	condition := bson.M{"_id": roomID + "_" + userID}
	err = c.repairRoomUserColl.Find(condition).One(&repairRoomUser)

	if err != nil {
		// 不存在，直接当正常结束
		xl.Infof("room_user is not exit ,roomId: %s,userID: %s", roomID, userID)
		return nil
	}
	// update room_user
	repairRoomUser.Status = int(model.RepairRoomUserStatusCodeDelete)
	repairRoomUser.UpdateTime = time.Now()
	repairRoomUser.LastHeartBeatTime = time.Now()
	err = c.repairRoomUserColl.UpdateId(repairRoomUser.ID, repairRoomUser)
	if err != nil {
		xl.Errorf("repairRoomUserColl.UpdateId err:%v", err)
		return err
	}

	// 查看房间里还有多少人，没有人的话房间关闭
	allRoomUserDos, allUsersQueryErr := c.AllRoomUsers(xl, roomID)
	if allUsersQueryErr != nil {
		xl.Errorf("allRoomUsers failed  roomId %s, error %v", roomID, err)
		return allUsersQueryErr
	}
	if len(allRoomUserDos) < 1 {
		room.UpdateTime = time.Now()
		room.Status = int(model.RepairRoomStatusCodeClose)
		c.repairRoomColl.UpdateId(roomID, room)
	}

	return nil

}

func (c *RepairService) ListRoomsByPage(xl *xlog.Logger, userID string, pageNum int, pageSize int) ([]model.RepairRoomDo, int, error) {

	if xl == nil {
		xl = c.xl
	}
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	repairRooms := []model.RepairRoomDo{}
	err := c.repairRoomColl.Find(bson.M{"status": model.RepairRoomUserStatusCodeNormal}).Sort("-createTime").Skip(skip).Limit(limit).All(&repairRooms)
	if err != nil {
		xl.Errorf("failed to ListRoomsByPage of userId %s, error %v", userID, err)
		return nil, 0, err
	}
	total, err := c.repairRoomColl.Find(bson.M{"status": model.RepairRoomUserStatusCodeNormal}).Count()
	if err != nil {
		xl.Errorf("failed to ListRoomsByPage count of userId %s, error %v", userID, err)
		return nil, 0, err
	}
	return repairRooms, total, err

}

func (c *RepairService) GetRoomByID(xl *xlog.Logger, roomID string) (*model.RepairRoomDo, error) {
	if xl == nil {
		xl = c.xl
	}
	fields := map[string]interface{}{"_id": roomID}

	repairRoom := model.RepairRoomDo{}
	err := c.repairRoomColl.Find(fields).One(&repairRoom)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("no such room for fields %v", fields)
			return nil, &errors2.ServerError{Code: errors2.ServerErrorRoomNotFound}
		}
		xl.Errorf("failed to get room, error %v", fields)
		return nil, err
	}
	return &repairRoom, nil
}

func (c *RepairService) HeartBeat(xl *xlog.Logger, userID string, roomID string) error {
	if xl == nil {
		xl = c.xl
	}

	// 查看是否存在房间
	room, err := c.GetRoomByID(xl, roomID)
	if err != nil || room.Status == int(model.RepairRoomStatusCodeClose) {
		xl.Infof("room is not exit or close ,roomId: %s", roomID)
		return nil
	}

	// 查看room_user
	var repairRoomUser model.RepairRoomUserDo
	condition := bson.M{"_id": roomID + "_" + userID}
	err = c.repairRoomUserColl.Find(condition).One(&repairRoomUser)

	if err != nil {
		// 不存在，直接当正常结束
		xl.Infof("room_user is not exit ,roomId: %s,userID: %s", roomID, userID)
		return nil
	}

	repairRoomUser.UpdateTime = time.Now()
	repairRoomUser.LastHeartBeatTime = time.Now()
	err = c.repairRoomUserColl.UpdateId(repairRoomUser.ID, repairRoomUser)
	if err != nil {
		xl.Errorf("repairRoomUserColl.UpdateId err:%v", err)
		return err
	}
	return nil
}

func (c *RepairService) GetRoomContent(xl *xlog.Logger, userID string, roomID string) (*model.RepairRoomDo, []model.RepairRoomUserDo, error) {

	if xl == nil {
		xl = c.xl
	}

	// 查看是否存在房间
	room, err := c.GetRoomByID(xl, roomID)
	if err != nil {
		xl.Infof("room is not exit ,roomId: %s", roomID)
		return nil, nil, err
	}
	// 加入房间
	allRoomUserDos, allUsersQueryErr := c.AllRoomUsers(xl, roomID)
	if allUsersQueryErr != nil {
		xl.Errorf("allRoomUsers failed  roomId %s, error %v", roomID, err)
		return nil, nil, allUsersQueryErr
	}
	return room, allRoomUserDos, nil
}

func (c *RepairService) ListHeartBeatTimeOutUser(xl *xlog.Logger) ([]model.RepairRoomUserDo, error) {

	if xl == nil {
		xl = c.xl
	}
	ddl := time.Now().Add((time.Duration(model.HeartBeatInterval) * 5) * time.Second * (-1))
	condition := bson.M{
		"lastHeartBeatTime": bson.M{
			"$lt": ddl,
		},
		"status": int(model.RepairRoomUserStatusCodeNormal),
	}
	roomUsers := make([]model.RepairRoomUserDo, 0)
	err := c.repairRoomUserColl.Find(condition).Sort("-createTime").Limit(10).All(&roomUsers)
	return roomUsers, err

}

func (c *RepairService) ContainStaff(roomID string) (bool, error) {

	repairRoomUserDos := []model.RepairRoomUserDo{}
	c.repairRoomUserColl.Find(bson.M{"roomId": roomID, "role": model.RepairRoomRoleStaff, "status": model.RepairRoomUserStatusCodeNormal}).All(&repairRoomUserDos)
	if len(repairRoomUserDos) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (c *RepairService) AllRoomUsers(xl *xlog.Logger, roomID string) ([]model.RepairRoomUserDo, error) {
	if xl == nil {
		xl = c.xl
	}
	repairRoomUserDos := []model.RepairRoomUserDo{}
	err := c.repairRoomUserColl.Find(bson.M{"roomId": roomID, "status": model.RepairRoomUserStatusCodeNormal}).All(&repairRoomUserDos)
	return repairRoomUserDos, err
}
