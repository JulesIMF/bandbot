package fuzzy

import "strings"

func levenshtein(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	la := len([]rune(a))
	lb := len([]rune(b))

	ra := []rune(a)
	rb := []rune(b)

	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			dp[i][j] = min3(
				dp[i-1][j]+1,
				dp[i][j-1]+1,
				dp[i-1][j-1]+cost,
			)
		}
	}
	return dp[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

type Match struct {
	Name     string
	Distance int
}

func FindCandidates(query string, names []string, threshold float64) []Match {
	var candidates []Match
	for _, name := range names {
		dist := levenshtein(query, name)
		maxLen := len([]rune(query))
		if nl := len([]rune(name)); nl > maxLen {
			maxLen = nl
		}
		if maxLen == 0 {
			continue
		}
		ratio := float64(dist) / float64(maxLen)
		if ratio <= threshold {
			candidates = append(candidates, Match{Name: name, Distance: dist})
		}
	}
	sortByDistance(candidates)
	return candidates
}

func sortByDistance(matches []Match) {
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && matches[j].Distance < matches[j-1].Distance; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}
}
