package main

import (
	"fmt"
	"math/rand"
)

func main() {
	letters := []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	}
	fmt.Println("Letter is:", letters[rand.Intn(len(letters))])
	categories := []string{
		"Spicy food", "Languages", "Kitchen", "Furniture", "Flowers", "Countries", "Musical Instruments", "Animals", "Cartoon Characters", "Four letter words", "Software", "Movie titles", "Book titles", "Drinks", "Body parts", "Broad games", "Superheroes",
	}
	for i := 0; i < 4; i++ {
		fmt.Println("Category is:", categories[rand.Intn(len(categories))])
	}
}
