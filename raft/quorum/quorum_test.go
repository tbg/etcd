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
	"strings"
	"testing"

	"github.com/cockroachdb/datadriven"
)

func TestMajorityQuorum(t *testing.T) {

	const input = `
# The empty quorum commits index zero only. This isn't a case relevant in
# practice.
committed
----
0 (final)



# A single voter quorum is not final when no index is known.
committed cfg=(1)
----
0 (not final)

# When an index is known, that's the committed index, and that's final.
committed cfg=(1) idx=(12)
----
12 (final)




# With two nodes, start out similarly.
committed cfg=(1, 2)
----
0 (not final)

# The first committed index becomes known (for n1). Nothing changes in the
# output because idx=12 is not known to be on a quorum (which is both nodes).
committed cfg=(1, 2) idx=(12)
----
0 (not final)

# The second index comes in and finalize the decision. The result will be the
# smaller of the two indexes.
committed cfg=(1,2) idx=(12,5)
----
5 (final)




# No surprises for three nodes.
committed cfg=(1,2,3)
----
0 (not final)

committed cfg=(1,2,3) idx=(12)
----
0 (not final)

# We see a committed index, but a higher committed index for the last pending
# votes could change (increment) the outcome, so not final yet.
committed cfg=(1,2,3) idx=(12,5)
----
5 (not final)

# a) the case in which it does:
committed cfg=(1,2,3) idx=(12,5,6)
----
6 (final)

# b) the case in which it does not:
committed cfg=(1,2,3) idx=(12,5,4)
----
5 (final)

# c) a different case in which the last index is pending but it has no chance of
# swaying the outcome (because nobody in the current quorum agrees on anything
# higher than the candidate):
committed cfg=(1,2,3) idx=(5,5)
----
5 (final)

# c) continued: Doesn't matter what shows up last. The result is final.
committed cfg=(1,2,3) idx=(5,5,12)
----
5 (final)

# With all committed idx known, the result is final.
committed cfg=(1, 2, 3) idx=(100, 101, 103)
----
101 (final)



# Some more complicated examples. Similar to case c) above. The result is
# already final because no index higher than 103 is one short of quorum.
committed cfg=(1, 2, 3, 4, 5) idx=(101, 104, 103, 103)
----
103 (final)

# A similar case which is not final because another vote for >= 103 would change
# the outcome.
committed cfg=(1, 2, 3, 4, 5) idx=(101, 102, 103, 103)
----
102 (not final)
`
	datadriven.RunTestFromString(t, input, func(d *datadriven.TestData) string {
		var ids []uint64
		var idxs []uint64
		for _, arg := range d.CmdArgs {
			for i := range arg.Vals {
				var n uint64
				arg.Scan(t, i, &n)
				switch arg.Key {
				case "cfg":
					ids = append(ids, n)
				case "idx":
					idxs = append(idxs, n)
				default:
					t.Fatalf("unknown arg %s", arg.Key)
				}
			}
		}

		c := MajorityConfig{}
		l := mapLookuper{}
		for i, id := range ids {
			c[id] = struct{}{}
			if i < len(idxs) {
				l[id] = idxs[i]
			}
		}

		idx, final := c.CommittedIndex(l)

		s := map[bool]string{
			true:  "final",
			false: "not final",
		}

		var buf strings.Builder
		if aidx, afinal := alternativeMajorityCommittedIndex(c, l); aidx != idx || afinal != final {
			fmt.Fprintf(&buf, "%d (%s) <-- via alternative computation\n", aidx, s[afinal])
		}

		fmt.Fprintf(&buf, "%d (%s)\n", idx, s[final])
		return buf.String()
	})
}
