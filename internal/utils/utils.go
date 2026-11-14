package utils

import (
	"io"
	"sort"
	"strings"
)

// FindLongestCommonPrefix finds the longest common path prefix of a set of paths. For example,
// given the following:
//
//	github.com/owner/repo/cmd/foo
//	github.com/owner/repo/cmd/bar
//
// The longest common prefix is: github.com/owner/repo/cmd/ (note the trailing slash is included).
func FindLongestCommonPrefix(paths []string) string {
	if len(paths) < 2 {
		return ""
	}
	// Sort the paths to optimize comparison.
	sort.Strings(paths)

	// After sorting, the prefix shared by the whole set is the prefix shared by
	// the first and last paths: every path in between is bounded by them.
	first, last := paths[0], paths[len(paths)-1]
	if first == last {
		return first
	}
	commonPrefix := first[:commonPrefixLength(first, last)]

	// Ensure the common prefix ends at a boundary.
	if n := strings.LastIndex(commonPrefix, "/"); n != -1 {
		return commonPrefix[:n+1]
	}
	return ""
}

// commonPrefixLength returns the number of leading bytes the given paths have
// in common. The scan stops at the first byte that differs, staying within the
// bounds of the shorter path.
func commonPrefixLength(first, last string) int {
	n := 0
	for n < min(len(first), len(last)) && first[n] == last[n] {
		n++
	}
	return n
}

// DiscardCloser is an io.Writer that implements io.Closer by doing nothing.
//
// https://github.com/golang/go/issues/22823
type WriteNopCloser struct {
	io.Writer
}

func (WriteNopCloser) Close() error {
	return nil
}
