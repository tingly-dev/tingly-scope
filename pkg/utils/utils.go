package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// Timestamp returns a formatted timestamp string
func Timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}

// TimestampWithRandom returns a timestamp with a random suffix
func TimestampWithRandom() string {
	return fmt.Sprintf("%s_%03x", Timestamp(), rand.Intn(0x1000))
}

// GenerateID creates a unique identifier
func GenerateID() string {
	return fmt.Sprintf("%s_%s", time.Now().Format("20060102150405"), randString(8))
}

// GenerateIDFromText creates a deterministic ID from text
func GenerateIDFromText(text string) string {
	// Simple hash-based ID generation
	h := 0
	for _, c := range text {
		h = h*31 + int(c)
	}
	return fmt.Sprintf("text_%x", h)
}

// randString generates a random string of length n
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// MaxInt returns the maximum of two integers
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
