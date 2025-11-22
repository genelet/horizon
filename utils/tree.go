package utils

import (
	"sync"
)

// Use constants from constants.go
const (
	// Deprecated: Use TreeNodeVar instead
	VAR = TreeNodeVar
	// Deprecated: Use ContextKeyAttributes instead
	ATTRIBUTES = ContextKeyAttributes
	// Deprecated: Use ContextKeyFunctions instead
	FUNCTIONS = ContextKeyFunctions
)

// Tree is a tree structure that can be used to represent a nested structure in HCL.
//
// Tree is thread-safe for concurrent access. All methods use appropriate locking
// to protect shared state. The Data field uses sync.Map for lock-free concurrent access.
type Tree struct {
	mu    sync.RWMutex
	Name  string
	Data  sync.Map
	Up    *Tree
	Downs []*Tree
}

// NewTree creates a new tree with the given name
func NewTree(name string) *Tree {
	return &Tree{Name: name, Data: sync.Map{}}
}

// AddNode adds a new node to the tree. It returns the newly created node.
// If a node with the same name already exists, returns the existing node.
// Thread-safe: Uses write lock to protect against concurrent modifications.
func (t *Tree) AddNode(name string) *Tree {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if node already exists (must be inside lock to avoid TOCTOU)
	first := findNodeByName(t.Downs, name)
	if first != nil {
		return first
	}

	child := NewTree(name)
	t.Downs = append(t.Downs, child)
	child.Up = t
	return child
}

// AddNodes adds a new node to the tree, and then recursively adds the given names
// as children to the new node, creating a path.
// Returns the leaf node at the end of the path.
func (t *Tree) AddNodes(tag string, names ...string) *Tree {
	node := t.AddNode(tag)
	for _, name := range names {
		node = node.AddNode(name)
	}
	return node
}

// findNodeByName searches for a child node with the given name in the slice.
// Returns the node if found, nil otherwise.
func findNodeByName(downs []*Tree, name string) *Tree {
	for _, down := range downs {
		if down.Name == name {
			return down
		}
	}
	return nil
}

// GetNode returns the node at the specified path (tag + names).
// If tag is empty, returns self. If the node does not exist, returns nil.
// Example: GetNode("service", "http", "web") returns the node at path service/http/web
// Thread-safe: Uses read locks to safely traverse the tree.
func (t *Tree) GetNode(tag string, names ...string) *Tree {
	if tag == "" {
		return t
	}

	t.mu.RLock()
	down := findNodeByName(t.Downs, tag)
	t.mu.RUnlock()

	if down == nil {
		return nil
	}

	for _, name := range names {
		down.mu.RLock()
		next := findNodeByName(down.Downs, name)
		down.mu.RUnlock()

		if next == nil {
			return nil
		}
		down = next
	}

	return down
}

// DeleteNode deletes the node with the given name.
// Thread-safe: Uses write lock to protect against concurrent modifications.
func (t *Tree) DeleteNode(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, item := range t.Downs {
		if item.Name == name {
			if i+1 == len(t.Downs) {
				t.Downs = t.Downs[:i]
			} else {
				t.Downs = append(t.Downs[:i], t.Downs[i+1:]...)
			}
			return
		}
	}
}

// AddItem adds or updates a key-value pair in the tree's data storage.
// The value can be any type and is stored as any.
// Thread-safe: Uses sync.Map which provides lock-free concurrent access.
func (t *Tree) AddItem(k string, v any) {
	t.Data.Store(k, v)
}

// DeleteItem removes a key-value pair from the tree's data storage.
// Thread-safe: Uses sync.Map which provides lock-free concurrent access.
func (t *Tree) DeleteItem(k string) {
	t.Data.Delete(k)
}

// FindNode searches for a node by following a path of names through the tree.
// Returns nil if any part of the path is not found.
// Example: FindNode([]string{"service", "http"}) finds the http node under service.
// Thread-safe: Uses read locks to safely traverse the tree.
func (t *Tree) FindNode(names []string) *Tree {
	if len(names) == 0 {
		return nil
	}

	// Lock and copy children to avoid holding lock during recursion
	t.mu.RLock()
	downs := make([]*Tree, len(t.Downs))
	copy(downs, t.Downs)
	t.mu.RUnlock()

	var down *Tree
	for _, item := range downs {
		if item.Name == names[0] {
			if len(names) == 1 {
				return item
			}
			return item.FindNode(names[1:])
		} else {
			down = item.FindNode(names)
			if down != nil {
				return down
			}
		}
	}

	return nil
}

// Variables returns all variables in the tree as a generic map.
// For HCL expression evaluation, use CtyVariables instead.
// Thread-safe: Uses read locks and copies children before recursive calls.
func (t *Tree) Variables() map[string]any {
	hash := make(map[string]any)

	// Lock and copy children to avoid holding lock during recursion
	t.mu.RLock()
	downs := make([]*Tree, len(t.Downs))
	copy(downs, t.Downs)
	name := t.Name
	t.mu.RUnlock()

	for _, down := range downs {
		if variables := down.Variables(); variables != nil {
			hash[down.Name] = variables
		}
	}

	// Data.Range is already thread-safe (sync.Map)
	t.Data.Range(func(k, v any) bool {
		if k != VAR {
			hash[k.(string)] = v
		}
		return true
	})

	if name == VAR {
		hash[VAR] = hash
	}

	return hash
}
