// Copyright 2019 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package quorum

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/cockroachdb/datadriven"
)

type mapLookuper map[uint64]uint64

func (m mapLookuper) Index(id uint64) (uint64, bool) {
	idx, ok := m[id]
	return idx, ok
}

func genMap(rand *rand.Rand, size int) map[uint64]uint64 {
	// Hard-code a reasonably small size here (quick will hard-code 50, which
	// is not useful here).
	size = 11

	n1, n2 := rand.Intn(size), rand.Intn(size)
	sl1, sl2 := rand.Perm(2 * n1)[:n1], rand.Perm(2 * n2)[:n2]

	m := map[uint64]uint64{}
	n := len(sl1)
	if len(sl2) < len(sl1) {
		n = len(sl2)
	}
	for i := 0; i < n; i++ {
		m[uint64(sl1[i])] = uint64(sl2[i])
	}
	return m
}

type idxMap map[uint64]uint64

func (idxMap) Generate(rand *rand.Rand, size int) reflect.Value {
	m := genMap(rand, size)
	return reflect.ValueOf(m)
}

type memberMap map[uint64]struct{}

func (memberMap) Generate(rand *rand.Rand, size int) reflect.Value {
	m := genMap(rand, size)
	mm := map[uint64]struct{}{}
	for id := range m {
		mm[id] = struct{}{}
	}
	return reflect.ValueOf(mm)
}

func hideLookuper(
	f func(MajorityConfig, IndexLookuper) (uint64, bool),
) func(memberMap, idxMap) (uint64, bool) {
	return func(v memberMap, l idxMap) (uint64, bool) {
		return f(MajorityConfig(v), mapLookuper(l))
	}
}

func majorityCommittedIdx(c MajorityConfig, l IndexLookuper) (uint64, bool) {
	return MajorityConfig(c).CommittedIndex(l)
}

func alternativeMajorityCommittedIndex(c MajorityConfig, l IndexLookuper) (_committedIdx uint64, _malleable bool) {
	idToIdx := map[uint64]uint64{}
	for id := range c {
		if idx, ok := l.Index(id); ok {
			idToIdx[id] = idx
		}
	}

	pendingVotes := len(c) - len(idToIdx)

	// Build a map from index to voters who have acked that or any higher index.
	idxToVotes := map[uint64]int{}
	for _, idx := range idToIdx {
		idxToVotes[idx] = 0
	}

	for _, idx := range idToIdx {
		for idy := range idxToVotes {
			if idy > idx {
				continue
			}
			idxToVotes[idy]++
		}
	}

	// Find the maximum index that has achieved quorum.
	q := len(c)/2 + 1
	var maxQuorumIdx uint64
	for idx, n := range idxToVotes {
		if n >= q && idx > maxQuorumIdx {
			maxQuorumIdx = idx
		}
	}

	// Check whether there's a higher index that could receive
	// more votes, reach a quorum, and thus increase maxQuorumIdx
	// in the future.
	final := pendingVotes < q
	for idx, n := range idxToVotes {
		if idx <= maxQuorumIdx {
			continue
		}
		if n+pendingVotes >= q {
			final = false
			break
		}
	}

	return maxQuorumIdx, final
}

func TestVoteParity(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10000,
	}
	if err := quick.CheckEqual(
		hideLookuper(majorityCommittedIdx),
		hideLookuper(alternativeMajorityCommittedIndex),
		cfg,
	); err != nil {
		t.Fatal(err)
	}
}

func TestMajorityQuorum(t *testing.T) {

	const input = `
# The empty quorum commits index zero only. This isn't a case relevant in
# practice.
committed
----
0 (final)



# A single voter quorum is not final when no index is known.
committed cfg=(1)
----
0 (not final)

# When an index is known, that's the committed index, and that's final.
committed cfg=(1) idx=(12)
----
12 (final)




# With two nodes, start out similarly.
committed cfg=(1, 2)
----
0 (not final)

# The first committed index becomes known (for n1). Nothing changes in the
# output because idx=12 is not known to be on a quorum (which is both nodes).
committed cfg=(1, 2) idx=(12)
----
0 (not final)

# The second index comes in and finalize the decision. The result will be the
# smaller of the two indexes.
committed cfg=(1,2) idx=(12,5)
----
5 (final)




# No surprises for three nodes.
committed cfg=(1,2,3)
----
0 (not final)

committed cfg=(1,2,3) idx=(12)
----
0 (not final)

# We see a committed index, but a higher committed index for the last pending
# votes could change (increment) the outcome, so not final yet.
committed cfg=(1,2,3) idx=(12,5)
----
5 (not final)

# a) the case in which it does:
committed cfg=(1,2,3) idx=(12,5,6)
----
6 (final)

# b) the case in which it does not:
committed cfg=(1,2,3) idx=(12,5,4)
----
5 (final)

# c) a different case in which the last index is pending but it has no chance of
# swaying the outcome (because nobody in the current quorum agrees on anything
# higher than the candidate):
committed cfg=(1,2,3) idx=(5,5)
----
5 (final)

# c) continued: Doesn't matter what shows up last. The result is final.
committed cfg=(1,2,3) idx=(5,5,12)
----
5 (final)

# With all committed idx known, the result is final.
committed cfg=(1, 2, 3) idx=(100, 101, 103)
----
101 (final)



# Some more complicated examples. Similar to case c) above. The result is
# already final because no index higher than 103 is one short of quorum.
committed cfg=(1, 2, 3, 4, 5) idx=(101, 104, 103, 103)
----
103 (final)

# A similar case which is not final because another vote for >= 103 would change
# the outcome.
committed cfg=(1, 2, 3, 4, 5) idx=(101, 102, 103, 103)
----
102 (not final)
`
	datadriven.RunTestFromString(t, input, func(d *datadriven.TestData) string {
		var ids []uint64
		var idxs []uint64
		for _, arg := range d.CmdArgs {
			for i := range arg.Vals {
				var n uint64
				arg.Scan(t, i, &n)
				switch arg.Key {
				case "cfg":
					ids = append(ids, n)
				case "idx":
					idxs = append(idxs, n)
				default:
					t.Fatalf("unknown arg %s", arg.Key)
				}
			}
		}

		c := MajorityConfig{}
		l := mapLookuper{}
		for i, id := range ids {
			c[id] = struct{}{}
			if i < len(idxs) {
				l[id] = idxs[i]
			}
		}

		idx, final := c.CommittedIndex(l)

		s := map[bool]string{
			true:  "final",
			false: "not final",
		}

		var buf strings.Builder
		if aidx, afinal := alternativeMajorityCommittedIndex(c, l); aidx != idx || afinal != final {
			fmt.Fprintf(&buf, "%d (%s) <-- via alternative computation", aidx, s[afinal])
		}

		fmt.Fprintf(&buf, "%d (%s)\n", idx, s[final])
		return buf.String()
	})
}

// 0 0 0 1 bet
// 0 x x    malleable

// 0 0 . 2 idToIdx
// . 0 0 not malleable

// 0 0 0 11 15 malleable
// 11 15 . . .

// 0 0 10 11 15
// 10 11 15 xx xx malleable

//     |
// x 0 0 1 1

// val(mididx + missing) > val(mididx)

//     |
// x x x x x

// . . 1 2 3
