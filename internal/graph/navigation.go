package graph

func MoveSelectableGraphPointer(current int, rows []Row, delta int) int {
	if len(rows) == 0 {
		return -1
	}
	if current < 0 || current >= len(rows) {
		current = 0
	}
	if delta == 0 {
		if rows[current].Commit.Hash != "" {
			return current
		}
		return NearestSelectableGraphRow(rows, current, 1)
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	remaining := delta
	if remaining < 0 {
		remaining = -remaining
	}
	idx := current
	for remaining > 0 {
		idx += step
		for idx >= 0 && idx < len(rows) && rows[idx].Commit.Hash == "" {
			idx += step
		}
		if idx < 0 {
			return 0
		}
		if idx >= len(rows) {
			return len(rows) - 1
		}
		remaining--
	}
	return NearestSelectableGraphRow(rows, idx, step)
}

func NearestSelectableGraphRow(rows []Row, start, step int) int {
	if len(rows) == 0 {
		return -1
	}
	if start < 0 {
		start = 0
	}
	if start >= len(rows) {
		start = len(rows) - 1
	}
	if rows[start].Commit.Hash != "" {
		return start
	}
	if step == 0 {
		step = 1
	}
	for i := start + step; i >= 0 && i < len(rows); i += step {
		if rows[i].Commit.Hash != "" {
			return i
		}
	}
	for i := start - step; i >= 0 && i < len(rows); i -= step {
		if rows[i].Commit.Hash != "" {
			return i
		}
	}
	return start
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	default:
		return 0
	}
}
