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
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

// TestQuick uses quickcheck to heuristically assert that the main
// implementation of (MajorityConfig).CommittedIndex agrees with a "dumb"
// alternative version.
func TestQuick(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50000,
	}

	t.Run("majority_commit", func(t *testing.T) {
		fn1 := func(c memberMap, l idxMap) CommitRange {
			return MajorityConfig(c).CommittedIndex(mapLookuper(l))
		}
		fn2 := func(c memberMap, l idxMap) CommitRange {
			return alternativeMajorityCommittedIndex(MajorityConfig(c), mapLookuper(l))
		}
		if err := quick.CheckEqual(fn1, fn2, cfg); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("majority_vote", func(t *testing.T) {
		fn1 := func(c memberMap, votes votesMap) VoteResult {
			return MajorityConfig(c).VoteResult(votes)
		}

		fn2 := func(c memberMap, votes votesMap) VoteResult {
			return voteResultVia(MajorityConfig(c), votes, alternativeMajorityCommittedIndex)
		}

		if err := quick.CheckEqual(fn1, fn2, cfg); err != nil {
			t.Fatal(err)
		}
	})
}

// smallRandIdxMap returns a reasonably sized map of ids to commit indexes.
func smallRandIdxMap(rand *rand.Rand, size int) map[uint64]uint64 {
	// Hard-code a reasonably small size here (quick will hard-code 50, which
	// is not useful here).
	size = 10

	n := rand.Intn(size)
	ids := rand.Perm(2 * n)[:n]
	idxs := make([]int, len(ids))
	for i := range idxs {
		idxs[i] = rand.Intn(n)
	}

	m := map[uint64]uint64{}
	for i := range ids {
		m[uint64(ids[i])] = uint64(idxs[i])
	}
	return m
}

type idxMap map[uint64]uint64

func (idxMap) Generate(rand *rand.Rand, size int) reflect.Value {
	m := smallRandIdxMap(rand, size)
	return reflect.ValueOf(m)
}

type memberMap map[uint64]struct{}

func (memberMap) Generate(rand *rand.Rand, size int) reflect.Value {
	m := smallRandIdxMap(rand, size)
	mm := map[uint64]struct{}{}
	for id := range m {
		mm[id] = struct{}{}
	}
	return reflect.ValueOf(mm)
}

type votesMap map[uint64]bool

func (votesMap) Generate(rand *rand.Rand, size int) reflect.Value {
	m := smallRandIdxMap(rand, size)
	mm := map[uint64]bool{}
	for id := range m {
		mm[id] = m[id]%2 == 0
	}
	return reflect.ValueOf(mm)
}

// This is an alternative implementation of (MajorityConfig).CommittedIndex(l).
func alternativeMajorityCommittedIndex(c MajorityConfig, l IndexLookuper) CommitRange {
	if len(c) == 0 {
		return CommitRange{Definitely: math.MaxUint64, Maybe: math.MaxUint64}
	}

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

	// Find the largest index that could possibly achieve quorum via the pending
	// votes (infinity if a quorum is still open).
	var hi uint64
	for idx, n := range idxToVotes {
		if n+pendingVotes >= q && idx > hi {
			hi = idx
		}
	}
	if pendingVotes >= q {
		hi = math.MaxUint64
	}

	return CommitRange{Definitely: maxQuorumIdx, Maybe: hi}
}
