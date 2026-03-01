package resolve

import "strings"

func similarity(query, target string) float64 {
	q := strings.ToLower(strings.TrimSpace(query))
	t := strings.ToLower(strings.TrimSpace(target))
	if q == "" || t == "" {
		return 0
	}
	if q == t {
		return 1.0
	}
	if strings.HasPrefix(t, q) {
		penalty := float64(len(t)-len(q)) / float64(max(len(t), 1))
		s := 0.93 - penalty*0.15
		if s < 0 {
			return 0
		}
		return s
	}
	if strings.Contains(t, q) {
		return 0.75
	}
	sub := subsequenceScore(q, t)
	lev := levenshteinScore(q, t)
	return maxFloat(sub*0.45+lev*0.55, 0)
}

func subsequenceScore(query, target string) float64 {
	qi := 0
	ti := 0
	match := 0
	for qi < len(query) && ti < len(target) {
		if query[qi] == target[ti] {
			match++
			qi++
		}
		ti++
	}
	if len(query) == 0 {
		return 0
	}
	return float64(match) / float64(len(query))
}

func levenshteinScore(a, b string) float64 {
	d := levenshtein(a, b)
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1
	}
	return 1.0 - float64(d)/float64(maxLen)
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(
				curr[j-1]+1,
				prev[j]+1,
				prev[j-1]+cost,
			)
		}
		copy(prev, curr)
	}
	return prev[len(b)]
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
