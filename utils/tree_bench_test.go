package utils

import (
	"sync"
	"testing"
)

// Benchmark single-threaded GetNode
func BenchmarkTreeGetNodeSingle(b *testing.B) {
	tree := NewTree("root")
	tree.AddNode("child1")
	tree.AddNode("child2")
	tree.AddNode("child3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GetNode("child1")
		tree.GetNode("child2")
		tree.GetNode("child3")
	}
}

// Benchmark concurrent GetNode (demonstrates RWMutex benefit)
func BenchmarkTreeGetNodeConcurrent(b *testing.B) {
	tree := NewTree("root")
	tree.AddNode("child1")
	tree.AddNode("child2")
	tree.AddNode("child3")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tree.GetNode("child1")
			tree.GetNode("child2")
			tree.GetNode("child3")
		}
	})
}

// Benchmark single-threaded AddNode
func BenchmarkTreeAddNodeSingle(b *testing.B) {
	tree := NewTree("root")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.AddNode("node")
		tree.DeleteNode("node")
	}
}

// Benchmark Variables method
func BenchmarkTreeVariablesSingle(b *testing.B) {
	tree := NewTree("root")
	tree.AddNode("child1")
	tree.AddNode("child2")
	tree.AddNode("child3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Variables()
	}
}

// Benchmark concurrent Variables
func BenchmarkTreeVariablesConcurrent(b *testing.B) {
	tree := NewTree("root")
	tree.AddNode("child1")
	tree.AddNode("child2")
	tree.AddNode("child3")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tree.Variables()
		}
	})
}

// Benchmark mixed read/write workload
func BenchmarkTreeMixedWorkload(b *testing.B) {
	tree := NewTree("root")
	tree.AddNode("child1")
	tree.AddNode("child2")

	b.ResetTimer()
	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine (10% of operations)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N/10; i++ {
			tree.AddNode("temp")
			tree.DeleteNode("temp")
		}
	}()

	// Reader goroutine (90% of operations)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N*9/10; i++ {
			tree.GetNode("child1")
			tree.GetNode("child2")
		}
	}()

	wg.Wait()
}
