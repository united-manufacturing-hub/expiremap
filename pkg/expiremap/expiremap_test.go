package expiremap

import (
	"math/rand"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New[int, string]()
	if m == nil {
		t.Fatal("New returned nil")
	}
}

func TestSetAndGet(t *testing.T) {
	m := New[int, string]()
	m.Set(1, "one")

	v, ok := m.Get(1)
	if !ok || *v != "one" {
		t.Fatalf("expected %v, got %v", "one", v)
	}
}

func TestSetExAndGet(t *testing.T) {
	m := New[int, string]()
	m.SetEx(1, "one", time.Second*2)

	v, ok := m.Get(1)
	if !ok || *v != "one" {
		t.Fatalf("expected %v, got %v", "one", v)
	}

	time.Sleep(time.Second * 3)

	_, ok = m.Get(1)
	if ok {
		t.Fatal("Expected value to be expired")
	}
}

func TestLoadAndDelete(t *testing.T) {
	m := New[int, string]()
	m.Set(1, "one")

	v, ok := m.LoadAndDelete(1)
	if !ok || *v != "one" {
		t.Fatalf("expected %v, got %v", "one", v)
	}

	_, ok = m.Get(1)
	if ok {
		t.Fatal("Expected key to be deleted")
	}
}

func TestLoad(t *testing.T) {
	m := New[int, string]()
	m.Set(1, "one")

	v, ok := m.Load(1)
	if !ok || *v != "one" {
		t.Fatalf("expected %v, got %v", "one", v)
	}
}

func TestDelete(t *testing.T) {
	m := New[int, string]()
	m.Set(1, "one")
	m.Delete(1)

	_, ok := m.Get(1)
	if ok {
		t.Fatal("Expected key to be deleted")
	}
}

// This is a simple test to check cull function.
// However, for complex scenarios, you might want to simulate time or use a mock clock.
func TestCull(t *testing.T) {
	m := NewEx[int, string](time.Second, time.Second)
	m.Set(1, "one")

	time.Sleep(time.Second * 2)

	_, ok := m.Get(1)
	if ok {
		t.Fatal("Expected value to be expired")
	}
}

func TestConcurrency(t *testing.T) {
	m := New[int, string]()
	const numRoutines = 1000
	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < numRoutines; i++ {
		go func(i int) {
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			m.Set(i, "value")
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numRoutines; i++ {
		go func(i int) {
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			m.Get(i)
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numRoutines*2; i++ {
		<-done
	}

	// Check the final state of the map
	for i := 0; i < numRoutines; i++ {
		value, ok := m.Get(i)
		if !ok || *value != "value" {
			t.Fatalf("Expected key %v to be 'value', got %v", i, value)
		}
	}
}

func TestLoadAndDeleteNoKey(t *testing.T) {
	m := New[int, string]()
	_, ok := m.LoadAndDelete(1)
	if ok {
		t.Fatal("Expected ok to be false")
	}
}

func TestLoadAndDeleteExpiredKey(t *testing.T) {
	m := NewEx[int, string](time.Minute, time.Millisecond)
	m.Set(1, "one")
	time.Sleep(time.Millisecond * 2)
	_, ok := m.LoadAndDelete(1)
	if ok {
		t.Fatal("Expected ok to be false")
	}
}

func TestLoadNoKey(t *testing.T) {
	m := New[int, string]()
	_, ok := m.Load(1)
	if ok {
		t.Fatal("Expected ok to be false")
	}
}

func TestLoadExpiredKey(t *testing.T) {
	m := NewEx[int, string](time.Minute, time.Millisecond)
	m.Set(1, "one")
	time.Sleep(time.Millisecond * 2)
	_, ok := m.Load(1)
	if ok {
		t.Fatal("Expected ok to be false")
	}
}

func TestNoCull(t *testing.T) {
	m := NewEx[int, string](time.Millisecond, time.Hour)
	m.Set(1, "one")
	time.Sleep(time.Millisecond * 2)
	_, ok := m.Get(1)
	if !ok {
		t.Fatal("Expected ok to be true")
	}
}
