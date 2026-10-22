package main

import (
	"math/rand/v2"
	"strings"
	"testing"
)

func TestRoundPicksDistinctCategories(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 100; i++ {
		letter, cats := round(r, 4)
		if !strings.ContainsRune(letters, rune(letter)) {
			t.Fatalf("unexpected letter %c", letter)
		}
		seen := map[string]bool{}
		for _, c := range cats {
			if seen[c] {
				t.Fatalf("duplicate category %q in %v", c, cats)
			}
			seen[c] = true
		}
	}
}
