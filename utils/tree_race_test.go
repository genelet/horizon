package utils

import (
	"sync"
	"testing"
)

// TestTreeConcurrentAddNode demonstrates the race condition in AddNode
func TestTreeConcurrentAddNode(t *testing.T) {
	tree := NewTree("root")

	const numGoroutines = 10
	const nodeName = "shared"

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines trying to add the same node
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			tree.AddNode(nodeName)
		}()
	}

	wg.Wait()

	// Should only have 1 child node named "shared", but due to race condition,
	// might have duplicates
	count := 0
	for _, down := range tree.Downs {
		if down.Name == nodeName {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 node named %q, got %d (race condition detected)", nodeName, count)
	}
}

// TestTreeConcurrentReadWrite demonstrates race between AddNode and GetNode
func TestTreeConcurrentReadWrite(t *testing.T) {
	tree := NewTree("root")

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tree.AddNode("node")
		}
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tree.GetNode("node")
		}
	}()

	wg.Wait()
}

// TestTreeConcurrentFindNode demonstrates race in FindNode
func TestTreeConcurrentFindNode(t *testing.T) {
	tree := NewTree("root")
	tree.AddNode("child1")
	tree.AddNode("child2")

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tree.AddNode("child3")
			tree.DeleteNode("child3")
		}
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tree.FindNode([]string{"child1"})
			tree.FindNode([]string{"child2"})
		}
	}()

	wg.Wait()
}

// TestTreeConcurrentVariables demonstrates race in Variables
func TestTreeConcurrentVariables(t *testing.T) {
	tree := NewTree("root")

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tree.AddNode("node")
		}
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tree.Variables()
		}
	}()

	wg.Wait()
}
