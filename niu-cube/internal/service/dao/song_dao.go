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

type SongDaoInterface interface {
	Insert(xl *xlog.Logger, songDo *model.SongDo) (*model.SongDo, error)

	Update(xl *xlog.Logger, songDo *model.SongDo) error

	Select(xl *xlog.Logger, songId string) (*model.SongDo, error)

	SelectByNameAndAuthor(xl *xlog.Logger, songName, author string) (*model.SongDo, error)

	Delete(xl *xlog.Logger, songId string) error

	ListByNameFuzzy(xl *xlog.Logger, songName string) ([]model.SongDo, error)

	ListByAuthorFuzzy(xl *xlog.Logger, authorName string) ([]model.SongDo, error)

	ListAll(xl *xlog.Logger, pageNum, pageSize int) ([]model.SongDo, int, int, error)
}

type SongDaoService struct {
	client           *mgo.Session
	songColl         *mgo.Collection
	roomUserSongColl *mgo.Collection
	xl               *xlog.Logger
}

func NewSongDaoService(xl *xlog.Logger, config *utils.MongoConfig) (*SongDaoService, error) {
	if xl == nil {
		xl = xlog.New("niu-cube-song")
	}
	client, err := mgo.Dial(config.URI)
	if err != nil {
		xl.Error("failed to create mongo client, error: %v", err)
		return nil, err
	}
	songColl := client.DB(config.Database).C(dao.CollectionSong)
	roomUserSongColl := client.DB(config.Database).C(dao.CollectionRoomUserSong)
	return &SongDaoService{
		client,
		songColl,
		roomUserSongColl,
		xl,
	}, nil
}

func (s *SongDaoService) Insert(xl *xlog.Logger, songDo *model.SongDo) (*model.SongDo, error) {
	if xl == nil {
		xl = s.xl
	}
	songDo.Id = bson.NewObjectId().Hex()
	songDo.CreatedTime = time.Now()
	songDo.UpdatedTime = time.Now()
	err := s.songColl.Insert(songDo)
	if err != nil {
		xl.Error("insert into song failed.")
		return nil, err
	}
	return songDo, nil
}

func (s *SongDaoService) Update(xl *xlog.Logger, songDo *model.SongDo) error {
	if xl == nil {
		xl = s.xl
	}
	songDo.UpdatedTime = time.Now()
	err := s.songColl.UpdateId(songDo.Id, songDo)
	if err != nil {
		xl.Error("update song failed.")
		return err
	}
	return nil
}

func (s *SongDaoService) Select(xl *xlog.Logger, songId string) (*model.SongDo, error) {
	if xl == nil {
		xl = s.xl
	}
	var songDo model.SongDo
	err := s.songColl.FindId(songId).One(&songDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record from song")
		} else {
			xl.Error("list those records failed.")
		}
		return nil, err
	}
	return &songDo, nil
}

func (s *SongDaoService) SelectByNameAndAuthor(xl *xlog.Logger, songName, author string) (*model.SongDo, error) {
	if xl == nil {
		xl = s.xl
	}
	var songDo model.SongDo
	err := s.songColl.Find(bson.M{"name": songName, "author": author, "status": model.SongAvailable}).One(&songDo)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find this record.")
		} else {
			xl.Error("select from song failed.")
		}
		return nil, err
	}
	return &songDo, nil
}

func (s *SongDaoService) Delete(xl *xlog.Logger, songId string) error {
	if xl == nil {
		xl = s.xl
	}
	err := s.songColl.RemoveId(songId)
	if err != nil {
		xl.Error("delete from song failed.")
		return err
	}
	return nil
}

// ListByNameFuzzy 这个方法暂时没有需求，做保留
func (s *SongDaoService) ListByNameFuzzy(xl *xlog.Logger, songName string) ([]model.SongDo, error) {
	panic("implement me")
}

// ListByAuthorFuzzy 同上原因
func (s *SongDaoService) ListByAuthorFuzzy(xl *xlog.Logger, authorName string) ([]model.SongDo, error) {
	panic("implement me")
}

func (s *SongDaoService) ListAll(xl *xlog.Logger, pageNum, pageSize int) ([]model.SongDo, int, int, error) {
	if xl == nil {
		xl = s.xl
	}
	songDos := make([]model.SongDo, 0, pageSize)
	skip := (pageNum - 1) * pageSize
	limit := pageSize
	err := s.songColl.Find(bson.M{"status": model.SongAvailable}).Sort("created_time").Skip(skip).Limit(limit).All(&songDos)
	if err != nil {
		if err == mgo.ErrNotFound {
			xl.Info("can't find those records from song.")
		} else {
			xl.Error("list song failed.")
		}
		return nil, 0, 0, err
	}
	total, _ := s.songColl.Find(bson.M{"status": model.SongAvailable}).Count()
	return songDos, total, len(songDos), nil
}
