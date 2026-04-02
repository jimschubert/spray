package sequence

import (
	"fmt"
	"iter"
)

// GetFirstValue returns the first value from the given sequence, or an error if empty.
func GetFirstValue[T any](seq iter.Seq[T]) (T, error) {
	pull, stop := iter.Pull(seq)
	defer stop()

	firstValue, ok := pull()
	if !ok {
		return firstValue, fmt.Errorf("sequence is empty")
	}
	return firstValue, nil
}

// GetFirstTuple returns the first key-value pair from the given sequence, or an error if empty.
func GetFirstTuple[K, V any](seq iter.Seq2[K, V]) (K, V, error) {
	pull, stop := iter.Pull2(seq)
	defer stop()

	firstValue1, firstValue2, ok := pull()
	if !ok {
		var key K
		var value V
		return key, value, fmt.Errorf("sequence is empty")
	}
	return firstValue1, firstValue2, nil
}

// FindFirst returns the first element in the sequence that satisfies the given predicate, or an error if it isn't found.
func FindFirst[T any](seq iter.Seq[T], predicate func(T) bool) (T, error) {
	pull, stop := iter.Pull(seq)
	defer stop()

	for {
		value, ok := pull()
		if !ok {
			var zeroValue T
			return zeroValue, fmt.Errorf("no matching element found")
		}
		if predicate(value) {
			return value, nil
		}
	}
}

// FindFirst2 returns the first key-value pair in the sequence that satisfies the given predicate, or an error if it isn't found.
func FindFirst2[K comparable, V any](seq iter.Seq2[K, V], predicate func(K, V) bool) (K, V, error) {
	pull, stop := iter.Pull2(seq)
	defer stop()

	for {
		key, value, ok := pull()
		if !ok {
			var k K
			var v V
			return k, v, fmt.Errorf("no matching element found")
		}
		if predicate(key, value) {
			return key, value, nil
		}
	}
}
