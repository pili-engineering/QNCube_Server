package db

import (
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"gopkg.in/mgo.v2"
	"math/rand"
)

type VersionService struct {
	versionCollection *mgo.Collection
	xl                *xlog.Logger
}

func NewVersionService(xl *xlog.Logger, config utils.MongoConfig) *VersionService {
	v := new(VersionService)
	v.xl = xlog.New("version service")
	db, err := mgo.Dial(config.URI)
	if err != nil {
		v.xl.Fatalf("error dialing service error:%v", err)
	}
	err = db.Ping()
	if err != nil {
		v.xl.Fatalf("err ping db error:%v", err)
	}
	v.versionCollection = db.DB(config.Database).C("versions")
	return v
}

// CRUD

func (v *VersionService) Create(xl *xlog.Logger, version model.VersionDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	if version.ID == "" {
		version.ID = v.generateID()
	}
	err := v.versionCollection.Insert(version)
	if err != nil {
		logger.Errorf("error create versionDo %v err:%v", version, err)
	}
	return err
}

func (v *VersionService) Update(xl *xlog.Logger, version model.VersionDo) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.versionCollection.UpdateId(version.ID, version)
	if err != nil {
		logger.Errorf("error update versionDo %v err:%v", version, err)
	}
	return err
}

func (v *VersionService) Delete(xl *xlog.Logger, id string) error {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	err := v.versionCollection.RemoveId(id)
	if err != nil {
		logger.Errorf("error update versionId %v err:%v", id, err)
	}
	return err
}

func (v *VersionService) GetOneByMap(xl *xlog.Logger, filter interface{}) (model.VersionDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	var verson model.VersionDo
	err := v.versionCollection.Find(filter).One(&verson)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return verson, err
	}
	return verson, err
}

// GetByMap
func (v *VersionService) GetByMap(xl *xlog.Logger, filter interface{}) ([]model.VersionDo, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	versions := make([]model.VersionDo, 0)
	var err error
	err = v.versionCollection.Find(filter).All(&versions)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return versions, err
	}
	return versions, err
}

// GetPageByMap
func (v *VersionService) GetPageByMap(xl *xlog.Logger, filter interface{}, pageNum, pageSize int) ([]model.VersionDo, int, error) {
	var logger *xlog.Logger
	if xl != nil {
		logger = xl
	}
	versions := make([]model.VersionDo, 0)
	var err error
	err = v.versionCollection.Find(filter).Skip((pageNum - 1) * pageSize).Limit(pageSize).All(&versions)
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return versions, 0, err
	}
	cnt, err := v.versionCollection.Find(filter).Count()
	if err != nil {
		logger.Debugf("error get by filter %v err:%v", filter, err)
		return versions, 0, err
	}
	return versions, cnt, err
}

// generateID utils func: for 12-digit random id generation
func (v *VersionService) generateID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	idLength := 12
	id := ""
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		id = id + string(alphaNum[index])
	}
	return id
}

func (v *VersionService) Permit(action string, userId string) bool {
	return true
}
