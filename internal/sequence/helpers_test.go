package sequence

import (
	"iter"
	"slices"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestGetFirstValue(t *testing.T) {
	tests := []struct {
		name      string
		seq       iter.Seq[string]
		want      string
		wantError bool
	}{
		{
			name:      "empty sequence means error",
			seq:       slices.Values([]string{}),
			wantError: true,
		},
		{
			name: "single element",
			seq:  slices.Values([]string{"alpha"}),
			want: "alpha",
		},
		{
			name: "multiple elements returns first",
			seq:  slices.Values([]string{"first", "second", "third"}),
			want: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFirstValue(tt.seq)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetFirstValue_int(t *testing.T) {
	tests := []struct {
		name      string
		seq       iter.Seq[int]
		want      int
		wantError bool
	}{
		{
			name:      "empty sequence returns error",
			seq:       slices.Values([]int{}),
			wantError: true,
		},
		{
			name: "returns first int",
			seq:  slices.Values([]int{42, 99, 7}),
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFirstValue(tt.seq)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetFirstTuple(t *testing.T) {
	// helper to create iter.Seq2 from a slice of pairs
	type pair struct {
		key   string
		value int
	}
	pairsToSeq2 := func(pairs []pair) iter.Seq2[string, int] {
		return func(yield func(string, int) bool) {
			for _, p := range pairs {
				if !yield(p.key, p.value) {
					return
				}
			}
		}
	}

	tests := []struct {
		name      string
		seq       iter.Seq2[string, int]
		wantKey   string
		wantValue int
		wantError bool
	}{
		{
			name:      "empty sequence returns error",
			seq:       pairsToSeq2(nil),
			wantError: true,
		},
		{
			name:      "single pair",
			seq:       pairsToSeq2([]pair{{key: "a", value: 1}}),
			wantKey:   "a",
			wantValue: 1,
		},
		{
			name: "multiple pairs returns first",
			seq: pairsToSeq2([]pair{
				{key: "first", value: 10},
				{key: "second", value: 20},
			}),
			wantKey:   "first",
			wantValue: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, err := GetFirstTuple(tt.seq)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantKey, key)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}

func TestFindFirst(t *testing.T) {
	tests := []struct {
		name      string
		seq       iter.Seq[int]
		predicate func(int) bool
		want      int
		wantError bool
	}{
		{
			name:      "empty sequence means error",
			seq:       slices.Values([]int{}),
			predicate: func(i int) bool { return i > 0 },
			wantError: true,
		},
		{
			name:      "no match means error",
			seq:       slices.Values([]int{1, 2, 3}),
			predicate: func(i int) bool { return i > 100 },
			wantError: true,
		},
		{
			name:      "match start",
			seq:       slices.Values([]int{10, 20, 30}),
			predicate: func(i int) bool { return i >= 10 },
			want:      10,
		},
		{
			name:      "match middle",
			seq:       slices.Values([]int{1, 2, 42, 3}),
			predicate: func(i int) bool { return i == 42 },
			want:      42,
		},
		{
			name:      "match end",
			seq:       slices.Values([]int{1, 2, 3}),
			predicate: func(i int) bool { return i == 3 },
			want:      3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindFirst(tt.seq, tt.predicate)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFindFirst2(t *testing.T) {
	type pair struct {
		key   string
		value int
	}
	pairsToSeq2 := func(pairs []pair) iter.Seq2[string, int] {
		return func(yield func(string, int) bool) {
			for _, p := range pairs {
				if !yield(p.key, p.value) {
					return
				}
			}
		}
	}

	tests := []struct {
		name      string
		seq       iter.Seq2[string, int]
		predicate func(string, int) bool
		wantKey   string
		wantValue int
		wantError bool
	}{
		{
			name:      "empty sequence means error",
			seq:       pairsToSeq2(nil),
			predicate: func(k string, v int) bool { return true },
			wantError: true,
		},
		{
			name: "no match means error",
			seq: pairsToSeq2([]pair{
				{key: "a", value: 1},
				{key: "b", value: 2},
			}),
			predicate: func(k string, v int) bool { return k == "z" },
			wantError: true,
		},
		{
			name: "match key",
			seq: pairsToSeq2([]pair{
				{key: "a", value: 1},
				{key: "b", value: 2},
				{key: "c", value: 3},
			}),
			predicate: func(k string, v int) bool { return k == "b" },
			wantKey:   "b",
			wantValue: 2,
		},
		{
			name: "match value",
			seq: pairsToSeq2([]pair{
				{key: "x", value: 10},
				{key: "y", value: 42},
			}),
			predicate: func(k string, v int) bool { return v == 42 },
			wantKey:   "y",
			wantValue: 42,
		},
		{
			name: "returns first match found",
			seq: pairsToSeq2([]pair{
				{key: "a", value: 1},
				{key: "b", value: 2},
				{key: "c", value: 3},
			}),
			predicate: func(k string, v int) bool { return v > 0 },
			wantKey:   "a",
			wantValue: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, err := FindFirst2(tt.seq, tt.predicate)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantKey, key)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}
