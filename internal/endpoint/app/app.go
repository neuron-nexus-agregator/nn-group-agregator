package app

import (
	"agregator/group/internal/interfaces"
	model "agregator/group/internal/model/kafka"
	"agregator/group/internal/service/db"
	"agregator/group/internal/service/elastic"
	"agregator/group/internal/service/embedding"
	"agregator/group/internal/service/kafka"
	"agregator/group/internal/service/newgroupmaker"
	"context"
	"os"
	"strings"
	"sync"
	"time"
)

type App struct {
	embedding *embedding.Service
	kafka     *kafka.Kafka
	db        *db.DB
	maker     *newgroupmaker.Group
	elastic   *elastic.Elastic
	words     []string
	timeOut   time.Duration
	mu        sync.Mutex
	logger    interfaces.Logger
}

func New(diff, maxDistance, alpha float64, timeOut time.Duration, logger interfaces.Logger) *App {

	db, err := db.New(10)
	if err != nil {
		logger.Error("Error creating db", "error", err)
		return nil
	}
	elastic := elastic.New()
	embedding := embedding.New(logger)
	kafka := kafka.New([]string{os.Getenv("KAFKA_HOST")}, "aggregator-group", "group-maker", "elastic-text-read")
	maker := newgroupmaker.New(db, kafka, elastic)

	words, err := db.GetRTWords()
	if err != nil {
		logger.Error("Error getting RT words", "error", err)
		return nil
	}

	return &App{
		timeOut:   timeOut,
		logger:    logger,
		embedding: embedding,
		db:        db,
		kafka:     kafka,
		elastic:   elastic,
		maker:     maker,
		words:     words,
		mu:        sync.Mutex{},
	}
}

func (a *App) UpdateWords() {
	timer := time.NewTicker(10 * time.Minute)
	defer timer.Stop()
	for range timer.C {
		words, err := a.db.GetRTWords()
		if err != nil {
			a.logger.Error("Error getting RT words", "error", err)
			return
		}
		a.mu.Lock()
		a.words = words
		a.mu.Unlock()
	}
}

func (a *App) process() {
	textInput := a.kafka.TextOutput()
	go func() {
		a.kafka.StartReadingText(context.Background())
	}()
	for text := range textInput {
		go func(text model.News) {
			textEmbedding, err := a.embedding.GetEmbedding(text.Title, text.Description, text.FullText)
			if err != nil {
				a.logger.Error("Error getting embedding", "error", err)
				return
			}
			similars, err := a.elastic.GetClosest(textEmbedding.GetArray(), 15)
			if err != nil {
				a.logger.Error("Error getting similars", "error", err)
				return
			}
			a.checkForRT(&text)
			found := false
			for _, similar := range similars {
				if similar.Distance >= a.maker.CalculateDynamicThresholdLogarithmicish(similar.NewsCount) {
					found = true
					text.ClusterID = similar.ID
					break
				}
			}
			if !found {
				err := a.maker.MakeNewGroup(&text)
				if err != nil {
					a.logger.Error("Error making new group", "error", err)
					return
				}
			}
			err = a.maker.SaveNews(text)
			if err != nil {
				a.logger.Error("Error saving news", "error", err)
				return
			}
		}(text)
	}
}

func (a *App) Run() {
	go a.UpdateWords()
	a.process()
}

func (a *App) checkForRT(item *model.News) {
	a.mu.Lock()
	words := a.words
	a.mu.Unlock()
	item.IsRT = false
	full := item.Title + "\n\n" + item.Description + "\n\n" + item.FullText
	full = strings.ToLower(full)
	for _, word := range words {
		word := strings.ToLower(word)
		if strings.HasPrefix(full, word) {
			item.IsRT = true
			return
		}
		if strings.HasSuffix(full, word) {
			item.IsRT = true
			return
		}
		if strings.Contains(full, " "+word) {
			item.IsRT = true
			return
		}
	}
}
