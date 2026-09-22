package fauxinnati

import (
	"fmt"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/google/go-cmp/cmp"
)

func TestLatestPatchVersion(t *testing.T) {
	tests := []struct {
		name           string
		queriedVersion semver.Version
		version        semver.Version
		latest         semver.Version
		latestErr      error

		expected semver.Version
	}{
		{
			name:           "channel head between the queried and the synthesized version is used",
			queriedVersion: semver.MustParse("4.17.5"),
			version:        semver.MustParse("4.17.9"),
			latest:         semver.MustParse("4.17.7"),
			expected:       semver.MustParse("4.17.7"),
		},
		{
			name:           "channel head equal to the synthesized version is used",
			queriedVersion: semver.MustParse("4.17.5"),
			version:        semver.MustParse("4.17.6"),
			latest:         semver.MustParse("4.17.6"),
			expected:       semver.MustParse("4.17.6"),
		},
		{
			name:           "prerelease channel head of the same minor is used",
			queriedVersion: semver.MustParse("5.0.0-rc.1"),
			version:        semver.MustParse("5.0.1"),
			latest:         semver.MustParse("5.0.0-rc.2"),
			expected:       semver.MustParse("5.0.0-rc.2"),
		},
		{
			name:           "channel head from an older minor is not used",
			queriedVersion: semver.MustParse("5.0.0-0.nightly-2026-09-21-030332"),
			version:        semver.MustParse("5.1.0"),
			latest:         semver.MustParse("5.0.0-rc.2"),
			expected:       semver.MustParse("5.1.0"),
		},
		{
			name:           "channel head from an older major is not used",
			queriedVersion: semver.MustParse("4.18.0"),
			version:        semver.MustParse("5.0.1"),
			latest:         semver.MustParse("4.19.0"),
			expected:       semver.MustParse("5.0.1"),
		},
		{
			name:           "channel head newer than the synthesized version is not used",
			queriedVersion: semver.MustParse("4.17.5"),
			version:        semver.MustParse("4.17.6"),
			latest:         semver.MustParse("4.17.9"),
			expected:       semver.MustParse("4.17.6"),
		},
		{
			name:           "channel head not newer than the queried version is not used",
			queriedVersion: semver.MustParse("4.17.5"),
			version:        semver.MustParse("4.17.9"),
			latest:         semver.MustParse("4.17.5"),
			expected:       semver.MustParse("4.17.9"),
		},
		{
			name:           "failed lookup falls back to the synthesized version",
			queriedVersion: semver.MustParse("4.17.5"),
			version:        semver.MustParse("4.17.9"),
			latestErr:      fmt.Errorf("no candidates found for version 4"),
			expected:       semver.MustParse("4.17.9"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMajor, gotMinor uint64
			getLatest := func(_ Client, major, minor uint64) (semver.Version, error) {
				gotMajor, gotMinor = major, minor
				return tt.latest, tt.latestErr
			}

			actual := resolvePatchVersion(nil, tt.queriedVersion, tt.version, getLatest)

			if diff := cmp.Diff(tt.expected.String(), actual.String()); diff != "" {
				t.Errorf("unexpected version (-expected +actual):\n%s", diff)
			}
			// The channel to consult comes from version, never from queriedVersion.
			if diff := cmp.Diff(fmt.Sprintf("%d.%d", tt.version.Major, tt.version.Minor), fmt.Sprintf("%d.%d", gotMajor, gotMinor)); diff != "" {
				t.Errorf("unexpected major.minor passed to the channel lookup (-expected +actual):\n%s", diff)
			}
		})
	}
}
