// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package version

import "testing"

func TestVersion(t *testing.T) {
	// The Version variable is defined in version.go and is not empty by default.
	if Version == "" {
		t.Error("Version should not be empty")
	}
}
