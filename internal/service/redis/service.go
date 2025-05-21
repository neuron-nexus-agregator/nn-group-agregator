package redis

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

type Redis struct {
	redisClient *redis.Client
	ctx         context.Context
}

const (
	redisKeyPrefix = "news:"
)

// New создает новый экземпляр Redis с проверкой подключения.
func New() (*Redis, error) {
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
		log.Printf("Ошибка подключения к Redis: %v", err)
		return nil, err
	}

	r := &Redis{
		redisClient: redisClient,
		ctx:         context.Background(),
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
		return fmt.Errorf("ошибка при выполнении конвейерных команд SET: %w", err)
	}
	return nil
}

// SaveNewsItems сохраняет массив структур NewsItem в Redis с указанной датой.
func (r *Redis) SavNews(item interface{}) error {

	expiration := 48 * time.Hour
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("ошибка при маршалинге данных: %w", err)
	}
	err = r.redisClient.Set(r.ctx, redisKeyPrefix+r.generateUniqueID(), data, expiration).Err()
	return err
}

// LoadTodayNewsItems загружает из Redis все структуры NewsItem, сохраненные на сегодняшнюю дату.
func (r *Redis) LoadTodayNews() ([][]byte, []error) {
	pattern := fmt.Sprintf("%s:*", redisKeyPrefix)
	keys, err := r.redisClient.Keys(r.ctx, pattern).Result()
	if err != nil {
		return nil, []error{fmt.Errorf("ошибка при получении ключей из Redis: %w", err)}
	}

	var newsItems [][]byte
	var errors []error
	for _, key := range keys {
		item, err := r.redisClient.Get(r.ctx, key).Result()
		if err != nil {
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
