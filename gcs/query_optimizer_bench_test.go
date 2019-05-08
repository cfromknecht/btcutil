package gcs

import (
	"fmt"
	prand "math/rand"
	"sort"
	"testing"

	"github.com/aead/siphash"
	"github.com/kkdai/bstream"
)

func TestOptimize(t *testing.T) {
	for _, n := range []int{1, 10, 100, 1000, 10000} {
		fmt.Printf("N: %d\t", n)
		if n < 10000 {
			fmt.Printf("\t")
		}
		for _, q := range []int{1, 10, 100, 1000, 10000, 100000, 1000000} {
			mType, r := Optimize(q, n)
			fmt.Printf("q:%d=%s (%.03f x)\t", q, mType, 1/r)
		}
		fmt.Println()
	}
}

func BenchmarkCostSort(t *testing.B) {
	s := make([]uint64, t.N)
	for i := range s {
		s[i] = uint64(prand.Int63())
	}

	t.ReportAllocs()
	t.ResetTimer()

	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
}

func BenchmarkCostInsert(t *testing.B) {
	s := make([]uint64, t.N)
	for i := range s {
		s[i] = uint64(prand.Int63())
	}

	t.ReportAllocs()
	t.ResetTimer()

	m := make(map[uint64]struct{}, t.N)
	for _, v := range s {
		m[v] = struct{}{}
	}
}

var ok bool

func BenchmarkCostLookup(t *testing.B) {
	s := make([]uint64, t.N)
	for i := range s {
		s[i] = uint64(prand.Int63())
	}

	m := make(map[uint64]struct{}, t.N)
	for _, v := range s {
		m[v] = struct{}{}
	}

	t.ReportAllocs()
	t.ResetTimer()

	for _, v := range s {
		_, ok = m[v]
	}
	_ = ok
}

func BenchmarkCostComp(t *testing.B) {
	s1 := make([]uint64, t.N)
	s2 := make([]uint64, t.N)
	for i := range s1 {
		s1[i] = uint64(prand.Int63())
		s2[i] = uint64(prand.Int63())
	}

	t.ReportAllocs()
	t.ResetTimer()

	var idx1, idx2 int
	for {
		switch {
		case idx1 == len(s1):
			return
		case idx2 == len(s2):
			return
		case s1[idx1] < s2[idx2]:
			idx1++
		default:
			idx2++
		}
	}
}

var i uint64

func BenchmarkCostKey(t *testing.B) {
	data := make([][]byte, t.N)
	for i := range data {
		data[i] = make([]byte, 24)
	}

	modulusNP := 10000 * uint64(784931)
	key := [KeySize]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16,
	}

	t.ReportAllocs()
	t.ResetTimer()

	nphi := modulusNP >> 32
	nplo := uint64(uint32(modulusNP))
	for _, d := range data {
		v := siphash.Sum64(d, &key)
		i = fastReduction(v, nphi, nplo)
	}
	_ = i
}

func BenchmarkCostRead(t *testing.B) {
	data := make([][]byte, t.N)
	for i := range data {
		data[i] = make([]byte, 24)
	}

	key := [KeySize]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16,
	}

	filter, _ := BuildGCSFilter(19, 784931, key, data)

	filterData, _ := filter.Bytes()

	b := bstream.NewBStreamReader(filterData)

	t.ReportAllocs()
	t.ResetTimer()

	var value uint64
	for {
		delta, err := filter.readFullUint64(b)
		if err != nil {
			return
		}
		value += delta
	}
}
