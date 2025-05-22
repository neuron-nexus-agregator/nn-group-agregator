package redis

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"agregator/group/internal/interfaces"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

type Redis struct {
	redisClient *redis.Client
	ctx         context.Context
	logger      interfaces.Logger
}

const (
	redisKeyPrefix = "news:"
)

// New создает новый экземпляр Redis с проверкой подключения.
func New(logger interfaces.Logger) (*Redis, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		logger.Error("Error pinging redis", "error", err)
		return nil, err
	}

	r := &Redis{
		redisClient: redisClient,
		ctx:         context.Background(),
		logger:      logger,
	}

	return r, nil
}

// SaveNewsItems сохраняет массив структур NewsItem в Redis с указанной датой.
func (r *Redis) SavManyNews(items [][]byte) error {
	pipe := r.redisClient.Pipeline()
	expiration := 48 * time.Hour
	for _, item := range items {
		key := fmt.Sprintf("%s:%s", redisKeyPrefix, r.generateUniqueID())
		pipe.Set(r.ctx, key, string(item), expiration)
	}
	if _, err := pipe.Exec(r.ctx); err != nil {
		r.logger.Error("Error executing pipeline", "error", err)
	}
	return nil
}

// SaveNewsItems сохраняет массив структур NewsItem в Redis с указанной датой.
func (r *Redis) SaveNews(item interface{}) error {

	expiration := 48 * time.Hour
	data, err := json.Marshal(item)
	if err != nil {
		r.logger.Error("Error marshaling data", "error", err, "data", item)
	}
	err = r.redisClient.Set(r.ctx, redisKeyPrefix+r.generateUniqueID(), data, expiration).Err()
	if err != nil {
		r.logger.Error("Error saving data to Redis", "error", err, "data", item)
	}
	return err
}

// LoadTodayNewsItems загружает из Redis все структуры NewsItem, сохраненные на сегодняшнюю дату.
func (r *Redis) LoadTodayNews() ([][]byte, []error) {
	pattern := fmt.Sprintf("%s:*", redisKeyPrefix)
	keys, err := r.redisClient.Keys(r.ctx, pattern).Result()
	if err != nil {
		r.logger.Error("Error getting keys from Redis", "error", err)
		return nil, []error{fmt.Errorf("ошибка при получении ключей из Redis: %w", err)}
	}

	var newsItems [][]byte
	var errors []error
	for _, key := range keys {
		item, err := r.redisClient.Get(r.ctx, key).Result()
		if err != nil {
			r.logger.Error("Error getting value from Redis", "error", err)
			errors = append(errors, fmt.Errorf("ошибка при получении значения из Redis (ключ: %s): %w", key, err))
			continue
		}
		newsItems = append(newsItems, []byte(item))
	}
	return newsItems, errors
}

// generateUniqueID генерирует уникальный идентификатор.
func (r *Redis) generateUniqueID() string {
	return uuid.NewString()
}
