package rtchecker

import (
	"agregator/group/internal/interfaces"
	model "agregator/group/internal/model/kafka"
	"agregator/group/internal/service/db"
	"strings"
	"sync"
	"time"
)

type Service struct {
	words  []string
	db     *db.DB
	logger interfaces.Logger
	mu     sync.Mutex
}

func New(logger interfaces.Logger) *Service {
	db, err := db.New(1)
	if err != nil {
		logger.Error("Error creating db", "error", err)
		return nil
	}
	words, err := db.GetRTWords()
	if err != nil {
		logger.Error("Error getting RT words", "error", err)
		return nil
	}
	return &Service{
		words:  words,
		mu:     sync.Mutex{},
		db:     db,
		logger: logger,
	}
}

func (s *Service) CheckForRT(item *model.News) bool {
	item.IsRT = false
	full := s.getTextCore(0.2, item)
	full = strings.ToLower(full)
	for _, word := range s.words {
		word := strings.ToLower(word)
		if strings.HasPrefix(full, word) {
			return true
		}
		if strings.HasSuffix(full, word) {
			return true
		}
		if strings.Contains(full, " "+word) {
			return true
		}
	}
	return false
}

func (s *Service) getTextCore(ratio float64, item *model.News) string {
	core := item.Title + " " + item.Description

	runes := []rune(item.FullText)
	length := len(runes)

	if ratio > 1 || ratio < 0 {
		ratio = 0.2
	}

	cutoff := int(float64(length) * ratio)
	if cutoff < 1 {
		cutoff = 1
	}

	core += " " + string(runes[:cutoff])

	return core
}

func (s *Service) UpdateWords(interval time.Duration) {
	timer := time.NewTicker(interval)
	defer timer.Stop()
	for range timer.C {
		words, err := s.db.GetRTWords()
		if err != nil {
			s.logger.Error("Error getting RT words", "error", err)
			return
		}
		s.mu.Lock()
		s.words = words
		s.mu.Unlock()
	}
}
