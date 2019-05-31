// Copyright 2019 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package raft

import (
	"math/rand"
	"reflect"
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
	n := rand.Intn(size)
	if n > 3 {
		n = 3
	}
	sl1, sl2 := rand.Perm(2 * n)[:n], rand.Perm(2 * n)[:n]

	m := map[uint64]uint64{}
	for i := range sl1 {
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
	f func(map[uint64]struct{}, IndexLookuper) (uint64, bool),
) func(memberMap, idxMap) (uint64, bool) {
	return func(v memberMap, l idxMap) (uint64, bool) {
		return f(v, mapLookuper(l))
	}
}

func majorityCommittedIdx(c map[uint64]struct{}, l IndexLookuper) (uint64, bool) {
	return MajorityConfig(c).CommittedIndex(l)
}

func TestVoteParity(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 10000,
	}
	if err := quick.CheckEqual(
		hideLookuper(majorityCommittedIdx),
		hideLookuper(dumbMajorityCommittedIdx),
		cfg,
	); err != nil {
		t.Fatal(err)
	}
}

func TestMajorityQuorum(t *testing.T) {
	datadriven.RunTestFromString
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
