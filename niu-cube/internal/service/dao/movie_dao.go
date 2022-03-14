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

type MovieDaoInterface interface {
	Insert(xl *xlog.Logger, movieDo *model.MovieDo) error

	Select(xl *xlog.Logger, movieId string) (*model.MovieDo, error)

	SelectByNameDirector(xl *xlog.Logger, name, director string) (*model.MovieDo, error)

	ListAll(xl *xlog.Logger, pageNum, pageSize int) ([]model.MovieDo, int, error)

	Update(xl *xlog.Logger, movieDo *model.MovieDo) error

	Delete(xl *xlog.Logger, movieId string) error
}

type MovieDaoService struct {
	client    *mgo.Session
	movieColl *mgo.Collection
	xl        *xlog.Logger
}

func NewMovieDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*MovieDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-movie")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Error("failed to create mongo client, error: %v", err)
		return nil, err
	}
	movieColl := client.DB(config.Database).C(dao.CollectionMovie)
	return &MovieDaoService{
		client,
		movieColl,
		xl,
	}, nil
}

func (m *MovieDaoService) Insert(xl *xlog.Logger, movieDo *model.MovieDo) error {
	if xl == nil {
		xl = m.xl
	}
	movieDo.Id = bson.NewObjectId().Hex()
	movieDo.CreatedTime = time.Now()
	movieDo.UpdatedTime = time.Now()
	err := m.movieColl.Insert(movieDo)
	if err != nil {
		xl.Error("insert into movie failed.")
		return err
	}
	return nil
}

func (m *MovieDaoService) Select(xl *xlog.Logger, movieId string) (*model.MovieDo, error) {
	if xl == nil {
		xl = m.xl
	}
	result := model.MovieDo{}
	err := m.movieColl.FindId(movieId).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from movie")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &result, nil
}

func (m *MovieDaoService) SelectByNameDirector(xl *xlog.Logger, name, director string) (*model.MovieDo, error) {
	if xl == nil {
		xl = m.xl
	}
	result := model.MovieDo{}
	err := m.movieColl.Find(bson.M{"status": model.MovieAvailable, "name": name, "director": director}).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from movie")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &result, nil
}

func (m *MovieDaoService) ListAll(xl *xlog.Logger, pageNum, pageSize int) ([]model.MovieDo, int, error) {
	if xl == nil {
		xl = m.xl
	}
	movieDos := make([]model.MovieDo, 0, pageSize)
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	err := m.movieColl.Find(bson.M{"status": model.MovieAvailable}).Sort("created_time").Skip(skip).Limit(limit).All(&movieDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find those records from song.")
		} else {
			xl.Error("list song failed.")
		}
		return nil, 0, err
	}
	total, _ := m.movieColl.Find(bson.M{"status": model.MovieAvailable}).Count()
	return movieDos, total, nil
}

func (m *MovieDaoService) Update(xl *xlog.Logger, movieDo *model.MovieDo) error {
	if xl == nil {
		xl = m.xl
	}
	movieDo.UpdatedTime = time.Now()
	err := m.movieColl.UpdateId(movieDo.Id, movieDo)
	if err != nil {
		xl.Error("update movie failed.")
		return err
	}
	return nil
}

func (m *MovieDaoService) Delete(xl *xlog.Logger, movieId string) error {
	if xl == nil {
		xl = m.xl
	}
	err := m.movieColl.RemoveId(movieId)
	if err != nil {
		xl.Error("delete from movie failed.")
		return err
	}
	return nil
}
