// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Provides functions for sorting tx inputs and outputs according to BIP 69
// (https://github.com/bitcoin/bips/blob/master/bip-0069.mediawiki)

package txsort

import (
	"bytes"
	"sort"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// InPlaceSort modifies the passed transaction inputs and outputs to be sorted
// based on BIP 69.
//
// WARNING: This function must NOT be called with published transactions since
// it will mutate the transaction if it's not already sorted.  This can cause
// issues if you mutate a tx in a block, for example, which would invalidate the
// block.  It could also cause cached hashes, such as in a btcutil.Tx to become
// invalidated.
//
// The function should only be used if the caller is creating the transaction or
// is otherwise 100% positive mutating will not cause adverse affects due to
// other dependencies.
func InPlaceSort(tx *wire.MsgTx) {
	sort.Sort(sortableInputSlice(tx.TxIn))
	sort.Sort(sortableOutputSlice(tx.TxOut))
}

// Sort returns a new transaction with the inputs and outputs sorted based on
// BIP 69.  The passed transaction is not modified and the new transaction
// might have a different hash if any sorting was done.
func Sort(tx *wire.MsgTx) *wire.MsgTx {
	txCopy := tx.Copy()
	sort.Sort(sortableInputSlice(txCopy.TxIn))
	sort.Sort(sortableOutputSlice(txCopy.TxOut))
	return txCopy
}

// IsSorted checks whether tx has inputs and outputs sorted according to BIP
// 69.
func IsSorted(tx *wire.MsgTx) bool {
	if !sort.IsSorted(sortableInputSlice(tx.TxIn)) {
		return false
	}
	if !sort.IsSorted(sortableOutputSlice(tx.TxOut)) {
		return false
	}
	return true
}

type sortableInputSlice []*wire.TxIn
type sortableOutputSlice []*wire.TxOut

// For SortableInputSlice and SortableOutputSlice, three functions are needed
// to make it sortable with sort.Sort() -- Len, Less, and Swap
// Len and Swap are trivial.  Less is BIP 69 specific.
func (s sortableInputSlice) Len() int       { return len(s) }
func (s sortableOutputSlice) Len() int      { return len(s) }
func (s sortableOutputSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableInputSlice) Swap(i, j int)  { s[i], s[j] = s[j], s[i] }

// Input comparison function.
// First sort based on input hash (reversed / rpc-style), then index.
func (s sortableInputSlice) Less(i, j int) bool {
	// Input hashes are the same, so compare the index.
	ihash := s[i].PreviousOutPoint.Hash
	jhash := s[j].PreviousOutPoint.Hash
	if ihash == jhash {
		return s[i].PreviousOutPoint.Index < s[j].PreviousOutPoint.Index
	}

	// At this point, the hashes are not equal, so reverse them to
	// big-endian and return the result of the comparison.
	const hashSize = chainhash.HashSize
	for b := 0; b < hashSize/2; b++ {
		ihash[b], ihash[hashSize-1-b] = ihash[hashSize-1-b], ihash[b]
		jhash[b], jhash[hashSize-1-b] = jhash[hashSize-1-b], jhash[b]
	}
	return bytes.Compare(ihash[:], jhash[:]) == -1
}

// Output comparison function.
// First sort based on amount (smallest first), then PkScript.
func (s sortableOutputSlice) Less(i, j int) bool {
	if s[i].Value == s[j].Value {
		return bytes.Compare(s[i].PkScript, s[j].PkScript) < 0
	}
	return s[i].Value < s[j].Value
}

// InPlaceContextualSort modifies the passed transaction inputs and outputs to
// be sorted based on a _contextual_ BIP 69. Sorting occurs as defined in BIP
// 69, and uses a contextual byte slice as a tie breaker if two outputs are
// otherwise identical in value and pubkey script.
//
// WARNING: This function must NOT be called with published transactions since
// it will mutate the transaction if it's not already sorted.  This can cause
// issues if you mutate a tx in a block, for example, which would invalidate the
// block.  It could also cause cached hashes, such as in a btcutil.Tx to become
// invalidated.
//
// The function should only be used if the caller is creating the transaction or
// is otherwise 100% positive mutating will not cause adverse affects due to
// other dependencies.
func InPlaceContextualSort(tx *wire.MsgTx, ctxts [][]byte) {
	sort.Sort(sortableInputSlice(tx.TxIn))
	sort.Sort(makeSortableOutputContextSlice(tx.TxOut, ctxts))
}

// ContextualSort returns a new transaction with the inputs and outputs sorted
// based on _contextual_ BIP 69. Sorting occurs as defined in BIP 69, and uses
// a contextual byte slice as a tie breaker if two outputs are otherwise
// identical in value and pubkey script. The passed transaction is not modified
// and the new transaction might have a different hash if any sorting was done.
func ContextualSort(tx *wire.MsgTx, ctxts [][]byte) *wire.MsgTx {
	txCopy := tx.Copy()
	sort.Sort(sortableInputSlice(txCopy.TxIn))
	sort.Sort(makeSortableOutputContextSlice(tx.TxOut, ctxts))
	return txCopy
}

// IsContextualSorted checks whether tx has inputs and outputs sorted according
// a _contextual_ BIP 69 sort.
func IsContextualSorted(tx *wire.MsgTx, ctxts [][]byte) bool {
	if !sort.IsSorted(sortableInputSlice(tx.TxIn)) {
		return false
	}
	if !sort.IsSorted(makeSortableOutputContextSlice(tx.TxOut, ctxts)) {
		return false
	}
	return true
}

type txOutWithContext struct {
	*wire.TxOut
	Context []byte
}

// makeSortableOutputContextSlice zips a slice of txouts with it's contexts so
// that the outputs can be sorted using a contextual BIP 69 sort.
func makeSortableOutputContextSlice(
	txouts []*wire.TxOut, ctxts [][]byte) sortableOutputContextSlice {

	if len(txouts) != len(ctxts) {
		panic("length of txouts and contexts must be equal")
	}

	// Create a txOutWithContext that marries each txout to it's contextual
	// byte slice.
	contextualOutputs := make(sortableOutputContextSlice, len(txouts))
	for i, txout := range txouts {
		contextualOutputs[i] = txOutWithContext{
			TxOut:   txout,
			Context: ctxts[i],
		}
	}

	return contextualOutputs
}

type sortableOutputContextSlice []txOutWithContext

// Implement the sort.Interface.
func (s sortableOutputContextSlice) Len() int      { return len(s) }
func (s sortableOutputContextSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableOutputContextSlice) Less(i, j int) bool {
	// If the values are distinct, return value_i < value_j.
	if s[i].Value != s[j].Value {
		return s[i].Value < s[j].Value
	}

	// If the pk scripts are distinct, return pkscript_i < pkscript_j.
	pkScriptCmp := bytes.Compare(s[i].PkScript, s[j].PkScript)
	if pkScriptCmp != 0 {
		return pkScriptCmp == -1
	}

	// Otherwise, compare the contexts to determine if i < j.
	return bytes.Compare(s[i].Context, s[j].Context) == -1
}
