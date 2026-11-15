package presence

import (
	"sync"
	"testing"
)

func TestNewRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		if _, err := New(limit); err == nil {
			t.Fatalf("New(%d) returned nil error", limit)
		}
	}
}

func TestAddKeepsOnlyLastItems(t *testing.T) {
	active, err := New(2)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	active.Add(1)
	active.Add(2)
	active.Add(3)

	got := active.Snapshot()
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("Snapshot() = %v, want [2 3]", got)
	}
}

func TestAddDeduplicates(t *testing.T) {
	active, err := New(3)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	active.Add(1)
	active.Add(2)
	active.Add(1)

	if got, want := active.Snapshot(), []int{2, 1}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Snapshot() = %v, want %v", got, want)
	}

	active.Add(2)

	if got, want := active.Snapshot(), []int{1, 2}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Snapshot() = %v, want %v", got, want)
	}
}

func TestSnapshotReturnsCopy(t *testing.T) {
	active, err := New(3)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	active.Add(1)

	got := active.Snapshot()
	got[0] = 99

	again := active.Snapshot()
	if again[0] != 1 {
		t.Fatalf("Snapshot() shares underlying slice: got %v", again)
	}
}

func TestConcurrentAdd(t *testing.T) {
	const goroutines = 100

	active, err := New(goroutines)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			active.Add(id)
		}(i + 1)
	}
	wg.Wait()

	got := active.Snapshot()
	if len(got) != goroutines {
		t.Fatalf("len(Snapshot()) = %d, want %d", len(got), goroutines)
	}

	seen := make(map[int]struct{}, goroutines)
	for _, id := range got {
		seen[id] = struct{}{}
	}
	if len(seen) != goroutines {
		t.Fatalf("Snapshot() contains duplicates: %v", got)
	}
}
