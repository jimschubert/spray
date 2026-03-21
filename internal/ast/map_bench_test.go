package ast

import (
	"testing"
)

func BenchmarkOrderedTypeMap_Set(b *testing.B) {
	m := &OrderedTypeMap{}
	value := &StringLiteral{Value: "value"}
	pos := Position{Line: 1, Col: 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set("key", value, pos)
	}
}

func BenchmarkOrderedTypeMap_Get(b *testing.B) {
	m := &OrderedTypeMap{}
	m.Set("key", &StringLiteral{Value: "value"}, Position{Line: 1, Col: 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get("key")
	}
}

func BenchmarkOrderedTypeMap_PositionOf(b *testing.B) {
	m := &OrderedTypeMap{}
	m.Set("key", &StringLiteral{Value: "value"}, Position{Line: 1, Col: 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.PositionOf("key")
	}
}

func BenchmarkOrderedTypeMap_SetAndGet(b *testing.B) {
	m := &OrderedTypeMap{}
	value := &StringLiteral{Value: "value"}
	pos := Position{Line: 1, Col: 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set("key", value, pos)
		m.Get("key")
		m.PositionOf("key")
	}
}

func BenchmarkOrderedTypeMap_AllWithPositions(b *testing.B) {
	m := &OrderedTypeMap{}
	for i := 0; i < 10; i++ {
		m.Set("key", &StringLiteral{Value: "value"}, Position{Line: i, Col: 1})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.AllWithPositions()(func(key string, val TypeNode, pos Position) bool {
			return true
		})
	}
}

func BenchmarkOrderedTypeMap_Delete(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := &OrderedTypeMap{}
		m.Set("key", &StringLiteral{Value: "value"}, Position{Line: 1, Col: 1})
		m.Delete("key")
	}
}
