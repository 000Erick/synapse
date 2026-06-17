package domain

import "sort"

// RRF implements Reciprocal Rank Fusion over multiple ranked lists.
// k is the RRF constant (typically 60).
// Rank is 0-based within each list.
func RRF(lists [][]Ranked, k int) []Ranked {
	scores := make(map[int64]float64)
	sources := make(map[int64]Source)

	for _, list := range lists {
		for rank, item := range list {
			scores[item.ID] += 1.0 / float64(k+rank+1)
			sources[item.ID] |= item.Source
		}
	}

	result := make([]Ranked, 0, len(scores))
	for id, score := range scores {
		result = append(result, Ranked{
			ID:     id,
			Score:  score,
			Source: sources[id],
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	return result
}
