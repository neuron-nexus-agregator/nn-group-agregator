package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	model "agregator/group/internal/model/kafka"

	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	textReader  *kafka.Reader
	textChannel chan model.News
	brokers     []string
	writeTopic  string
	writer      *kafka.Writer
}

func New(brokers []string, groupID, textTopic string, writeTopic string) *Kafka {
	textReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   textTopic,
	})

	writer := &kafka.Writer{
		Addr:      kafka.TCP(brokers...),
		Topic:     writeTopic,
		Balancer:  &kafka.LeastBytes{},
		BatchSize: 10,
	}

	return &Kafka{
		textReader:  textReader,
		textChannel: make(chan model.News, 100),
		brokers:     brokers,
		writeTopic:  writeTopic,
		writer:      writer,
	}
}

func (k *Kafka) TextOutput() <-chan model.News {
	return k.textChannel
}

func (k *Kafka) StartReadingText(ctx context.Context) {
	defer func() {
		k.textReader.Close()
		close(k.textChannel) // Закрываем канал при завершении
	}()

	for {
		select {
		case <-ctx.Done(): // Завершаем чтение, если контекст отменен
			return
		default:
			msg, err := k.textReader.ReadMessage(ctx)
			if err != nil {
				continue
			}
			log.Default().Println("Reading from Kafka", "data", string(msg.Value))
			item := model.Item{}
			err = json.Unmarshal(msg.Value, &item)
			if err != nil {
				continue
			}
			if item.Changed {
				continue
			}
			news := model.News{
				MD5:         item.MD5,
				ID:          int64(item.ID),
				ClusterID:   0,
				Title:       item.Title,
				Description: item.Description,
				FullText:    item.FullText,
				Enclosure:   item.Enclosure,
				PublishDate: item.PubDate.Format(time.RFC3339),
				Embedding:   make([]float64, 0, 256),
				IsRT:        false,
				SourceName:  item.Name,
				URL:         item.Link,
			}

			select {
			case k.textChannel <- news: // Отправляем сообщение в канал
			case <-ctx.Done(): // Проверяем отмену контекста
				return
			}
		}
	}
}

func (k *Kafka) Write(data model.News) error {
	message, err := json.Marshal(data)
	if err != nil {
		return err
	}
	log.Default().Println("Writing to Kafka", "data", string(message))
	return k.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(data.MD5), // Используем MD5 как ключ
		Value: message,
	})

}
