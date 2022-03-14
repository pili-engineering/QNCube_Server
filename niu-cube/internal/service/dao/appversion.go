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

type AppVersionDao interface {
	GetNewestAppVersion(arch string) (*model.AppVersion, error)

	InsertAppVersion(version *model.AppVersion) error
}

type AppVersionDaoService struct {
	collection *mongo.Collection
	logger     *xlog.Logger
}

func NewAppVersionDaoService(config *utils.MongoConfig) *AppVersionDaoService {
	logger := xlog.New("app_version dao service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
	if err != nil {
		panic(err)
	}
	collection := client.Database(config.Database).Collection(dao.CollectionAppVersion)
	return &AppVersionDaoService{
		collection,
		logger,
	}
}

func (a *AppVersionDaoService) GetNewestAppVersion(arch string) (*model.AppVersion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var result model.AppVersion
	err := a.collection.FindOne(ctx, primitive.M{"arch": arch}, &options.FindOneOptions{
		Sort: primitive.M{"created_time": -1},
	}).Decode(&result)
	if err != nil {
		a.logger.Errorf("get newest app version error: %v", err)
		return nil, err
	}
	return &result, nil
}

func (a *AppVersionDaoService) InsertAppVersion(version *model.AppVersion) error {
	version.Id = primitive.NewObjectID().Hex()
	version.CreatedTime = time.Now()
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	_, err := a.collection.InsertOne(timeout, version)
	if err != nil {
		a.logger.Errorf("insert app version error: %v", err)
		return err
	}
	return nil
}
