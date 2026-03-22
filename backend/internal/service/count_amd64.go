package service

// countStatuses counts encoded task statuses using SSE2 + POPCNT.
// Encoding: 0 = open, 1 = in_progress, 2 = done.
//
// Implemented in count_amd64.s.
//
//go:noescape
func countStatuses(statuses []byte) (todo, inProgress, done int)
