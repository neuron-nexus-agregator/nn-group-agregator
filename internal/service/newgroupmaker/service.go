package newgroupmaker

import (
	model "agregator/group/internal/model/kafka"
	"agregator/group/internal/service/db"
	"agregator/group/internal/service/elastic"
	"agregator/group/internal/service/kafka"
	"agregator/group/service/vector"
	"fmt"
	"time"
)

type Group struct {
	db      *db.DB
	kafka   *kafka.Kafka
	elastic *elastic.Elastic
}

func New(db *db.DB, kafka *kafka.Kafka, elastic *elastic.Elastic) *Group {
	return &Group{
		db:      db,
		kafka:   kafka,
		elastic: elastic,
	}
}

func (group *Group) MakeNewGroup(item *model.News) error {
	date, err := time.Parse(time.RFC3339, item.PublishDate)
	if err != nil {
		return err
	}
	vec := vector.New(item.Embedding)
	id, err := group.db.Insert(date, item.ID, item.IsRT, vec)
	if err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("failed to insert news into database: such text is already inserted")
	}
	item.ClusterID = int64(id)
	err = group.elastic.RegisterClusert(int64(id), item.PublishDate, item.Embedding, item.Title, item.FullText, item.Description)
	if err != nil {
		return err
	}
	return nil
}

func (group *Group) SaveNews(item model.News) error {
	err := group.db.InsertCompares(uint64(item.ClusterID), uint64(item.ID))
	if err != nil {
		return err
	}
	err = group.db.UpdateParsed(uint64(item.ID), true)
	if err != nil {
		return err
	}
	return group.kafka.Write(item)
}
