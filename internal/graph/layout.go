package graph

func RowWidth(row Row) int {
	if row.Graph != "" {
		return max(1, len([]rune(row.Graph)))
	}
	if shouldCollapseRowDisplay(row) {
		return 1
	}
	if row.DisplayWidth > 0 {
		return row.DisplayWidth
	}
	width := len(row.Before)
	if len(row.After) > width {
		width = len(row.After)
	}
	if width == 0 {
		width = 1
	}
	return width
}

func PageSize(height int) int {
	if height <= 12 {
		return 3
	}
	totalHeight := int(float64(height) * 0.76)
	if totalHeight < 18 {
		totalHeight = 18
	}
	if height > 0 && totalHeight > height-2 {
		totalHeight = height - 2
	}
	top := totalHeight / 8
	if top < 1 {
		top = 1
	}
	bottom := totalHeight - top
	if bottom < 1 {
		bottom = 1
	}
	size := bottom - 2
	if size < 3 {
		size = 3
	}
	return size
}

func PointerLane(row Row) int {
	if row.Graph != "" {
		if idx := indexRune(row.Graph, '*'); idx >= 0 {
			return idx
		}
		if len([]rune(row.Graph)) == 0 {
			return 0
		}
		return 0
	}
	width := rowWidth(row)
	if width <= 1 {
		return 0
	}
	if row.Lane >= 0 && row.Lane < width {
		return row.Lane
	}
	if row.Lane < 0 {
		return 0
	}
	return width - 1
}

func rowWidth(row Row) int {
	if row.Graph != "" {
		return max(1, len([]rune(row.Graph)))
	}
	if row.DisplayWidth > 0 {
		return row.DisplayWidth
	}
	width := len(row.Before)
	if len(row.After) > width {
		width = len(row.After)
	}
	if width == 0 {
		width = 1
	}
	return width
}

func shouldCollapseRowDisplay(row Row) bool {
	if len(row.Before) <= 1 {
		return false
	}
	for _, ref := range row.Before {
		if ref.Hash != row.Commit.Hash {
			return false
		}
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func indexRune(s string, target rune) int {
	for i, r := range s {
		if r == target {
			return i
		}
	}
	return -1
}
