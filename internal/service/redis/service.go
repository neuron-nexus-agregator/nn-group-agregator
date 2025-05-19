package redis

import (
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
	dateFormat     = "2006-01-02"
)

// New создает новый экземпляр Redis с проверкой подключения.
func New() (*Redis, error) {
	redisAddr := fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
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
func (r *Redis) SaveNews(items [][]byte, date time.Time) error {
	dateStr := date.Format(dateFormat)
	pipe := r.redisClient.Pipeline()
	expiration := 48 * time.Hour
	for _, item := range items {
		key := fmt.Sprintf("%s%s:%s", redisKeyPrefix, dateStr, r.generateUniqueID())
		pipe.Set(r.ctx, key, string(item), expiration)
	}
	if _, err := pipe.Exec(r.ctx); err != nil {
		return fmt.Errorf("ошибка при выполнении конвейерных команд SET: %w", err)
	}
	return nil
}

// LoadTodayNewsItems загружает из Redis все структуры NewsItem, сохраненные на сегодняшнюю дату.
func (r *Redis) LoadTodayNews() ([][]byte, []error) {
	today := time.Now().Format(dateFormat)
	pattern := fmt.Sprintf("%s%s:*", redisKeyPrefix, today)
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

func (r *Redis) DeleteOldNews() error {
	today := time.Now().Format(dateFormat)
	zsetKey := "news_dates"

	// Удаляем записи с датой раньше сегодняшней
	_, err := r.redisClient.ZRemRangeByScore(r.ctx, zsetKey, "-inf", today).Result()
	if err != nil {
		return fmt.Errorf("ошибка при удалении устаревших записей из ZSET: %w", err)
	}

	log.Println("Устаревшие записи успешно удалены из ZSET.")
	return nil
}
