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
func (self *Tree) AddNode(name string) *Tree {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Check if node already exists (must be inside lock to avoid TOCTOU)
	first := findNodeByName(self.Downs, name)
	if first != nil {
		return first
	}

	child := NewTree(name)
	self.Downs = append(self.Downs, child)
	child.Up = self
	return child
}

// AddNodes adds a new node to the tree, and then recursively adds the given names
// as children to the new node, creating a path.
// Returns the leaf node at the end of the path.
func (self *Tree) AddNodes(tag string, names ...string) *Tree {
	node := self.AddNode(tag)
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
func (self *Tree) GetNode(tag string, names ...string) *Tree {
	if tag == "" {
		return self
	}

	self.mu.RLock()
	down := findNodeByName(self.Downs, tag)
	self.mu.RUnlock()

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
func (self *Tree) DeleteNode(name string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for i, item := range self.Downs {
		if item.Name == name {
			if i+1 == len(self.Downs) {
				self.Downs = self.Downs[:i]
			} else {
				self.Downs = append(self.Downs[:i], self.Downs[i+1:]...)
			}
			return
		}
	}
}

// AddItem adds or updates a key-value pair in the tree's data storage.
// The value can be any type and is stored as interface{}.
// Thread-safe: Uses sync.Map which provides lock-free concurrent access.
func (self *Tree) AddItem(k string, v interface{}) {
	self.Data.Store(k, v)
}

// DeleteItem removes a key-value pair from the tree's data storage.
// Thread-safe: Uses sync.Map which provides lock-free concurrent access.
func (self *Tree) DeleteItem(k string) {
	self.Data.Delete(k)
}

// FindNode searches for a node by following a path of names through the tree.
// Returns nil if any part of the path is not found.
// Example: FindNode([]string{"service", "http"}) finds the http node under service.
// Thread-safe: Uses read locks to safely traverse the tree.
func (self *Tree) FindNode(names []string) *Tree {
	if names == nil || len(names) == 0 {
		return nil
	}

	// Lock and copy children to avoid holding lock during recursion
	self.mu.RLock()
	downs := make([]*Tree, len(self.Downs))
	copy(downs, self.Downs)
	self.mu.RUnlock()

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
func (self *Tree) Variables() map[string]interface{} {
	hash := make(map[string]interface{})

	// Lock and copy children to avoid holding lock during recursion
	self.mu.RLock()
	downs := make([]*Tree, len(self.Downs))
	copy(downs, self.Downs)
	name := self.Name
	self.mu.RUnlock()

	for _, down := range downs {
		if variables := down.Variables(); variables != nil {
			hash[down.Name] = variables
		}
	}

	// Data.Range is already thread-safe (sync.Map)
	self.Data.Range(func(k, v any) bool {
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
