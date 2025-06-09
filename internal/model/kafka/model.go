package kafka

import (
	"time"
)

type Item struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	PubDate     *time.Time `json:"pubDate"`
	Description string     `json:"description"`
	FullText    string     `json:"fullText"`
	Name        string     `json:"name"`
	Link        string     `json:"link"`
	MD5         string     `json:"md5"`
	Enclosure   string     `json:"enclosure"`
	Category    string     `json:"category"`
	Changed     bool       `json:"changed"`
}

type News struct {
	MD5         string    `json:"-"`
	ID          int64     `json:"id"`
	ClusterID   int64     `json:"cluster_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	FullText    string    `json:"full_text"`
	Enclosure   string    `json:"enclosure,omitempty"`
	Embedding   []float64 `json:"embedding"`
	PublishDate string    `json:"publish_date"` // Должен быть в формате RFC3339 или Unix Epoch Millis
	IsRT        bool      `json:"is_rt"`
	SourceName  string    `json:"source_name"`
	URL         string    `json:"url"`
}

type Cluster struct {
	ID        int64   `json:"cluster_id"`
	NewsCount int64   `json:"news_count"`
	Distance  float64 `json:"distance"`
	IsRT      bool    `json:"is_rt"`
}
