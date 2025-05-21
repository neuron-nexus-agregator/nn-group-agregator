package groupmaker

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	model "agregator/group/internal/model/db/feed"
	"agregator/group/internal/service/db"
	"agregator/group/internal/service/embedding"
	cache "agregator/group/internal/service/redis"
	"agregator/group/service/group"
	"agregator/group/service/vector"
)

func cleanString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	// Удаление HTML-тегов
	re := regexp.MustCompile("<[^>]*>")
	s = re.ReplaceAllString(s, "")

	// Удаление множественных пробелов
	s = strings.Join(strings.Fields(s), " ")

	return s
}

type GroupMaker struct {
	groups            []*group.Group
	db                *db.DB
	vectorizer        *embedding.Service
	minDiff           float64
	alpha             float64
	maxDistance       float64
	mu                sync.RWMutex
	timeLife          time.Duration
	acceptOldGroups   bool
	noDeleteOldGroups bool
	cache             *cache.Redis
}

func NewGroupMaker(minDiff, maxDistance, alpha float64, timeLife time.Duration) *GroupMaker {
	maxConnStr := os.Getenv("MAX_REQUESTS")
	maxConn, err := strconv.Atoi(maxConnStr)
	if err != nil {
		maxConn = 10
	}

	acceptOldGroups := strings.ToLower(os.Getenv("ACCEPT_OLD_GROUPS")) == "true"
	noDeleteOldGroups := strings.ToLower(os.Getenv("NO_DELETE_OLD_GROUPS")) == "true"

	db, err := db.New(maxConn)
	if err != nil {
		log.Fatalf("Error creating DB instance: %v", err)
	}

	cache, err := cache.New()
	if err != nil {
		log.Fatalf("Error creating cache instance: %v", err)
	}

	vectorizer := embedding.New()

	groups, err := loadFromCache(cache, vectorizer, db)
	if err != nil {
		groups = make([]*group.Group, 0, 100)
	}

	return &GroupMaker{
		db:                db,
		vectorizer:        vectorizer,
		minDiff:           minDiff,
		alpha:             alpha,
		mu:                sync.RWMutex{},
		timeLife:          timeLife,
		acceptOldGroups:   acceptOldGroups,
		noDeleteOldGroups: noDeleteOldGroups,
		maxDistance:       maxDistance,
		groups:            groups,
		cache:             cache,
	}
}

func loadFromCache(cache *cache.Redis, vectorizer *embedding.Service, db *db.DB) ([]*group.Group, error) {
	items, errors := cache.LoadTodayNews()
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}
	groups := make([]*group.Group, 0, len(items))
	for _, item := range items {
		gr, err := group.NewFromJSON(item, vectorizer, db)
		if err != nil {
			log.Printf("Error creating group from JSON: %v", err)
			continue
		}
		groups = append(groups, gr)
	}
	return groups, nil
}

func (g *GroupMaker) correctItem(m model.Model) (title string, description string, full_text string) {
	full_text = strings.TrimSpace(cleanString(m.FullText))
	title = strings.TrimSpace(cleanString(m.Title))
	description = strings.TrimSpace(cleanString(m.Description))
	return title, description, full_text
}

func (g *GroupMaker) insertVector(vec *vector.Vector, m model.Model) error {
	if m.Title == "" || m.FullText == "" {
		return fmt.Errorf("empty item")
	}
	if m.Description == "" {
		m.Description = m.Title
	}

	// Проверяем существующие группы
	for _, gr := range g.groups {
		if gr.CheckCompare(vec) {
			gr.Add(vec, m)
			return nil
		}
	}

	// Если группа не найдена, создаем новую
	newGroup, err := group.New(g.vectorizer, g.db, m, g.minDiff, g.maxDistance, g.alpha)
	if err != nil {
		log.Printf("Error adding new group: %v", err)
		return err
	}
	err = g.cache.SavNews(newGroup)
	if err != nil {
		log.Printf("Error saving group to cache: %v", err)
		return err
	}
	g.groups = append(g.groups, newGroup)
	return nil
}

func (g *GroupMaker) deleteOld() {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()

	// Inplace-фильтрация
	n := 0
	for _, group := range g.groups {
		if now.Sub(group.LastDate) <= g.timeLife {
			g.groups[n] = group
			n++
		}
	}
	g.groups = g.groups[:n]
}

func (g *GroupMaker) UpdateGroups() error {
	newFeeds, err := g.db.Get()
	if err != nil {
		log.Printf("Error getting new feeds: %v", err)
		return err
	}

	if len(newFeeds) == 0 {
		log.Println("No new feeds to parse")
		return nil
	}
	log.Printf("Parsing %d feeds", len(newFeeds))
	parsedList := make([]uint64, 0, 20)
	// Обрабатываем фиды последовательно
	for _, feed := range newFeeds {
		if feed.Parsed {
			continue
		}
		if !g.processFeed(feed) {
			continue
		}
		if len(parsedList) >= 20 {
			g.db.UpdateParsedBatch(parsedList, true)
			parsedList = nil
			parsedList = make([]uint64, 0, 20)
		}
		feed.Parsed = true
		parsedList = append(parsedList, feed.ID)
	}

	if len(parsedList) > 0 {
		g.db.UpdateParsedBatch(parsedList, true)
	}

	log.Printf("Parsing complete for %d groups", len(g.groups))
	if !g.noDeleteOldGroups {
		g.deleteOld()
	}
	return nil
}

func (g *GroupMaker) processFeed(feed model.Model) bool {
	if feed.Parsed {
		return true
	}

	vec, err := g.vectorizer.GetEmbedding(g.correctItem(feed))
	if err != nil {
		log.Printf("Error getting embedding for item: %v", err)
		return false
	}

	if err := g.insertVector(vec, feed); err != nil {
		log.Printf("Error inserting vector for item: %v", err)
		return false
	}

	return true // Возвращаем true, если фид успешно обработан
}
