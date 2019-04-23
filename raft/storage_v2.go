// Copyright 2015 The etcd Authors
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

package raft

import (
	pb "go.etcd.io/etcd/raft/raftpb"
)

// Storage is an interface that may be implemented by the application
// to retrieve log entries from storage.
//
// If any Storage method returns an error, the raft instance will
// become inoperable and refuse to participate in elections; the
// application is responsible for cleanup and recovery in this case.
type StorageV2 interface {
	// InitialState returns the saved HardState and ConfState information.
	InitialState() (pb.HardStateV2, pb.ConfStateV2, error)
	// Entries returns a slice of log entries in the range [lo,hi).
	// MaxSize limits the total size of the log entries returned, but
	// Entries returns at least one entry if any.
	Entries(lo, hi, maxSize uint64) ([]pb.Entry, error)
	// Term returns the term of entry i, which must be in the range
	// [FirstIndex()-1, LastIndex()]. The term of the entry before
	// FirstIndex is retained for matching purposes even though the
	// rest of that entry may not be available.
	Term(i uint64) (uint64, error)
	// LastIndex returns the index of the last entry in the log.
	LastIndex() (uint64, error)
	// FirstIndex returns the index of the first log entry that is
	// possibly available via Entries (older entries have been incorporated
	// into the latest Snapshot; if storage only contains the dummy entry the
	// first log entry is not available).
	FirstIndex() (uint64, error)
	// Snapshot returns the most recent snapshot.
	// If snapshot is temporarily unavailable, it should return ErrSnapshotTemporarilyUnavailable,
	// so raft state machine could know that Storage needs some time to prepare
	// snapshot and call Snapshot later.
	Snapshot() (pb.SnapshotV2, error)
}

// compatStore masquerades a Storage as a StorageV2, however without allowing the
// use of joint consensus. It exists to allow applications to continue using the
// v1 membership change protocol without duplicating a lot of code internally.
type compatStorage struct {
	s Storage
}

var _ StorageV2 = (*compatStorage)(nil)

func (cps *compatStorage) InitialState() (pb.HardStateV2, pb.ConfStateV2, error) {
	hs, cs, err := cps.s.InitialState()
	if err != nil {
		return pb.HardStateV2{}, pb.ConfStateV2{}, err
	}

	return hs.V2(), cs.V2(), nil
}

func (cps *compatStorage) Entries(lo, hi, maxSize uint64) ([]pb.Entry, error) {
	return cps.s.Entries(lo, hi, maxSize)
}
func (cps *compatStorage) Term(i uint64) (uint64, error) {
	return cps.s.Term(i)
}
func (cps *compatStorage) LastIndex() (uint64, error) {
	return cps.s.LastIndex()
}

func (cps *compatStorage) FirstIndex() (uint64, error) {
	return cps.s.FirstIndex()
}

func (cps *compatStorage) Snapshot() (pb.SnapshotV2, error) {
	snap, err := cps.s.Snapshot()
	if err != nil {
		return pb.SnapshotV2{}, nil
	}
	return snap.V2(), nil
}

// MemoryStorageV2 implements the Storage interface backed by an
// in-memory array.
//
// TODO(tbg): actually make this implement the V2 functionality.
type MemoryStorageV2 struct {
	*compatStorage
	actual *MemoryStorage
}

// NewMemoryStorageV2 creates an empty MemoryStorageV2.
func NewMemoryStorageV2() *MemoryStorageV2 {
	actual := NewMemoryStorage()
	cp := &compatStorage{actual}
	return &MemoryStorageV2{
		compatStorage: cp,
		actual:        actual,
	}
}
