package gcs

import "math"

const M = 784931.0

type MatchType uint8

const (
	MatchBlock MatchType = iota

	MatchZip

	MatchHash
)

func (m MatchType) String() string {
	switch m {
	case MatchBlock:
		return "Block"
	case MatchZip:
		return "Zip"
	case MatchHash:
		return "Hash"
	default:
		return "Unknown"
	}
}

const (
	CostSort = 130.0

	CostRead = 30.0

	CostInsert = 45.0

	CostLookup = 30.0

	CostComp = 1.0

	CostKey = 3.0
)

func Optimize(querySize, filterSize int) (MatchType, float64) {
	Q := float64(querySize)
	N := float64(filterSize)

	expQ := expQueries(Q)
	expN := expReads(Q, N, expQ)

	cZip := costZip(Q, expQ, expN)
	cHash := costHash(Q, N, expQ)

	if cZip < cHash {
		return MatchZip, cZip / cHash
	}
	return MatchHash, cHash / cZip
}

func costZip(q, expQ, expN float64) float64 {
	return q*CostKey +
		q*math.Log2(q)*CostSort +
		(expQ+expN)*CostComp +
		expN*CostRead
}

func costHash(q, n, expQ float64) float64 {
	return n*(CostInsert+CostRead) +
		expQ*CostKey +
		expQ*CostLookup
}

func expQueries(q float64) float64 {
	return M * (1 - math.Exp(-q/M))
}

func expQuerySlot(q, n, expQ float64) float64 {
	return (n * M / (q + 1)) * expQ
}

func expReads(q, n, expQ float64) float64 {
	return ((n + 1) / (q + 1)) * expQ
}
