// word-game picks a random letter and four different categories for a round of the
// "name a word in each category" party game.
package main

import (
	"fmt"
	"math/rand/v2"
)

var categories = []string{
	"Spicy food", "Languages", "Kitchen", "Furniture", "Flowers", "Countries",
	"Musical instruments", "Animals", "Cartoon characters", "Four-letter words",
	"Software", "Movie titles", "Book titles", "Drinks", "Body parts", "Board games", "Superheroes",
}

// Letters that are hard to play (Q, X, Z...) are left out.
const letters = "ABCDEFGHIJKLMNOPRSTW"

func round(r *rand.Rand, n int) (byte, []string) {
	idx := r.Perm(len(categories))[:n]
	picked := make([]string, n)
	for i, k := range idx {
		picked[i] = categories[k]
	}
	return letters[r.IntN(len(letters))], picked
}

func main() {
	// A party game does not need cryptographic randomness.
	letter, cats := round(rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), 4) // #nosec G404
	fmt.Printf("Letter: %c\n", letter)
	for i, c := range cats {
		fmt.Printf("%d. %s\n", i+1, c)
	}
}
