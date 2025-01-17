package growthbook

import (
	"hash/fnv"
	"net/url"
	"strconv"
)

const epsilon = 0.01

// VariationRange represents a single bucket range.
type VariationRange struct {
	Min float64
	Max float64
}

// Returns an array of floats with numVariations items that are all
// equal and sum to 1.
func getEqualWeights(numVariations int) []float64 {
	if numVariations < 0 {
		numVariations = 0
	}
	equal := make([]float64, numVariations)
	for i := range equal {
		equal[i] = 1.0 / float64(numVariations)
	}
	return equal
}

// This converts an experiment's coverage and variation weights into
// an array of bucket ranges.
func getBucketRanges(numVariations int, coverage float64, weights []float64) []VariationRange {
	// Make sure coverage is within bounds.
	if coverage < 0 {
		logWarn(WarnExpCoverageMustBePositive)
		coverage = 0
	}
	if coverage > 1 {
		logWarn(WarnExpCoverageMustBeFraction)
		coverage = 1
	}

	// Default to equal weights if missing or invalid
	if len(weights) == 0 {
		weights = getEqualWeights(numVariations)
	}
	if len(weights) != numVariations {
		logWarn(WarnExpWeightsWrongLength)
		weights = getEqualWeights(numVariations)
	}

	// If weights don't add up to 1 (or close to it), default to equal weights
	totalWeight := 0.0
	for i := range weights {
		totalWeight += weights[i]
	}
	if totalWeight < 1-epsilon || totalWeight > 1+epsilon {
		logWarn(WarnExpWeightsWrongTotal)
		weights = getEqualWeights(numVariations)
	}

	// Convert weights to ranges
	cumulative := 0.0
	ranges := make([]VariationRange, len(weights))
	for i := range weights {
		start := cumulative
		cumulative += weights[i]
		ranges[i] = VariationRange{start, start + coverage*weights[i]}
	}
	return ranges
}

// Given a hash and bucket ranges, assigns one of the bucket ranges.
func chooseVariation(n float64, ranges []VariationRange) int {
	for i := range ranges {
		if n >= ranges[i].Min && n < ranges[i].Max {
			return i
		}
	}
	return -1
}

// Checks if an experiment variation is being forced via a URL query
// string.
//
// As an example, if the id is "my-test" and url is
// http://localhost/?my-test=1, this function returns 1.
func getQueryStringOverride(id string, url *url.URL, numVariations int) *int {
	v, ok := url.Query()[id]
	if !ok || len(v) > 1 {
		return nil
	}

	vi, err := strconv.Atoi(v[0])
	if err != nil {
		return nil
	}

	if vi < 0 || vi >= numVariations {
		return nil
	}

	return &vi
}

// Namespace specifies what part of a namespace an experiment
// includes. If two experiments are in the same namespace and their
// ranges don't overlap, they wil be mutually exclusive.
type Namespace struct {
	ID    string
	Start float64
	End   float64
}

// Determine whether a user's ID lies within a given namespace.
func inNamespace(userID string, namespace *Namespace) bool {
	n := float64(hashFnv32a(userID+"__"+namespace.ID)%1000) / 1000
	return n >= namespace.Start && n < namespace.End
}

// Simple wrapper around Go standard library FNV32a hash function.
func hashFnv32a(s string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(s))
	return hash.Sum32()
}

// This function imitates Javascript's "truthiness" evaluation for Go
// values of unknown type.
func truthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case string:
		return val != ""
	case bool:
		return val
	case int:
		return val != 0
	case uint:
		return val != 0
	case float32:
		return val != 0
	case float64:
		return val != 0
	}
	return true
}
