package embedding

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	cfg "agregator/group/internal/config/yandex/embedding"
	"agregator/group/internal/interfaces"
	"agregator/group/service/vector"
)

type Service struct {
	maxRequests int
	mu          sync.Mutex
	cond        *sync.Cond
	client      *http.Client
	logger      interfaces.Logger
}

func New(logger interfaces.Logger) *Service {
	maxReqStr := os.Getenv("MAX_REQUESTS")
	maxReq, err := strconv.Atoi(maxReqStr)
	if err != nil {
		logger.Info("Invalid MAX_REQUESTS value, defaulting to 10")
		maxReq = 10
	}

	service := &Service{
		maxRequests: maxReq,
		client: &http.Client{
			Timeout: 60 * time.Second, // Таймаут для HTTP-запросов
		},
		logger: logger,
	}
	service.cond = sync.NewCond(&service.mu)
	return service
}

func (s *Service) wait() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.maxRequests <= 0 {
		s.cond.Wait()
	}
	s.maxRequests--
}

func (s *Service) release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maxRequests++
	s.cond.Signal()
}

func (s *Service) sendRequest(text string) (cfg.Response, error) {
	s.wait()
	defer s.release()

	request := cfg.Request{
		ModelURI: cfg.MODEL_URI,
		Text:     text,
	}
	data, err := json.Marshal(&request)
	if err != nil {
		s.logger.Error("Error marshaling request", "error", err)
		return cfg.Response{}, err
	}

	req, err := http.NewRequest("POST", cfg.URL, bytes.NewReader(data))
	if err != nil {
		s.logger.Error("Error creating request", "error", err)
		return cfg.Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Api-Key "+cfg.TOKEN)
	req.Header.Set("X-Folder-Id", cfg.FOLDER_ID)

	response, err := s.client.Do(req)
	if err != nil {
		s.logger.Error("Error sending request", "error", err)
		return cfg.Response{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		s.logger.Info("Error response from API", "status", response.StatusCode, "body", string(body))
		return cfg.Response{}, err
	}

	var apiResponse cfg.Response
	err = json.NewDecoder(response.Body).Decode(&apiResponse)
	if err != nil {
		s.logger.Error("Error decoding response", "error", err)
		return cfg.Response{}, err
	}
	return apiResponse, nil
}

func (s *Service) GetEmbedding(title, description, full_text string) (*vector.Vector, error) {
	if full_text == description {
		full_text = ""
	}
	text := strings.TrimSpace(title + "\n\n" + description + "\n\n" + full_text)
	if len(text) > 4000 {
		text = title + description
	}
	if len(text) > 4000 {
		text = title
	}
	response, err := s.sendRequest(text)
	if err != nil {
		s.logger.Error("Error getting embedding", "error", err)
		return vector.NewZeroVector(1), err
	}
	return vector.New(response.Embedding), nil
}

func (s *Service) GetSimilarity(text1, text2 *vector.Vector) float64 {
	return text1.CosDistance(text2)
}
