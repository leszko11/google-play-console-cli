package shared

import (
	"fmt"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

// SelectRolloutToHalt returns the single in-progress release that can be halted safely.
func SelectRolloutToHalt(track gpc.TrackInfo) (gpc.TrackReleaseInfo, error) {
	inProgress := make([]gpc.TrackReleaseInfo, 0, len(track.Releases))
	hasCompleted := false
	for _, release := range track.Releases {
		switch strings.TrimSpace(release.Status) {
		case "inProgress":
			inProgress = append(inProgress, release)
		case "completed":
			hasCompleted = true
		}
		if release.UserFraction >= 1 {
			hasCompleted = true
		}
	}

	switch len(inProgress) {
	case 1:
		if len(inProgress[0].VersionCodes) == 0 {
			return gpc.TrackReleaseInfo{}, fmt.Errorf("in-progress release on track %q has no version codes", track.Name)
		}
		return inProgress[0], nil
	case 0:
		if hasCompleted {
			return gpc.TrackReleaseInfo{}, fmt.Errorf("cannot halt a completed rollout on track %q", track.Name)
		}
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has no in-progress release to halt", track.Name)
	default:
		return gpc.TrackReleaseInfo{}, fmt.Errorf("track %q has multiple in-progress releases; refusing to halt implicitly", track.Name)
	}
}
