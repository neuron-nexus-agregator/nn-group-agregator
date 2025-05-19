package group

import (
	"fmt"
	"math"
	"strings"

	"agregator/group/service/vector"
)

func (g *Group) checkForTatarstan(texts ...string) bool {
	if len(texts) == 0 {
		return false
	}
	// Проверяем каждый текст
	for _, text := range texts {
		if g.containsTatarstanWord(text, g.wordSet) {
			return true
		}
	}

	return false
}

// containsTatarstanWord проверяет, содержит ли текст одно из слов из набора
func (g *Group) containsTatarstanWord(text string, wordSet map[string]struct{}) bool {
	text = strings.ToLower(strings.TrimSpace(text))

	for word := range wordSet {
		if strings.Contains(text, " "+word) ||
			strings.HasPrefix(text, word) ||
			strings.HasSuffix(text, word) ||
			strings.HasSuffix(text, word+".") ||
			strings.Contains(text, ">"+word) ||
			strings.Contains(text, "&nbsp;"+word) {
			return true
		}
	}

	return false
}

func (g *Group) updateVector(vec *vector.Vector) error {
	if len(g.texts) == 0 {
		return fmt.Errorf("empty vector: no texts available")
	}

	// Если это первый текст, просто устанавливаем вектор
	if len(g.texts) == 1 {
		g.centroid = vec
		return nil
	}

	// Обновляем вектор группы
	g.centroid.Multiply(float64(len(g.texts) - 1)).Add(vec).Divide(float64(len(g.texts)))
	return g.db.UpdateEmbedding(g.ID, g.centroid.ToPqString())
}

func (g *Group) calculateDynamicThreshold() float64 {
	// Пример: динамический порог - 0.5 * (max - min) + min
	adjustment := 0.0
	size := len(g.texts)
	if size < 3 {
		adjustment = 0.02 * float64(3-size)
		return math.Max(g.minDiff+adjustment, 0.75)
	}

	adjustment = 0.03 * math.Log10(float64(size))
	return math.Max(g.minDiff+adjustment, 0.95)
}

func (g *Group) calculateDynamycTresholdForDistance() float64 {
	size := len(g.texts)
	maxThreshold := 0.6
	minThreshold := 0.3
	rangeThreshold := maxThreshold - minThreshold

	if size < 3 {
		increaseFactor := float64(size-1) / 2.0 // 0 для size=1, 0.5 для size=2
		adjustment := rangeThreshold * increaseFactor
		return math.Max(g.maxDistance+adjustment, minThreshold)
	}

	adjustment := 0.03 * math.Log10(float64(size))
	return math.Min(math.Max(g.maxDistance+adjustment, minThreshold), maxThreshold)
}
