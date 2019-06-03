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
	"math"
	"sort"
	"strings"
	"sync"
)

// MajorityConfig is a set of IDs that uses majority quorums to make decisions.
type MajorityConfig map[uint64]struct{}

// Describe returns a (multi-line) representation of the commit indexes for the
// given lookuper.
func (c MajorityConfig) Describe(l IndexLookuper) string {
	if len(c) == 0 {
		return "<empty majority quorum>"
	}
	type tup struct {
		id, idx uint64
		ok      bool // idx found?
		bar     int  // length of bar displayed for this tup
	}

	// Below, populate .bar so that the i-th largest commit index has bar i (we
	// plot this as sort of a progress bar). The actual code is a bit more
	// complicated and also makes sure that equal index => equal bar.

	n := len(c)
	info := make([]tup, 0, n)
	for id := range c {
		idx, ok := l.Index(id)
		info = append(info, tup{id: id, idx: idx, ok: ok})
	}

	// Sort by index
	sort.Slice(info, func(i, j int) bool {
		if info[i].idx == info[j].idx {
			return info[i].id < info[j].id
		}
		return info[i].idx < info[j].idx
	})

	// Populate .bar.
	for i := range info {
		if i > 0 && info[i-1].idx < info[i].idx {
			info[i].bar = i
		}
	}

	// Sort by ID.
	sort.Slice(info, func(i, j int) bool {
		return info[i].id < info[j].id
	})

	var buf strings.Builder

	// Print.
	fmt.Fprint(&buf, strings.Repeat(" ", n)+"    idx\n")
	for i := range info {
		bar := info[i].bar
		if !info[i].ok {
			fmt.Fprint(&buf, "?"+strings.Repeat(" ", n))
		} else {
			fmt.Fprint(&buf, strings.Repeat("x", bar)+">"+strings.Repeat(" ", n-bar))
		}
		fmt.Fprintf(&buf, " %5d    (id=%d)\n", info[i].idx, info[i].id)
	}
	return buf.String()
}

// slicePool avoids allocations during commit index computations.
var slicePool = sync.Pool{New: func() interface{} {
	return make([]uint64, 0, 5)
}}

// CommittedIndex computes the committed index from those supplied via the
// provided IndexLookuper. The outcome (a CommitRange) is final (meaning that
// its two components agree) if enough voters are reflected in the
// IndexLookuper. Otherwise, necessarily the value for Maybe is larger than that
// for Defininitely (which is the largest index known committed based on the
// information so far) and future values for Definitely may increase (limited by
// Maybe) as previously missing voters are reflected in the IndexLookuper.
func (c MajorityConfig) CommittedIndex(l IndexLookuper) CommitRange {
	n := len(c)
	if n == 0 {
		return CommitRange{Definitely: math.MaxUint64, Maybe: math.MaxUint64}
	}

	srt := slicePool.Get().([]uint64)
	defer slicePool.Put(srt)

	if cap(srt) < n {
		srt = make([]uint64, n)
	}
	srt = srt[:n]

	var votesCast int
	{
		// Fill the slice from the right with the indexes observed. Any unused
		// slots on the left will be zeroed; these correspond to voters that
		// may report in, but haven't yet.
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

	// Sort by index.
	sort.Slice(srt, func(i, j int) bool { return srt[i] < srt[j] })

	// The smallest index into the array for which the value is acked by a
	// quorum. In other words, from the end of the slice, move n/2+1 to the
	// left (accounting for zero-indexing).
	pos := n - (n/2 + 1)

	// Every additional voter participating in the future has the potential to
	// "shift" srt towards index zero by adding a high idx. But there are limits
	// to this due to the number of outstanding votes (n-votesCast). We can't
	// shift more often than that, so the idx that would result from this (if
	// already determined) limits how high the final commit index can turn out.
	hi := uint64(math.MaxUint64)
	if votesCast > pos {
		hi = srt[pos+n-votesCast]
	}

	return CommitRange{Definitely: srt[pos], Maybe: hi}
}

// VoteResult takes a mapping of voters to yes/no (true/false) votes and returns
// a result indicating whether the vote is pending (i.e. neither a quorum of
// yes/no has been reached), won (a quorum of yes has been reached), or lost (a
// quorum of no has been reached).
func (c MajorityConfig) VoteResult(votes map[uint64]bool) VoteResult {
	return voteResultVia(c, votes, func(c MajorityConfig, l IndexLookuper) CommitRange {
		return c.CommittedIndex(l)
	})
}

func voteResultVia(
	c MajorityConfig,
	votes map[uint64]bool,
	committedIndex func(MajorityConfig, IndexLookuper) CommitRange,
) VoteResult {
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
	cr := committedIndex(c, l)
	if cr.Definitely != cr.Maybe {
		return VotePending
	}
	// NB: the zero config wins all votes. This happens to be convenient
	// behavior when using joint quorums.
	if cr.Definitely == 1 || len(c) == 0 {
		return VoteWon
	}
	return VoteLost
}
