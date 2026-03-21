package suggest

import "sort"

// LevenshteinDistance computes the minimum edit distance between two strings.
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	ra := []rune(a)
	rb := []rune(b)
	la, lb := len(ra), len(rb)

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			best := ins
			if del < best {
				best = del
			}
			if sub < best {
				best = sub
			}
			curr[j] = best
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

type suggestion struct {
	name     string
	distance int
}

// Suggest returns candidate strings within maxDistance edits of input,
// sorted by edit distance (ascending), then alphabetically.
// Exact matches (distance 0) are excluded.
func Suggest(input string, candidates []string, maxDistance int) []string {
	var matches []suggestion
	for _, c := range candidates {
		d := LevenshteinDistance(input, c)
		if d > 0 && d <= maxDistance {
			matches = append(matches, suggestion{name: c, distance: d})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		return matches[i].name < matches[j].name
	})
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.name
	}
	return out
}
