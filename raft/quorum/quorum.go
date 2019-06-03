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
	"strconv"
)

// IndexLookuper allows looking up a commit index for a given ID of a voter
// from a corresponding MajorityConfig.
type IndexLookuper interface {
	Index(voterID uint64) (idx uint64, found bool)
}

type mapLookuper map[uint64]uint64

func (m mapLookuper) Index(id uint64) (uint64, bool) {
	idx, ok := m[id]
	return idx, ok
}

// CommitRange is a tuple of commit indexes, one of which is known to be
// committed, and the other one being the highest one that could potentially
// be committed as more voters report back.
type CommitRange struct {
	Definitely uint64
	Maybe      uint64
}

// String implements fmt.Stringer. A CommitRange is printed as a uint64 if it
// contains only one possible commit index; otherwise it's printed as a-b with
// some special casing applied to the maximum unit64 value.
func (cr CommitRange) String() string {
	if cr.Maybe == math.MaxUint64 {
		if cr.Definitely == math.MaxUint64 {
			return "∞"
		}
		return fmt.Sprintf("%d-∞", cr.Definitely)
	}
	if cr.Definitely == cr.Maybe {
		return strconv.FormatUint(cr.Definitely, 10)
	}
	return fmt.Sprintf("%d-%d", cr.Definitely, cr.Maybe)
}

// VoteResult indicates the outcome of a vote.
//
//go:generate stringer -type=VoteResult
type VoteResult uint8

const (
	// VotePending indicates that the decision of the vote depends on future
	// votes, i.e. neither "yes" or "no" has reached quorum yet.
	VotePending VoteResult = 1 + iota
	// VoteLost indicates that the quorum has voted "no".
	VoteLost
	// VoteWon indicates that the quorum has voted "yes".
	VoteWon
)
