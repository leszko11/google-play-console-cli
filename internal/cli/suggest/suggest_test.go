package suggest

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "both empty", a: "", b: "", want: 0},
		{name: "first empty", a: "", b: "abc", want: 3},
		{name: "second empty", a: "abc", b: "", want: 3},
		{name: "identical", a: "release", b: "release", want: 0},
		{name: "single insertion", a: "ap", b: "app", want: 1},
		{name: "single deletion", a: "apps", b: "app", want: 1},
		{name: "single substitution", a: "apps", b: "aps", want: 1},
		{name: "transposition", a: "apps", b: "apsp", want: 2},
		{name: "completely different", a: "abc", b: "xyz", want: 3},
		{name: "case sensitive", a: "Apps", b: "apps", want: 1},
		{name: "prefix", a: "rel", b: "release", want: 4},
		{name: "longer edit", a: "kitten", b: "sitting", want: 3},
		{name: "unicode", a: "café", b: "cafe", want: 1},
		{name: "single char same", a: "a", b: "a", want: 0},
		{name: "single char diff", a: "a", b: "b", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestLevenshteinDistance_Symmetric(t *testing.T) {
	pairs := [][2]string{
		{"apps", "aps"},
		{"release", "releas"},
		{"", "hello"},
		{"kitten", "sitting"},
	}
	for _, p := range pairs {
		ab := LevenshteinDistance(p[0], p[1])
		ba := LevenshteinDistance(p[1], p[0])
		if ab != ba {
			t.Errorf("not symmetric: LevenshteinDistance(%q,%q)=%d but LevenshteinDistance(%q,%q)=%d",
				p[0], p[1], ab, p[1], p[0], ba)
		}
	}
}

func TestSuggest(t *testing.T) {
	candidates := []string{"apps", "auth", "release", "rollback", "reports", "reviews", "status", "deploy", "completion"}

	tests := []struct {
		name        string
		input       string
		candidates  []string
		maxDistance int
		want        []string
	}{
		{
			name:        "close typo",
			input:       "aps",
			candidates:  candidates,
			maxDistance: 2,
			want:        []string{"apps"},
		},
		{
			name:        "multiple matches sorted by distance",
			input:       "relase",
			candidates:  candidates,
			maxDistance: 3,
			want:        []string{"release"},
		},
		{
			name:        "no matches within distance",
			input:       "zzzzz",
			candidates:  candidates,
			maxDistance: 2,
			want:        nil,
		},
		{
			name:        "exact match excluded",
			input:       "apps",
			candidates:  candidates,
			maxDistance: 1,
			want:        []string{"aps"},
		},
		{
			name:        "empty input",
			input:       "",
			candidates:  candidates,
			maxDistance: 3,
			want:        nil,
		},
		{
			name:        "empty candidates",
			input:       "apps",
			candidates:  nil,
			maxDistance: 3,
			want:        nil,
		},
		{
			name:        "max distance 1 is strict",
			input:       "auht",
			candidates:  candidates,
			maxDistance: 1,
			want:        nil,
		},
		{
			name:        "max distance 2 catches transposition",
			input:       "auht",
			candidates:  candidates,
			maxDistance: 2,
			want:        []string{"auth"},
		},
		{
			name:        "alphabetical tiebreak",
			input:       "re",
			candidates:  []string{"rb", "ra", "rc"},
			maxDistance: 1,
			want:        []string{"ra", "rb", "rc"},
		},
		{
			name:        "distance sorting before alpha",
			input:       "deplpy",
			candidates:  candidates,
			maxDistance: 3,
			want:        []string{"deploy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "exact match excluded" test we need "aps" in candidates.
			cands := tt.candidates
			if tt.name == "exact match excluded" {
				cands = append([]string{"aps"}, cands...)
			}
			got := Suggest(tt.input, cands, tt.maxDistance)
			if !slicesEqual(got, tt.want) {
				t.Errorf("Suggest(%q, ..., %d) = %v, want %v", tt.input, tt.maxDistance, got, tt.want)
			}
		})
	}
}

func TestSuggest_NoEmptyResults(t *testing.T) {
	got := Suggest("xyz", []string{"abc", "def"}, 1)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
