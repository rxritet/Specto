package service

import "github.com/rxritet/Specto/internal/domain"

// TaskStats holds aggregate counts and percentages for a set of tasks.
type TaskStats struct {
	Total       int     `json:"total"`
	TodoCount   int     `json:"todo_count"`
	InProgCount int     `json:"in_progress_count"`
	DoneCount   int     `json:"done_count"`
	TodoPct     float64 `json:"todo_pct"`
	InProgPct   float64 `json:"in_progress_pct"`
	DonePct     float64 `json:"done_pct"`
}

// StatsByUser computes task statistics for a given user.
// The heavy counting loop is SIMD-accelerated on amd64 (SSE2 + POPCNT)
// and falls back to plain Go on other architectures.
func (s *TaskService) StatsByUser(userID int64) (*TaskStats, error) {
	tasks, err := s.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	return computeStats(tasks), nil
}

// computeStats encodes task statuses as a byte vector and delegates
// the counting to the architecture-specific countStatuses function.
func computeStats(tasks []domain.Task) *TaskStats {
	statuses := encodeStatuses(tasks)
	todo, inProg, done := countStatuses(statuses)

	total := len(tasks)
	st := &TaskStats{
		Total:       total,
		TodoCount:   todo,
		InProgCount: inProg,
		DoneCount:   done,
	}

	if total > 0 {
		ft := float64(total)
		st.TodoPct = float64(todo) / ft * 100
		st.InProgPct = float64(inProg) / ft * 100
		st.DonePct = float64(done) / ft * 100
	}

	return st
}

// encodeStatuses maps each task's status to a single byte:
//
//	0 = todo, 1 = in_progress, 2 = done.
//
// The resulting dense byte slice is the input for the SIMD counting path.
func encodeStatuses(tasks []domain.Task) []byte {
	buf := make([]byte, len(tasks))
	for i := range tasks {
		switch tasks[i].Status {
		case domain.TaskStatusTodo:
			buf[i] = 0
		case domain.TaskStatusInProgress:
			buf[i] = 1
		case domain.TaskStatusDone:
			buf[i] = 2
		}
	}
	return buf
}
