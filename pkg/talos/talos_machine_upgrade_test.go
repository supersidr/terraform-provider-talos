// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"testing"

	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/stretchr/testify/require"
)

func TestParseUpgradeRebootMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		expected   machineapi.UpgradeRequest_RebootMode
		shouldFail bool
	}{
		{
			name:     "empty defaults to default mode",
			input:    "",
			expected: machineapi.UpgradeRequest_DEFAULT,
		},
		{
			name:     "lower case mode",
			input:    "powercycle",
			expected: machineapi.UpgradeRequest_POWERCYCLE,
		},
		{
			name:     "mixed case with spaces",
			input:    "  DefAult  ",
			expected: machineapi.UpgradeRequest_DEFAULT,
		},
		{
			name:       "invalid mode",
			input:      "invalid-mode",
			shouldFail: true,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := parseUpgradeRebootMode(tc.input)
			if tc.shouldFail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
