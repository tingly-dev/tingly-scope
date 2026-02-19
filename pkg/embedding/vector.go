package embedding

import "math"

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 computes square root for float32.
func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// Float64To32 converts []float64 to []float32.
func Float64To32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}

// Float32To64 converts []float32 to []float64.
func Float32To64(in []float32) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(v)
	}
	return out
}

// Normalize normalizes a vector to unit length.
func Normalize(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	var norm float32
	for _, x := range v {
		norm += x * x
	}
	norm = sqrt32(norm)

	if norm == 0 {
		return v
	}

	result := make([]float32, len(v))
	for i, x := range v {
		result[i] = x / norm
	}
	return result
}
