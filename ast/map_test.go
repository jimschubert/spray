package ast

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestOrderedTypeMap_SetAndGet(t *testing.T) {
	t.Parallel()
	m := &OrderedTypeMap{}

	value := &StringLiteral{Value: "value"}
	pos := Position{Line: 1, Col: 5}

	m.Set("key1", value, pos)

	retrieved, ok := m.Get("key1")
	assert.True(t, ok)
	assert.NotZero(t, retrieved)

	lit, ok := retrieved.(*StringLiteral)
	assert.True(t, ok)
	assert.Equal(t, "value", lit.Value)

	retrievedPos := m.PositionOf("key1")
	assert.Equal(t, pos, retrievedPos)
}

func TestOrderedTypeMap_InsertionOrder(t *testing.T) {
	t.Parallel()
	m := &OrderedTypeMap{}

	m.Set("first", &StringLiteral{Value: "a"}, Position{Line: 1, Col: 1})
	m.Set("second", &StringLiteral{Value: "b"}, Position{Line: 2, Col: 1})
	m.Set("third", &StringLiteral{Value: "c"}, Position{Line: 3, Col: 1})

	var keys []string
	for key := range m.Keys() {
		keys = append(keys, key)
	}

	assert.Equal(t, []string{"first", "second", "third"}, keys)
}

func TestOrderedTypeMap_AllWithPositions(t *testing.T) {
	t.Parallel()
	m := &OrderedTypeMap{}

	m.Set("a", &StringLiteral{Value: "value_a"}, Position{Line: 1, Col: 1})
	m.Set("b", &StringLiteral{Value: "value_b"}, Position{Line: 2, Col: 5})

	type entry struct {
		key string
		val *StringLiteral
		pos Position
	}

	var entries []entry
	m.AllWithPositions()(func(key string, val TypeNode, pos Position) bool {
		lit := val.(*StringLiteral)
		entries = append(entries, entry{key, lit, pos})
		return true // continue iteration
	})

	assert.Equal(t, 2, len(entries))
	assert.Equal(t, "a", entries[0].key)
	assert.Equal(t, Position{Line: 1, Col: 1}, entries[0].pos)
	assert.Equal(t, "b", entries[1].key)
	assert.Equal(t, Position{Line: 2, Col: 5}, entries[1].pos)
}

func TestOrderedTypeMap_Delete(t *testing.T) {
	t.Parallel()
	m := &OrderedTypeMap{}

	m.Set("key", &StringLiteral{Value: "value"}, Position{Line: 1, Col: 1})
	m.Delete("key")

	_, ok := m.Get("key")
	assert.False(t, ok)

	pos := m.PositionOf("key")
	assert.Equal(t, Position{}, pos, "Retrieving non-existent key should return zero-value Position")
}
