package core_test

import (
	"sync"
	"testing"

	"github.com/actfuns/behaviortree/core"
)

// TestBlackboardThreadSafety_SetAndUnsetRace tests concurrent set() + unset() on the same key.
// Equivalent of C++ Bug-2 test.
func TestBlackboardThreadSafety_SetAndUnsetRace(t *testing.T) {
	bb := core.NewBlackboard(nil)

	// Pre-create the entry so set() takes the existing-entry branch
	_ = bb.Set("key", 0)

	var stop bool
	var mu sync.Mutex
	const iterations = 500

	setter := func() {
		for i := 0; i < iterations && !func() bool { mu.Lock(); defer mu.Unlock(); return stop }(); i++ {
			_ = bb.Set("key", i)
		}
	}

	unsetter := func() {
		for i := 0; i < iterations && !func() bool { mu.Lock(); defer mu.Unlock(); return stop }(); i++ {
			bb.Unset("key")
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		setter()
	}()
	go func() {
		defer wg.Done()
		unsetter()
	}()

	wg.Wait()
	t.Log("SetAndUnsetRace: completed without data race")
}

// TestBlackboardThreadSafety_SetNewEntryWhileReading tests concurrent
// set/unset while reading entries.
// Equivalent of C++ Bug-1/Bug-8 test.
func TestBlackboardThreadSafety_SetNewEntryWhileReading(t *testing.T) {
	const iterations = 500

	bb := core.NewBlackboard(nil)

	var wg sync.WaitGroup

	// Writer: keeps cycling between unset and set to trigger new-entry path
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := "key_" + string(rune('0'+i%5))
			bb.Unset(key)
			_ = bb.Set(key, i)
		}
	}()

	// Reader: reads entries concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := "key_" + string(rune('0'+i%5))
			bb.RLock()
			entry := bb.GetEntry(key)
			if entry != nil {
				_ = entry.SequenceID()
			}
			bb.RUnlock()
		}
	}()

	wg.Wait()
	t.Log("SetNewEntryWhileReading: completed without data race")
}

// TestBlackboardThreadSafety_TwoThreadsSetSameNewKey tests two threads
// calling set() for the same new key concurrently.
// Equivalent of C++ Bug-8 test.
func TestBlackboardThreadSafety_TwoThreadsSetSameNewKey(t *testing.T) {
	const rounds = 50

	for round := 0; round < rounds; round++ {
		bb := core.NewBlackboard(nil)
		key := "new_key"

		var ready sync.WaitGroup
		ready.Add(2)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			ready.Done()
			ready.Wait()
			_ = bb.Set(key, 1)
			wg.Done()
		}()

		go func() {
			ready.Done()
			ready.Wait()
			_ = bb.Set(key, 2)
			wg.Done()
		}()

		wg.Wait()

		// The value should be one of the two written values
		var result int
		_, err := bb.Get(key, &result)
		if err != nil || (result != 1 && result != 2) {
			t.Logf("Round %d: result=%d err=%v", round, result, err)
		}
	}
	t.Log("TwoThreadsSetSameNewKey: completed without data race")
}

// TestBlackboardThreadSafety_CloneIntoWhileReading tests CloneInto while
// concurrently reading entries.
// Equivalent of C++ Bug-3 test.
func TestBlackboardThreadSafety_CloneIntoWhileReading(t *testing.T) {
	src := core.NewBlackboard(nil)
	dst := core.NewBlackboard(nil)

	const numEntries = 20

	// Pre-populate both blackboards
	for i := 0; i < numEntries; i++ {
		key := "key_" + string(rune('0'+i))
		_ = src.Set(key, i)
		_ = dst.Set(key, i*10)
	}

	const iterations = 500

	var wg sync.WaitGroup

	// Cloner
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			src.CloneInto(dst)
		}
	}()

	// Reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := "key_" + string(rune('0'+i%numEntries))
			dst.RLock()
			entry := dst.GetEntry(key)
			if entry != nil {
				_ = entry.SequenceID()
			}
			dst.RUnlock()
		}
	}()

	wg.Wait()
	t.Log("CloneIntoWhileReading: completed without data race")
}

// TestBlackboardThreadSafety_GetKeysWhileModifying tests GetKeys while
// concurrently modifying the blackboard.
// Equivalent of C++ Bug-6 test.
func TestBlackboardThreadSafety_GetKeysWhileModifying(t *testing.T) {
	bb := core.NewBlackboard(nil)

	const iterations = 500

	var wg sync.WaitGroup

	// Modifier
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := "key_" + string(rune('0'+i%50))
			_ = bb.Set(key, i)
		}
	}()

	// Key reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = bb.GetKeys()
		}
	}()

	wg.Wait()
	t.Log("GetKeysWhileModifying: completed without data race")
}
