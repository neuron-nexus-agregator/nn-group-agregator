package elastic

import (
	"agregator/group/internal/model/kafka"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Elastic struct {
	url string
}

func New() *Elastic {
	return &Elastic{
		url: os.Getenv("ELASTIC_HOST"),
	}
}

func (e *Elastic) GetClosest(embedding []float64, limit int) ([]kafka.Cluster, error) {
	type Request struct {
		Embedding []float64 `json:"embedding"`
		Limit     int       `json:"limit"`
	}
	var req Request = Request{
		Embedding: embedding,
		Limit:     limit,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(e.url+"/get", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type Response struct {
		Items []kafka.Cluster `json:"items"`
	}
	var respStruct Response
	err = json.NewDecoder(resp.Body).Decode(&respStruct)
	if err != nil {
		return nil, err
	}
	return respStruct.Items, nil
}

func (e *Elastic) RegisterClusert(id int64, publishDate string, embedding []float64, title, fullText, description string) error {
	type Request struct {
		Id          int64     `json:"id"`
		PublishDate string    `json:"publishDate"`
		Embedding   []float64 `json:"embedding"`
		Title       string    `json:"title"`
		Rewrite     string    `json:"text"`
		Description string    `json:"description"`
	}
	var req Request = Request{
		Id:          id,
		PublishDate: publishDate,
		Embedding:   embedding,
		Title:       title,
		Rewrite:     fullText,
		Description: description,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := http.Post(e.url+"/register", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
