//go:build !amd64

package service

// countStatuses is the pure-Go fallback used on architectures where
// the SIMD (SSE2 + POPCNT) path is unavailable.
// Encoding: 0 = todo, 1 = in_progress, 2 = done.
func countStatuses(statuses []byte) (todo, inProgress, done int) {
	for _, s := range statuses {
		switch s {
		case 0:
			todo++
		case 1:
			inProgress++
		case 2:
			done++
		}
	}
	return
}
