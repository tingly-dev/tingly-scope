package embedding

import "errors"

var (
	// ErrUnavailable indicates the embedding provider is unavailable.
	ErrUnavailable = errors.New("embedding provider unavailable")

	// ErrInvalidInput indicates invalid input text.
	ErrInvalidInput = errors.New("invalid input text")

	// ErrRateLimited indicates rate limiting.
	ErrRateLimited = errors.New("rate limited")

	// ErrModelNotLoaded indicates the model is not loaded.
	ErrModelNotLoaded = errors.New("model not loaded")
)
