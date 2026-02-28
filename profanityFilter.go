package main

import (
	"strings"
)

func profanityFilter(text string) string {
	words := strings.Split(text, " ")
	for i, v := range words {
		lower := strings.ToLower(v)
		if lower == "kerfuffle" || lower == "sharbert" || lower == "fornax" {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}
