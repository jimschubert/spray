package ast

import (
	"iter"

	"rsc.io/omap"
)

// Seq3 is a sequence of 3 values
type Seq3[A, B, C any] func(yield func(A, B, C) bool)

// OrderedTypeMap is an ordered map that associates TypeNode values with string keys.
// The key Position is also tracked to allow for position-based features associated with the key.
type OrderedTypeMap struct {
	values    omap.Map[string, TypeNode]
	positions omap.Map[string, Position]
}

// Set upserts the TypeNode at the given key, tracking its Position.
func (m *OrderedTypeMap) Set(key string, value TypeNode, pos Position) {
	m.values.Set(key, value)
	m.positions.Set(key, pos)
}

// Get retrieves the TypeNode for the givern key.
func (m *OrderedTypeMap) Get(key string) (TypeNode, bool) {
	return m.values.Get(key)
}

// PositionOf returns the Position associated with the given key.
// If key is not in the map, it returns a zero-valued Position.
func (m *OrderedTypeMap) PositionOf(key string) Position {
	pos, _ := m.positions.Get(key)
	return pos
}

func (m *OrderedTypeMap) Delete(key string) {
	m.values.Delete(key)
	m.positions.Delete(key)
}

func (m *OrderedTypeMap) All() iter.Seq2[string, TypeNode] {
	return m.values.All()
}

// AllWithPositions iterates a (key, value, position) tuple.
func (m *OrderedTypeMap) AllWithPositions() Seq3[string, TypeNode, Position] {
	return func(yield func(string, TypeNode, Position) bool) {
		for key, value := range m.values.All() {
			pos := m.PositionOf(key)
			if !yield(key, value, pos) {
				return
			}
		}
	}
}

func (m *OrderedTypeMap) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for key := range m.values.All() {
			if !yield(key) {
				return
			}
		}
	}
}

func (m *OrderedTypeMap) Values() iter.Seq[TypeNode] {
	return func(yield func(TypeNode) bool) {
		for _, value := range m.values.All() {
			if !yield(value) {
				return
			}
		}
	}
}

func (m *OrderedTypeMap) Len() int {
	n := 0
	for range m.values.All() {
		n++
	}
	return n
}
