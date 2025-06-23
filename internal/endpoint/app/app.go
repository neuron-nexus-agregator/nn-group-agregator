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
	"math"
	"os"
	"sync"
	"time"
)

type RTChekcer interface {
	CheckForRT(item *model.News) bool
}

type App struct {
	embedding     *embedding.Service
	kafka         *kafka.Kafka
	db            *db.DB
	maker         *newgroupmaker.Group
	elastic       *elastic.Elastic
	timeOut       time.Duration
	mu            sync.Mutex
	logger        interfaces.Logger
	workerLimiter chan struct{}
	wg            sync.WaitGroup
	checker       RTChekcer
}

func New(diff, maxDistance, alpha float64, timeOut time.Duration, checker RTChekcer, logger interfaces.Logger) *App {

	db, err := db.New(30)
	if err != nil {
		logger.Error("Error creating db", "error", err)
		return nil
	}
	elastic := elastic.New()
	embedding := embedding.New(logger)
	kafka := kafka.New([]string{os.Getenv("KAFKA_HOST")}, "aggregator-group", "group-maker", "elastic-text-read")
	maker := newgroupmaker.New(db, kafka, elastic)

	return &App{
		timeOut:       timeOut,
		logger:        logger,
		embedding:     embedding,
		db:            db,
		kafka:         kafka,
		elastic:       elastic,
		maker:         maker,
		mu:            sync.Mutex{},
		workerLimiter: make(chan struct{}, 30),
		wg:            sync.WaitGroup{},
		checker:       checker,
	}
}

func (a *App) processItem(text model.News) {
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
	isRT := a.checker.CheckForRT(&text)
	text.IsRT = isRT
	found := false
	for _, similar := range similars {

		if similar.Distance <= math.Abs(1-a.maker.CalculateDynamicThresholdLogarithmicish(similar.NewsCount)) {
			found = true
			text.ClusterID = similar.ID
			break
		}
	}
	text.Embedding = textEmbedding.GetArray()
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
}

func (a *App) process() {
	textInput := a.kafka.TextOutput()
	go func() {
		a.kafka.StartReadingText(context.Background())
	}()
	for text := range textInput {
		a.workerLimiter <- struct{}{}

		a.wg.Add(1) // Увеличиваем счетчик WaitGroup
		go func(item model.News) {
			defer a.wg.Done()                    // Уменьшаем счетчик по завершении горутины
			defer func() { <-a.workerLimiter }() // Освобождаем "токен" после завершения работы воркера

			a.processItem(item)
		}(text)
	}
}

func (a *App) Run() {
	a.process()
}
