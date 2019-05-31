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
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

// TestVoteParity uses quickcheck to heuristically assert that the main
// implementation of (MajorityConfig).CommittedIndex agrees with a "dumb"
// alternative version.
func TestVoteParity(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50000,
	}
	if err := quick.CheckEqual(
		hideLookuper(majorityCommittedIdx),
		hideLookuper(alternativeMajorityCommittedIndex),
		cfg,
	); err != nil {
		t.Fatal(err)
	}
}

func smallRandMap(rand *rand.Rand, size int) map[uint64]uint64 {
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
	m := smallRandMap(rand, size)
	return reflect.ValueOf(m)
}

type memberMap map[uint64]struct{}

func (memberMap) Generate(rand *rand.Rand, size int) reflect.Value {
	m := smallRandMap(rand, size)
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

// This is an alternative implementation of (MajorityConfig).CommittedIndex(l).
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
