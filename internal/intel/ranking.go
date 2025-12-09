package intel

import (
	"sort"
	"time"
)

type Rankable interface {
	GetID() string
	GetRelevance() float64
	GetTimestamp() time.Time
	GetFrequency() int
	GetProximity() float64
}

type RankCriteria struct {
	RelevanceWeight  float64
	RecencyWeight    float64
	FrequencyWeight  float64
	ProximityWeight  float64
}

type RankItem struct {
	ID        string
	Relevance float64
	Timestamp time.Time
	Frequency int
	Proximity float64
	Score     float64
}

var DefaultRankCriteria = RankCriteria{
	RelevanceWeight:  0.4,
	RecencyWeight:    0.2,
	FrequencyWeight:  0.2,
	ProximityWeight:  0.2,
}

func Rank(items []Rankable, criteria RankCriteria) []Rankable {
	if len(items) == 0 {
		return items
	}

	rankItems := make([]RankItem, len(items))
	for i, item := range items {
		rankItems[i] = RankItem{
			ID:        item.GetID(),
			Relevance: item.GetRelevance(),
			Timestamp: item.GetTimestamp(),
			Frequency: item.GetFrequency(),
			Proximity: item.GetProximity(),
		}
	}

	normalizeScores(rankItems)

	for i := range rankItems {
		rankItems[i].Score = calculateRankScore(rankItems[i], criteria)
	}

	sort.Slice(rankItems, func(i, j int) bool {
		return rankItems[i].Score > rankItems[j].Score
	})

	result := make([]Rankable, len(rankItems))
	for i, rankItem := range rankItems {
		for _, original := range items {
			if original.GetID() == rankItem.ID {
				result[i] = original
				break
			}
		}
	}

	return result
}

func calculateRankScore(item RankItem, criteria RankCriteria) float64 {
	score := 0.0

	score += item.Relevance * criteria.RelevanceWeight
	score += normalizeRecency(item.Timestamp) * criteria.RecencyWeight
	score += normalizeFrequency(item.Frequency) * criteria.FrequencyWeight
	score += item.Proximity * criteria.ProximityWeight

	return score
}

func normalizeScores(items []RankItem) {
	if len(items) == 0 {
		return
	}

	maxRelevance := 0.0
	maxFrequency := 0

	for _, item := range items {
		if item.Relevance > maxRelevance {
			maxRelevance = item.Relevance
		}
		if item.Frequency > maxFrequency {
			maxFrequency = item.Frequency
		}
	}

	for i := range items {
		if maxRelevance > 0 {
			items[i].Relevance = items[i].Relevance / maxRelevance
		}
		items[i].Proximity = normalizeProximity(items[i].Proximity)
	}
}

func normalizeRecency(timestamp time.Time) float64 {
	now := time.Now()
	hoursSince := now.Sub(timestamp).Hours()

	if hoursSince < 24 {
		return 1.0
	}
	if hoursSince < 168 {
		return 0.8
	}
	if hoursSince < 720 {
		return 0.6
	}
	if hoursSince < 2160 {
		return 0.4
	}

	return 0.2
}

func normalizeFrequency(frequency int) float64 {
	switch {
	case frequency >= 100:
		return 1.0
	case frequency >= 50:
		return 0.8
	case frequency >= 20:
		return 0.6
	case frequency >= 5:
		return 0.4
	default:
		return float64(frequency) / 5.0
	}
}

func normalizeProximity(proximity float64) float64 {
	if proximity > 1.0 {
		proximity = 1.0
	}
	if proximity < 0 {
		proximity = 0
	}
	return proximity
}

func RankByRelevance(items []Rankable) []Rankable {
	criteria := RankCriteria{
		RelevanceWeight: 1.0,
	}
	return Rank(items, criteria)
}

func RankByRecency(items []Rankable) []Rankable {
	criteria := RankCriteria{
		RecencyWeight: 1.0,
	}
	return Rank(items, criteria)
}

func RankByFrequency(items []Rankable) []Rankable {
	criteria := RankCriteria{
		FrequencyWeight: 1.0,
	}
	return Rank(items, criteria)
}

type SimpleRankable struct {
	id        string
	relevance float64
	timestamp time.Time
	frequency int
	proximity float64
}

func (s *SimpleRankable) GetID() string           { return s.id }
func (s *SimpleRankable) GetRelevance() float64  { return s.relevance }
func (s *SimpleRankable) GetTimestamp() time.Time { return s.timestamp }
func (s *SimpleRankable) GetFrequency() int      { return s.frequency }
func (s *SimpleRankable) GetProximity() float64  { return s.proximity }

func NewSimpleRankable(id string, relevance float64, timestamp time.Time, frequency int, proximity float64) Rankable {
	return &SimpleRankable{
		id:        id,
		relevance: relevance,
		timestamp: timestamp,
		frequency: frequency,
		proximity: proximity,
	}
}

func FilterByThreshold(items []Rankable, scoreThreshold float64, criteria RankCriteria) []Rankable {
	ranked := Rank(items, criteria)
	var filtered []Rankable

	for _, item := range ranked {
		score := calculateItemScore(item, criteria)
		if score >= scoreThreshold {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func calculateItemScore(item Rankable, criteria RankCriteria) float64 {
	relevance := item.GetRelevance()
	recency := normalizeRecency(item.GetTimestamp())
	frequency := normalizeFrequency(item.GetFrequency())
	proximity := normalizeProximity(item.GetProximity())

	score := 0.0
	score += relevance * criteria.RelevanceWeight
	score += recency * criteria.RecencyWeight
	score += frequency * criteria.FrequencyWeight
	score += proximity * criteria.ProximityWeight

	return score
}

func TopN(items []Rankable, n int, criteria RankCriteria) []Rankable {
	ranked := Rank(items, criteria)

	if len(ranked) < n {
		return ranked
	}

	return ranked[:n]
}

type StringRankable struct {
	value    string
	score    float64
	count    int
	lastSeen time.Time
}

func (s *StringRankable) GetID() string           { return s.value }
func (s *StringRankable) GetRelevance() float64  { return s.score }
func (s *StringRankable) GetTimestamp() time.Time { return s.lastSeen }
func (s *StringRankable) GetFrequency() int      { return s.count }
func (s *StringRankable) GetProximity() float64  { return calculateStringProximity(s.value) }

func calculateStringProximity(value string) float64 {
	length := float64(len(value))
	if length > 100 {
		return 0.3
	}
	if length > 50 {
		return 0.6
	}
	return 0.9
}

func RankStrings(values []string) []string {
	freq := make(map[string]int)
	for _, v := range values {
		freq[v]++
	}

	items := make([]Rankable, 0, len(freq))
	for value, count := range freq {
		items = append(items, &StringRankable{
			value:    value,
			score:    1.0,
			count:    count,
			lastSeen: time.Now(),
		})
	}

	ranked := RankByFrequency(items)

	result := make([]string, len(ranked))
	for i, item := range ranked {
		result[i] = item.GetID()
	}

	return result
}
