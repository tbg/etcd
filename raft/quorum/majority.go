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
	"sort"
	"sync"
)

// A MajorityConfig is a set of IDs that uses majority quorums to make decisions.
type MajorityConfig map[uint64]struct{}

// An IndexLookuper allows looking up a commit index for a given ID of a voter
// from a corresponding MajorityConfig.
type IndexLookuper interface {
	Index(voterID uint64) (idx uint64, found bool)
}

type mapLookuper map[uint64]uint64

func (m mapLookuper) Index(id uint64) (uint64, bool) {
	idx, ok := m[id]
	return idx, ok
}

var slicePool = sync.Pool{New: func() interface{} {
	return make([]uint64, 0, 5)
}}

// CommittedIndex computes a candidate committed index from those reflected in
// the provided IndexLookuper. If final is false, the candidate may increase
// as additional commit indexes become known. If final is true, the candidate
// commit index will not change, even if no commit index is known for some of
// the voters.
func (c MajorityConfig) CommittedIndex(l IndexLookuper) (committedIdx uint64, final bool) {
	n := len(c)
	if n == 0 {
		return 0, true
	}

	srt := slicePool.Get().([]uint64)
	defer slicePool.Put(srt)

	if cap(srt) < n {
		srt = make([]uint64, n)
	}
	srt = srt[:n]

	var votesCast int
	{
		i := n - 1
		for id := range c {
			if idx, ok := l.Index(id); ok {
				srt[i] = idx
				i--
			}
		}
		votesCast = (n - 1) - i
		for j := 0; j <= i; j++ {
			srt[j] = 0
		}
	}

	sort.Slice(srt, func(i, j int) bool { return srt[i] < srt[j] })
	pos := n - (n/2 + 1)

	return srt[pos], votesCast > pos && srt[pos+n-votesCast] == srt[pos]
}

// A VoteResult indicates the outcome of a vote.
//go:generate stringer -type=VoteResult
type VoteResult uint8

const (
	// VotePending indicates that the decision of the vote depends on future
	// votes.
	VotePending VoteResult = iota
	// VoteLost indicates that a majority has voted "no".
	VoteLost
	// VoteLost indicates that a majority has voted "yes".
	VoteWon
)

// VoteResult takes a mapping of voters to yes/no (true/false) votes and returns
// a result indicating whether the vote is pending (i.e. neither a quorum of
// yes/no has been reached), won (a quorum of yes has been reached), or lost (a
// quorum of no has been reached).
func (c MajorityConfig) VoteResult(votes map[uint64]bool) VoteResult {
	// A vote is just a CommittedIndex computation in which "yes" corresponds to
	// index one and "no" to index zero.
	l := mapLookuper{}
	for nodeID, vote := range votes {
		if !vote {
			l[nodeID] = 0
		} else {
			l[nodeID] = 1
		}
	}
	idx, final := c.CommittedIndex(l)
	if !final {
		return VotePending
	}
	if idx == 1 {
		return VoteWon
	}
	return VoteLost
}
