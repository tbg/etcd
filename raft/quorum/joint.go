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

// JointConfig is a configuration of two groups of (possibly overlapping)
// majority configurations. Decisions require the support of both majorities.
type JointConfig [2]MajorityConfig

// Union returns a map corresponding to the union of the joint majority quorums.
func (c JointConfig) Union() map[uint64]struct{} {
	m := map[uint64]struct{}{}
	for _, mc := range c {
		for id := range mc {
			m[id] = struct{}{}
		}
	}
	return m
}

// Describe returns a (multi-line) representation of the commit indexes for the
// given lookuper.
func (c JointConfig) Describe(l IndexLookuper) string {
	return MajorityConfig(c.Union()).Describe(l)
}

func min(a, b uint64) uint64 {
	if a > b {
		return b
	}
	return a
}

// CommittedIndex returns a CommitRange for the given joint quorum. An index is
// jointly committed if it is committed on both constituent majorities.
func (c JointConfig) CommittedIndex(l IndexLookuper) CommitRange {
	cr1 := c[0].CommittedIndex(l)
	cr2 := c[1].CommittedIndex(l)

	return CommitRange{
		Definitely: min(cr1.Definitely, cr2.Definitely),
		Maybe:      min(cr1.Maybe, cr2.Maybe),
	}
}

// VoteResult takes a mapping of voters to yes/no (true/false) votes and returns
// a result indicating whether the vote is pending, lost, or won. A joint quorum
// requires both majority quorums to vote in favor.
func (c JointConfig) VoteResult(votes map[uint64]bool) VoteResult {
	r1 := c[0].VoteResult(votes)
	r2 := c[1].VoteResult(votes)

	if r1 == r2 {
		return r1
	}

	// Reduce number of cases via symmetry.
	if r1 > r2 {
		r1, r2 = r2, r1
	}

	// Now we're in either of the cases
	// - (pending, lost)
	// - (pending, won)
	// - (lost, pending)
	// - (lost, won)
	if r2 == VoteLost {
		return VoteLost
	}
	return r1
}
