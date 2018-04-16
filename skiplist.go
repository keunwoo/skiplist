// Package skiplist implements a very elementary skip list.
//
// This is not a production-quality skip list implementation.  I wrote it just
// to teach myself skip lists, because I didn't learn them in school and I've
// read various papers & the Wikipedia article about a half dozen times in my
// life since then, without retaining any understanding.
//
// About Skip Lists
//
// Most presentations of skip lists, including Bill Pugh's original, draw a
// diagram of a populated skip list, and then present algorithms for using them
// (which, incidentally, will break if you write them up naively in a real
// programming language --- you have to pay attention to details like Pugh's
// special treatment of nil).  Personally I find this hard to follow, compared
// to an inductive definition which starts with an empty skip list and describes
// how it is constructed.
//
// Inductively, we can describe skip lists as follows:
//
// A node is either nil, or a value plus a slice of node references of some
// height L, where L ranges from 1 to a maximum height M.
//
// A header is a slice of node references of height M.  Note that a header has
// no value associated with it, only a slice of node references.
//
// The empty skip list consists of a header whose node slice contains all
// pointers to nil.
//
// Inductively, a nonempty skip list is a header, a node N of randomly chosen
// height R (we will describe the random distribution later), and a successor
// skip list, such that the following conditions hold:
//
// (1) the header contains pointers to N for levels 0 through R - 1.
//
// (2) for levels L in R through M - 1, the header contains the values of the
// successor skip list's header at level L.
//
// (3) for levels L in 0 through R - 1, the node's successor slice contains the
// values of the succesor skip list's header level L.
//
// (4) the node's value V is less than all values in the successor skip list.
//
// Next we describe how skip lists are constructed in practice.
//
// TODO(keunwoo): text below needs more work.
//
// Again, an empty skip list is a header whose node slice contains all pointers
// to nil.
//
// An element is added to a skip list by choosing a random level L between 0 and
// M - 1, then inserting the node at some index i with level L.  Usually i is
// determined by a (total or partial) order over the domain of values inserted
// into the list; that is, a value V is inserted at the greatest index i such
// that the value V_j at all indices j < i is less than V.
//
// When a node N at a level L is inserted at index i, then for each level l in 0
// through L - 1, the successor list of height at least l with the greatest
// index j < i points to N at level l.  To do this, we search for the insertion
// point while recording the list of pointers that we traversed to get there,
// and then update those pointers on insertion.
//
// We define a special less-than relation between nodes and values such that the
// nil node has value greater than any legal value (a non-nil node's value is
// simply the value field of the node).
//
// Search proceeds as follows (in pseudocode):
//
//   fun search(V) ([]*node) {
//       current := head
//       for each level L from M - 1 to 0:
//           while current[L] < V:
//               current = current[L].next
//       return current
//
// Search returns the last node slice whose value at level 0 is less than V.
package skiplist

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// SkipList is a naive skip list implementation.
//
// This implementation is not currently thread-safe.
//
//   TODO(keunwoo): implement delete
//   TODO(keunwoo): benchmarks
//   TODO(keunwoo): more usage examples
//   TODO(keunwoo): thread safety
//   TODO(keunwoo): cache-conscious variant (see papers, e.g. M. Spiegel et al.)
//   TODO(keunwoo): helpers for common element types
//   TODO(keunwoo): support multiset/multimap usage (multiple items for which cmp == 0)
type SkipList struct {
	// Compare is the compare parameter to the New function that constructed
	// this SkipList.  It is exposed for convenience in client debugging only;
	// clients should not mutate it.
	// TODO(keunwoo): maybe this should just be hidden.
	Compare  func(a, b interface{}) int
	maxLevel int
	head     []*node
}

type node struct {
	val  interface{}
	next []*node
}

// New constructs a SkipList with the given comparator and maxLevel.
// maxLevel should be the log2 of the maximum anticipated element count.
// compare should return -1 if a < b, 0 if a == b, and 1 if a > b
//
// TODO(keunwoo): this API is both too basic (what if we add more performance
// parameters?) and too verbose in the common case (it's maybe not that useful
// to require the user to specify maxLevel).
func New(compare func(a, b interface{}) int, maxLevel int) *SkipList {
	return &SkipList{
		Compare:  compare,
		head:     make([]*node, maxLevel),
		maxLevel: maxLevel,
	}
}

func (s *SkipList) Contains(elem interface{}) bool {
	nodes := s.head
	for i := s.maxLevel - 1; i >= 0; i-- {
	inner:
		for {
			switch cmp := s.cmp(nodes[i], elem); cmp {
			case -1:
				nodes = nodes[i].next
			case 0:
				return true
			case 1:
				break inner
			default:
				panic("invalid result code from comparison")
			}
		}
	}
	n := nodes[0]
	return s.cmp(n, elem) == 0
}

