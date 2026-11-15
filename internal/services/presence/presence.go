package presence

import (
	"errors"
	"sync"
)

type ActiveUsers struct {
	mu    sync.Mutex
	items []int
	limit int
}

func New(limit int) (*ActiveUsers, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be a positive integer")
	}
	return &ActiveUsers{limit: limit}, nil
}

func (a *ActiveUsers) Add(id int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, item := range a.items {
		if item == id {
			a.items = append(a.items[:i], a.items[i+1:]...)
			break
		}
	}

	a.items = append(a.items, id)
	if len(a.items) > a.limit {
		a.items = a.items[len(a.items)-a.limit:]
	}
}

func (a *ActiveUsers) Snapshot() []int {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]int, len(a.items))
	copy(result, a.items)
	return result
}
