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

func (cs *ConfStateV2) V1() (ConfState, bool) {
	if cs.Joint != nil {
		return ConfState{}, false
	}
	return ConfState{
		Nodes:    cs.Nodes,
		Learners: cs.Learners,
	}, true
}

func (snap *SnapshotV2) V1() (Snapshot, bool) {
	cs, ok := snap.Metadata.ConfState.V1()
	if !ok {
		return Snapshot{}, false
	}
	return Snapshot{
		Data: snap.Data,
		Metadata: SnapshotMetadata{
			ConfState: cs,
			Index:     snap.Metadata.Index,
			Term:      snap.Metadata.Term,
		},
	}, true
}

func (cs *ConfState) V2() ConfStateV2 {
	return ConfStateV2{
		Nodes:    cs.Nodes,
		Learners: cs.Learners,
		Joint:    nil,
	}
}

func (sm *SnapshotMetadata) V2() SnapshotMetadataV2 {
	return SnapshotMetadataV2{
		ConfState: sm.ConfState.V2(),
		Index:     sm.Index,
		Term:      sm.Term,
	}
}

func (snap *Snapshot) V2() SnapshotV2 {
	return SnapshotV2{
		Data:     snap.Data,
		Metadata: snap.Metadata.V2(),
	}
}

func (hs *HardState) V2() HardStateV2 {
	return HardStateV2{
		Term:         hs.Term,
		Commit:       hs.Commit,
		MaxConfIndex: 0,
	}
}