// Returns nil when a new element is inserted.
// When an element is updated in-place, the prior element is returned.
func (s *SkipList) Insert(elem interface{}) interface{} {
	// Pointers that will have to be adjusted at the insertion point.
	updates := make([]**node, s.maxLevel)
	nodes := s.head
	for i := s.maxLevel - 1; i >= 0; i-- {
	inner:
		for {
			switch cmp := s.cmp(nodes[i], elem); cmp {
			case -1:
				nodes = nodes[i].next
			case 0:
				// Found an "equal" value; update it.
				prev := nodes[i].val
				nodes[i].val = elem
				return prev
			case 1:
				// We have reached a slice whose successor at the current level is
				// greater than elem.  This slice location may have to be updated
				// when we insert the value (depending on the ultimate level at
				// which the value is inserted).
				updates[i] = &nodes[i]
				break inner
			}
		}
	}
	// Dropped out of the loop; insertion must be necessary.
	// Pick a random level and update successor pointers.
	level := s.randLevel()
	n := &node{
		val:  elem,
		next: make([]*node, level),
	}
	for i := range n.next {
		n.next[i] = *updates[i]
		*updates[i] = n
	}
	return nil
}

// ForEach applies a callback to each element in this skip list.
// If the callback returns a non-nil error, iteration ceases, and the error is
// returned.
func (s *SkipList) ForEach(callback func(elem interface{}) error) error {
	for n := s.head[0]; n != nil; n = n.next[0] {
		err := callback(n.val)
		if err != nil {
			return err
		}
	}
	return nil
}

// String returns an ASCII art representation of this skip list, including its
// internal structure.
func (s *SkipList) String() string {
	const minValueLen = 5 // width of widest placeholder (e.g. ">nil") + 1

	// First, prepare all the stringified values on the first line of output.
	strValues := []string{strings.Repeat(" ", minValueLen)} // placeholder for header
	for cur := s.head[0]; cur != nil; cur = cur.next[0] {
		s := fmt.Sprintf(" %v ", cur.val)
		if len(s) < minValueLen {
			s += strings.Repeat(" ", minValueLen-len(s))
		}
		strValues = append(strValues, s)
	}

	// Now, march across the node slices, printing the pointers at each level.  Note that we
	// print "upside down"; the "wires" for the pointers at level L will be on line L + 1,
	// rather than the typical presentation where "higher" levels appear higher in the diagram.
	lines := make([][]string, s.maxLevel+1)
	lines[0] = strValues
	// true if there is currently an incoming pointer at a given level
	inPtr := make([]bool, s.maxLevel)
	i := -1 // current value index; incremented as we march across the lowest-level linked list
	cur := s.head
	for cur != nil {
		var curValue string
		if i == -1 {
			curValue = strings.Repeat(" ", minValueLen)
		} else {
			curValue = strValues[i]
		}
		for lvl := 0; lvl < s.maxLevel; lvl++ {
			segment := ""
			pad := " "
			if len(cur) <= lvl {
				if inPtr[lvl] {
					// "---------"
					pad = "-"
				}
			} else if cur[lvl] == nil {
				if inPtr[lvl] {
					// ">|nil     "
					segment = ">|nil "
				} else {
					// " |nil     "
					// Can only happen at the skip list header.
					segment = " |nil "
				}
				inPtr[lvl] = false
			} else {
				pad = "-"
				if inPtr[lvl] {
					// ">|--------"
					segment = ">|-"
				} else {
					// " |--------"
					segment = " |-"
				}
				inPtr[lvl] = true
			}
			if len(curValue) > len(segment) {
				segment += strings.Repeat(pad, len(curValue)-len(segment))
			}
			lines[lvl+1] = append(lines[lvl+1], segment)
		}
		i++
		if cur[0] == nil {
			cur = nil
		} else {
			cur = cur[0].next
		}
	}

	// Assemble final result.
	var buf []string
	buf = append(buf, fmt.Sprintf("{maxLevel: %v, nodes:\n", s.maxLevel))
	for _, l := range lines {
		buf = append(buf, strings.TrimRight(strings.Join(l, ""), " "), "\n")
	}
	buf = append(buf, "}")
	return strings.Join(buf, "")
}

func (s *SkipList) cmp(n *node, elem interface{}) int {
	// The nil node has value greater than any legal element value.
	if n == nil {
		return 1
	}
	return s.Compare(n.val, elem)
}

func (s *SkipList) randLevel() int {
	rb := []byte{1}
	level := 1
	for level < s.maxLevel {
		// This hardcodes the promotion ratio of 1/2; each level contains half as many nodes
		// as the level below it.

		// TODO(keunwoo): we could be more efficient about using the RNG here; here we throw
		// away 7/8 bits of randomness.  Probably this is lost in the noise.
		_, err := rand.Read(rb)
		if err != nil {
			panic(fmt.Sprintf("error getting random level: %v", err))
		}
		if rb[0]&1 == 1 {
			level++
		} else {
			break
		}
	}
	return level
}
