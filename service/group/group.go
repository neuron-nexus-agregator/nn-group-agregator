package group

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/copier"

	"agregator/group/internal/model/db/feed"
	"agregator/group/service/vector"
)

type Vectorizer interface {
	GetEmbedding(title, description, full_text string) (*vector.Vector, error)
}

type DBWorker interface {
	Insert(group *Group) (uint64, error)
	InsertCompares(groupID uint64, compareID uint64) error
	UpdateDate(groupID uint64, date time.Time, feed_id uint64) error
	UpdateRT(groupID uint64, isRT bool) error
	GetRTWords() ([]string, error)
	UpdateEmbedding(id uint64, pqVec string) error
}

type Group struct {
	ID          uint64
	Content_id  uint64
	centroid    *vector.Vector
	Date        time.Time
	texts       []string
	vervorizer  Vectorizer
	mu          sync.Mutex
	db          DBWorker
	LastDate    time.Time
	isTatarstan bool
	wordSet     map[string]struct{}
	alpha       float64
	minDiff     float64
	maxDistance float64
}

type JSONGroup struct {
	ID          uint64    `json:"id"`
	Content_id  uint64    `json:"content_id"`
	Vector      []float64 `json:"vector"`
	Date        time.Time `json:"date"`
	Texts       []string  `json:"texts"`
	LastDate    time.Time `json:"last_date"`
	IsTatarstan bool      `json:"is_tatarstan"`
	Alpha       float64   `json:"alpha"`
	MinDiff     float64   `json:"min_diff"`
	MaxDistance float64   `json:"max_distance"`
}

func New(vervorizer Vectorizer, db DBWorker, item feed.Model, minDiff, maxDistance, alpha float64) (*Group, error) {
	// Очистка и объединение текста
	text := strings.TrimSpace(item.Title + "\n\n" + item.Description + "\n\n" + item.FullText)

	// Создаем группу
	group := &Group{
		Content_id:  item.ID,
		vervorizer:  vervorizer,
		db:          db,
		Date:        item.Time,
		LastDate:    item.Time,
		isTatarstan: false,
		mu:          sync.Mutex{},
		alpha:       alpha,
		minDiff:     minDiff,
		maxDistance: maxDistance,
	}

	// Загружаем слова для проверки Татарстана
	if err := group.loadRTWords(); err != nil {
		log.Printf("Error loading RT words: %v", err)
	}

	// Проверяем на принадлежность к Татарстану
	group.isTatarstan = group.checkForTatarstan(text)

	// Добавляем текст и вектор
	group.texts = append(group.texts, text)
	vec, err := vervorizer.GetEmbedding(item.Title, item.Description, item.FullText)
	if err != nil {
		return nil, err
	}
	group.centroid = vec
	group.centroid.Divide(float64(len(group.texts)))
	// Вставляем группу в базу данных
	groupID, err := db.Insert(group)
	if err != nil {
		log.Printf("Error inserting group: %v", err)
		return group, err
	}

	if groupID == 0 {
		return nil, fmt.Errorf("error inserting group: groupID is 0")
	}

	group.ID = groupID
	// Добавляем связь группы с элементом
	if err := db.InsertCompares(groupID, item.ID); err != nil {
		return group, err
	}

	return group, nil
}

func (g *Group) loadRTWords() error {
	words, err := g.db.GetRTWords()
	if err != nil {
		return err
	}

	// Преобразуем слова в нижний регистр для оптимизации
	g.wordSet = make(map[string]struct{}, len(words))
	for _, word := range words {
		g.wordSet[strings.ToLower(strings.TrimSpace(word))] = struct{}{}
	}
	return nil
}

func NewFromJSON(jsonBytes []byte, vervorizer Vectorizer, db DBWorker) (*Group, error) {
	var jsonGroup JSONGroup
	err := json.Unmarshal(jsonBytes, &jsonGroup)
	if err != nil {
		return nil, err
	}
	vec := vector.New(jsonGroup.Vector)
	texts := jsonGroup.Texts
	vec.Divide(float64(len(texts)))
	group := &Group{
		ID:          jsonGroup.ID,
		Content_id:  jsonGroup.Content_id,
		centroid:    vec,
		Date:        jsonGroup.Date,
		texts:       texts,
		vervorizer:  vervorizer,
		mu:          sync.Mutex{},
		db:          db,
		LastDate:    jsonGroup.LastDate,
		isTatarstan: jsonGroup.IsTatarstan,
		alpha:       jsonGroup.Alpha,
		minDiff:     jsonGroup.MinDiff,
		maxDistance: jsonGroup.MaxDistance,
	}
	return group, nil
}

func (g *Group) Add(vec *vector.Vector, item feed.Model) error {
	if item.Title == "" || item.Description == "" || item.FullText == "" {
		return fmt.Errorf("empty item")
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	item.Title = strings.TrimSpace(item.Title)
	item.Description = strings.TrimSpace(item.Description)
	item.FullText = strings.TrimSpace(item.FullText)
	text := item.Title + "\n\n" + item.Description + "\n\n" + item.FullText
	rt := g.checkForTatarstan(text)

	if item.Time.After(g.Date) {
		err := g.db.UpdateDate(g.ID, item.Time, item.ID)
		if err != nil {
			return err
		}
		g.Date = item.Time
		g.LastDate = item.Time
	}

	err := g.db.InsertCompares(g.ID, item.ID)
	if err != nil {
		return err
	}

	if rt && !g.isTatarstan {
		g.isTatarstan = true
		err = g.db.UpdateRT(g.ID, g.isTatarstan)
		if err != nil {
			log.Default().Println("Error updating RT:", err.Error())
			log.Default().Println("Group ID:", g.ID)
			log.Default().Println("Item ID:", item.ID)
			log.Default().Println("Text:", text)
			log.Default().Println("RT:", rt)
			log.Default().Println("IsTatarstan:", g.isTatarstan)
			log.Default().Println("Date:", g.Date)
			return err
		}
	}

	text = strings.TrimSpace(text)
	g.texts = append(g.texts, text)

	return g.updateVector(vec)
}

func (g *Group) IsTatarstan() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isTatarstan
}

func (g *Group) Vector() *vector.Vector {
	vec := &vector.Vector{}
	err := copier.Copy(vec, g.centroid)
	if err != nil {
		log.Default().Println("Error copying vector:", err.Error())
		return nil
	}
	return vec.Multiply(float64(len(g.texts)))
}

func (g *Group) CheckCompare(vec *vector.Vector) bool {
	accept := g.centroid.CosDistance(vec) >= g.calculateDynamicThreshold()
	return accept
	// if !accept {
	// 	return false
	// }
	// return g.centroid.EuclideanDistance(vec) >= g.calculateDynamycTresholdForDistance()
}

func (g *Group) Length() int {
	return len(g.texts)
}

func (g *Group) Centroid() *vector.Vector {
	vec := &vector.Vector{}
	err := copier.Copy(vec, g.centroid)
	if err != nil {
		log.Default().Println("Error copying vector:", err.Error())
		return nil
	}
	return vec
}

func (g *Group) ToJSON() ([]byte, error) {

	vector := g.Vector()

	return json.Marshal(JSONGroup{
		ID:          g.ID,
		Content_id:  g.Content_id,
		Vector:      vector.GetArray(),
		Date:        g.Date,
		Texts:       g.texts,
		LastDate:    g.LastDate,
		IsTatarstan: g.isTatarstan,
		Alpha:       g.alpha,
		MinDiff:     g.minDiff,
		MaxDistance: g.maxDistance,
	})
}
