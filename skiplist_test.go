package skiplist

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestInsert(t *testing.T) {
	seed := time.Now().UnixNano()
	t.Logf("random seed: %v", seed)
	rng := rand.New(rand.NewSource(seed))

	s := newInt64SkipList()
	added := make(map[int64]bool)
	for i := int(0); i < 20; i++ {
		x := nextRandInt64(rng)
		if added[x] != s.Contains(x) {
			t.Errorf("added[%d] == %v, but Contains(%d) == %v",
				x, added[x], x, s.Contains(x))
		}
		shouldHavePrev := added[x]
		prev := s.Insert(x) // TODO(keunwoo): test return value
		if shouldHavePrev != (prev != nil) {
			t.Errorf("prev was %v (should we have a prev? %v)", prev, shouldHavePrev)
		}
		added[x] = true
		if !s.Contains(x) {
			t.Errorf("after adding %d, s.Contains(%d) is still false", x, x)
		}
		if t.Failed() {
			t.Fatalf("aborting test; skiplist state: %v", s)
		}
	}
	t.Logf("Final skiplist state: %v", s)
}

// Suggested invocation to see some random skip lists:
//
//   SKIPLIST_MAX_RAND=1000 go test -run TestForEach -v -count 20 github.com/keunwoo/skiplist
func TestForEach(t *testing.T) {
	seed := time.Now().UnixNano()
	t.Logf("random seed: %v", seed)
	rng := rand.New(rand.NewSource(seed))

	s := newInt64SkipList()
	added := make(map[int64]bool)
	for i := int(5); i < 37; i++ {
		x := nextRandInt64(rng)
		_ = s.Insert(x)
		added[x] = true
	}
	expectedCount := len(added)
	count := 0
	var prev int64
	_ = s.ForEach(func(i interface{}) error {
		x := i.(int64)
		if count > 0 && x <= prev {
			t.Errorf("got value %d out of order after %d", x, prev)
		}
		if !added[x] {
			t.Errorf("got value %d which was never added", x)
		}
		delete(added, x)
		prev = x
		count++
		return nil
	})
	if expectedCount != count {
		t.Errorf("Expected %d items, got %d", expectedCount, count)
	}
	if len(added) > 0 {
		t.Errorf("Iteration missed expected items: %v", added)
	}
	t.Logf("Final skiplist state: %v", s)
}

// TestJustSomeLogging is not really a test; it just contains t.Logf calls that
// print a simple skip list while it is being constructed.  This can be used
// during manual debugging as follows:
//
//   go test -run TestJustSomeLogging -v github.com/keunwoo/skiplist
func TestJustSomeLogging(t *testing.T) {
	s := newInt64SkipList()
	t.Logf("s.Contains(4) (empty): %v", s.Contains(int64(4)))
	s.Insert(int64(4))
	t.Logf("s.Contains(4) (inserted): %v", s.Contains(int64(4)))
	for _, i := range []int64{3, 5, 7, -2, 43, 20, -27} {
		s.Insert(i)
		t.Logf("skiplist after inserting %d: %v", i, s.String())
	}
}

func newInt64SkipList() *SkipList {
	return New(
		func(a, b interface{}) int {
			ai := a.(int64)
			bi := b.(int64)
			if ai < bi {
				return -1
			}
			if ai == bi {
				return 0
			}
			return 1
		},
		10)
}

// Set SKIPLIST_MAX_RAND to a small number to get more readable sample values.
func nextRandInt64(rng *rand.Rand) int64 {
	if max, _ := os.LookupEnv("SKIPLIST_MAX_RAND"); max != "" {
		maxParsed, err := strconv.ParseInt(max, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Invalid SKIPLIST_MAX_RAND: %q", max))
		}
		return rng.Int63n(maxParsed)
	}
	return rng.Int63()
}
