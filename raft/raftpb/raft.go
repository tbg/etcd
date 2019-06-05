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

package raftpb

import "fmt"

// ConfChangeV2er is a message that is equivalent to a ConfChangeV2. In
// practice, these messages are either a ConfChangeV2 or a (legacy) ConfChange.
type ConfChangeV2er interface {
	AsConfChangeV2() ConfChangeV2
}

// AsConfChangeV2 returns a V2 configuration change carrying out the same operation.
// This implements ConfChangeV2er.
func (cc ConfChange) AsConfChangeV2() ConfChangeV2 {
	return ConfChangeV2{
		Changes: []ConfChangeSimple{{
			ID:     cc.ID,
			Type:   cc.Type,
			NodeID: cc.NodeID,
		}},
		Context: cc.Context,
	}
}

// AsConfChangeV2 is the identity.
// This implements ConfChangeV2er.
func (cc ConfChangeV2) AsConfChangeV2() ConfChangeV2 { return cc }

// JointConsensus returns true if the configuration change will use Joint
// Consensus, which is the case if it contains more than more than one
// addition/removal of a voter. (Any changes to learners can be made).
//
// In particular, a change that adds and immediately removes a voter will have
// to use joint consensus. This is because what ultimately matters is the
// comparison between the base configuration (that this change applies on top
// of) and the result. The base isn't known, so we're conservatively using Joint
// Consensus whenever it *may* be necessary.
func (cc *ConfChangeV2) JointConsensus() bool {
	if cc.Transition != ConfChangeTransitionAuto {
		return true
	}
	changed := false
	for _, c := range cc.Changes {
		switch c.Type {
		case ConfChangeAddNode, ConfChangeRemoveNode:
			if changed {
				return false
			}
			changed = true
		case ConfChangeAddLearnerNode, ConfChangeUpdateNode:
		default:
			panic(fmt.Sprintf("unknown config change type in %+v", cc))
		}
	}
	return true
}
