package dao

import (
	"time"

	"github.com/qiniu/x/xlog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
)

// ImageFileDaoInterface
type ImageFileDaoInterface interface {
	InsertImageFile(xl *xlog.Logger, imageFile *model.ImageFileDo) (*model.ImageFileDo, error)

	SelectRecentImage(xl *xlog.Logger) (*model.ImageFileDo, error)
}

// ImageFileDaoService
type ImageFileDao struct {
	client        *mgo.Session
	imageFileColl *mgo.Collection
	xl            *xlog.Logger
}

func NewImageFileDao(xl *xlog.Logger, config *utils.MongoConfig) (*ImageFileDao, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-image-file")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Errorf("failed to create mongo client, error: %v", err)
		return nil, err
	}
	imageFileColl := client.DB(config.Database).C(dao.CollectionQiniuImageFile)
	return &ImageFileDao{
		client,
		imageFileColl,
		xl,
	}, nil
}

func (b *ImageFileDao) InsertImageFile(xl *xlog.Logger, imageFile *model.ImageFileDo) (*model.ImageFileDo, error) {
	if xl == nil {
		xl = b.xl
	}
	imageFile.CreateTime = time.Now()
	imageFile.UpdateTime = time.Now()
	imageFile.ID = bson.NewObjectId().Hex()
	err := b.imageFileColl.Insert(imageFile)
	if err != nil {
		xl.Error("insert into image_file failed.")
		return nil, err
	}
	return imageFile, nil
}

func (b *ImageFileDao) SelectRecentImage(xl *xlog.Logger) (*model.ImageFileDo, error) {
	if xl == nil {
		xl = b.xl
	}
	var imageFile model.ImageFileDo
	err := b.imageFileColl.Find(nil).Sort("-createTime").One(&imageFile)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Infof("can't find this records:[%s] from image_file.")
		} else {
			xl.Error("select from image_file failed.")
		}
		return nil, err
	}
	return &imageFile, nil
}
