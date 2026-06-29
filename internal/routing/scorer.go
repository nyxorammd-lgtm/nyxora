package routing

import (
	"math"
	"sort"
)

type PathScore struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Score    float64 `json:"score"`
	Latency  float64 `json:"latency"`
	Jitter   float64 `json:"jitter"`
	Loss     float64 `json:"packet_loss"`
	Stability float64 `json:"stability"`
	Bandwidth int     `json:"bandwidth"`
}

type Scorer struct {
	weights Weights
}

type Weights struct {
	LatencyWeight  float64
	LossWeight     float64
	JitterWeight   float64
	StabilityWeight float64
	BandwidthWeight float64
}

var DefaultWeights = Weights{
	LatencyWeight:   0.30,
	LossWeight:      0.25,
	JitterWeight:    0.15,
	StabilityWeight: 0.20,
	BandwidthWeight: 0.10,
}

func NewScorer() *Scorer {
	return &Scorer{weights: DefaultWeights}
}

func NewScorerWithWeights(w Weights) *Scorer {
	return &Scorer{weights: w}
}

func (s *Scorer) Score(latency, jitter, loss, stability float64, bandwidth int) float64 {
	if loss > 50 {
		return 0
	}
	if latency <= 0 {
		return 5
	}

	latencyScore := math.Max(0, 100-latency/2)
	lossScore := math.Max(0, 100-loss*2)
	jitterScore := math.Max(0, 100-jitter*3)
	stabilityScore := stability * 100
	bwScore := math.Min(100, float64(bandwidth)/10)

	return latencyScore*s.weights.LatencyWeight +
		lossScore*s.weights.LossWeight +
		jitterScore*s.weights.JitterWeight +
		stabilityScore*s.weights.StabilityWeight +
		bwScore*s.weights.BandwidthWeight
}

func (s *Scorer) Rank(scores []PathScore) []PathScore {
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})
	return scores
}

func (s *Scorer) Best(scores []PathScore) *PathScore {
	if len(scores) == 0 {
		return nil
	}
	ranked := s.Rank(scores)
	return &ranked[0]
}
