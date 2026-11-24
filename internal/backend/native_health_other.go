//go:build !windows

package backend

import "errors"

func nativeParentTimes([]nativeCandidate) (map[int]uint64, error) {
	return nil, errors.New("Windows process API required")
}
