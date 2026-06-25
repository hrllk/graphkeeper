package graph

import (
	"sort"
	"strings"

	"hrllk/graphkeeper/internal/git"
)

type Node struct {
	Hash        string
	Parents     []string
	RelativeAge string
	Author      string
	Decorations []string
	Subject     string
}

type LaneSide string

const (
	LaneLocal  LaneSide = "local"
	LaneRemote LaneSide = "remote"
	LaneOther  LaneSide = "other"
)

type LaneRef struct {
	Hash   string
	Family string
	Side   LaneSide
}

type Row struct {
	Commit       Node
	Graph        string
	Children     []string
	Before       []LaneRef
	After        []LaneRef
	Lane         int
	DisplayWidth int
	Collapse     bool
}

func Nodes(rs git.Status) []Node {
	items := make([]Node, 0, len(rs.GraphCommits))
	for _, commit := range rs.GraphCommits {
		items = append(items, Node{
			Hash:        commit.Hash,
			Parents:     append([]string(nil), commit.Parents...),
			RelativeAge: commit.RelativeAge,
			Author:      commit.Author,
			Decorations: append([]string(nil), commit.Decorations...),
			Subject:     commit.Subject,
		})
	}
	return items
}

func Rows(rs git.Status) []Row {
	rs = injectVirtualConflictNode(rs)
	if hasGraphPrefix(rs.GraphCommits) {
		return rowsFromGitGraph(rs)
	}
	return rowsLegacy(rs)
}

func CurrentFocus(rs git.Status, cursor int) Node {
	items := Rows(rs)
	if cursor < 0 || cursor >= len(items) {
		return Node{}
	}
	return items[cursor].Commit
}

func FindRowByHash(rows []Row, hash string) int {
	if hash == "" {
		return -1
	}
	for i, row := range rows {
		if row.Commit.Hash == hash {
			return i
		}
	}
	return -1
}

func injectVirtualConflictNode(rs git.Status) git.Status {
	if !rs.MergeInProgress && !rs.RebaseInProgress {
		return rs
	}

	newCommits := make([]git.GraphCommit, 0, len(rs.GraphCommits)+1)

	vc := git.GraphCommit{
		Hash:        "VIRTUAL_CONFLICT_HASH",
		Subject:     "conflict",
		RelativeAge: "now",
		Author:      "You",
	}
	if rs.Head != "" {
		vc.Parents = append(vc.Parents, rs.Head)
	}
	if rs.ConflictTarget != "" {
		vc.Parents = append(vc.Parents, rs.ConflictTarget)
	}

	if hasGraphPrefix(rs.GraphCommits) {
		if len(rs.GraphCommits) > 0 {
			originalGraph := rs.GraphCommits[0].Graph
			vc.Graph = originalGraph

			modifiedFirst := rs.GraphCommits[0]
			modifiedFirst.Graph = strings.ReplaceAll(originalGraph, "*", "|")

			newCommits = append(newCommits, vc)
			newCommits = append(newCommits, modifiedFirst)
			if len(rs.GraphCommits) > 1 {
				newCommits = append(newCommits, rs.GraphCommits[1:]...)
			}
		} else {
			vc.Graph = "*"
			newCommits = append(newCommits, vc)
		}
	} else {
		newCommits = append(newCommits, vc)
		newCommits = append(newCommits, rs.GraphCommits...)
	}

	rs.GraphCommits = newCommits
	return rs
}

func rowsFromGitGraph(rs git.Status) []Row {
	commits := Nodes(rs)
	rows := make([]Row, 0, len(commits))
	children := buildChildrenMap(commits)
	for _, commit := range rs.GraphCommits {
		if commit.Hash == "" && commit.Subject == "" && len(commit.Parents) == 0 && len(commit.Decorations) == 0 {
			rows = append(rows, Row{
				Graph:        commit.Graph,
				DisplayWidth: max(len([]rune(commit.Graph)), 1),
			})
			continue
		}
		childRefs := append([]string(nil), children[commit.Hash]...)
		row := Row{
			Commit:       Node{Hash: commit.Hash, Parents: append([]string(nil), commit.Parents...), RelativeAge: commit.RelativeAge, Author: commit.Author, Decorations: append([]string(nil), commit.Decorations...), Subject: commit.Subject},
			Graph:        commit.Graph,
			Children:     childRefs,
			DisplayWidth: max(max(len([]rune(commit.Graph)), len(childRefs)), 1),
		}
		rows = append(rows, row)
	}
	return rows
}

func rowsLegacy(rs git.Status) []Row {
	commits := Nodes(rs)
	rows := make([]Row, 0, len(commits))
	children := buildChildrenMap(commits)
	preferred := firstParentSet(commits, rs.Head)
	active := initialGraphLanes(commits, rs)
	for _, commit := range commits {
		matches := laneMatches(active, commit.Hash)
		if len(matches) == 0 {
			fallback := LaneRef{Hash: commit.Hash, Side: LaneOther}
			active = ensureLaneSeeds(active, commit.Hash, []LaneRef{fallback}, preferred[commit.Hash], rs.Branch)
			matches = laneMatches(active, commit.Hash)
		}
		lane := chooseDisplayLane(active, matches, rs.Branch)
		before := append([]LaneRef(nil), active...)
		after := advanceGraphLanes(before, matches, commit, rs.Branch, nil, false)
		childRefs := append([]string(nil), children[commit.Hash]...)
		row := Row{
			Commit:       commit,
			Children:     childRefs,
			Before:       before,
			After:        after,
			Lane:         lane,
			DisplayWidth: max(max(max(len(before), len(after)), len(childRefs)), 1),
		}
		rows = append(rows, row)
		active = after
	}
	return rows
}

func hasGraphPrefix(commits []git.GraphCommit) bool {
	for _, commit := range commits {
		if commit.Graph != "" {
			return true
		}
	}
	return false
}

func initialGraphLanes(commits []Node, rs git.Status) []LaneRef {
	if rs.Branch == "" || rs.Head == "" {
		return make([]LaneRef, 0, 8)
	}
	remoteTip := ""
	remoteDecoration := "origin/" + rs.Branch
	headPresent := false
	for _, commit := range commits {
		if commit.Hash == rs.Head {
			headPresent = true
		}
		for _, decoration := range commit.Decorations {
			if strings.TrimSpace(decoration) == remoteDecoration {
				remoteTip = commit.Hash
			}
		}
	}
	if !headPresent {
		return make([]LaneRef, 0, 8)
	}
	lanes := []LaneRef{{Hash: rs.Head, Family: rs.Branch, Side: LaneLocal}}
	if remoteTip != "" && remoteTip != rs.Head {
		lanes = append(lanes, LaneRef{Hash: remoteTip, Family: rs.Branch, Side: LaneRemote})
	}
	return lanes
}

func buildLaneSeeds(commits []Node, rs git.Status) map[string][]LaneRef {
	localSet := make(map[string]struct{}, len(rs.LocalBranches))
	for _, branch := range rs.LocalBranches {
		localSet[branch] = struct{}{}
	}
	seeds := make(map[string][]LaneRef, len(commits))
	for _, commit := range commits {
		refs := seedLaneRefs(commit.Decorations, localSet)
		if len(refs) == 0 {
			continue
		}
		for i := range refs {
			refs[i].Hash = commit.Hash
		}
		sort.SliceStable(refs, func(i, j int) bool {
			left := laneRefScore(refs[i], rs.Branch)
			right := laneRefScore(refs[j], rs.Branch)
			if left != right {
				return left > right
			}
			leftSide := laneSidePriority(refs[i].Side)
			rightSide := laneSidePriority(refs[j].Side)
			if leftSide != rightSide {
				return leftSide < rightSide
			}
			return refs[i].Family < refs[j].Family
		})
		seeds[commit.Hash] = refs
	}
	return seeds
}

func buildFamilyPriority(commits []Node, rs git.Status) map[string]int {
	priority := make(map[string]int, len(rs.LocalBranches)+len(rs.RemoteBranches)+1)
	if rs.Branch != "" {
		priority[rs.Branch] = 0
	}
	return priority
}

func laneSeedFromDecoration(decoration string, localSet map[string]struct{}) (LaneRef, bool) {
	decoration = strings.TrimSpace(decoration)
	switch {
	case strings.HasPrefix(decoration, "HEAD -> "):
		return LaneRef{Family: strings.TrimPrefix(decoration, "HEAD -> "), Side: LaneLocal}, true
	case strings.HasPrefix(decoration, "origin/"):
		family := strings.TrimPrefix(decoration, "origin/")
		if _, ok := localSet[family]; ok {
			return LaneRef{Family: family, Side: LaneRemote}, true
		}
		return LaneRef{}, false
	case strings.HasPrefix(decoration, "tag: "), decoration == "":
		return LaneRef{}, false
	case strings.Contains(decoration, "/"):
		return LaneRef{}, false
	default:
		return LaneRef{Family: decoration, Side: LaneLocal}, true
	}
}

func seedLaneRefs(decorations []string, localSet map[string]struct{}) []LaneRef {
	refs := make([]LaneRef, 0, len(decorations))
	seen := make(map[LaneRef]struct{}, len(decorations))
	for _, decoration := range decorations {
		ref, ok := laneSeedFromDecoration(decoration, localSet)
		if !ok {
			continue
		}
		if _, exists := seen[ref]; exists {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

func distinctFamilies(refs []LaneRef) map[string]struct{} {
	families := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.Family == "" {
			continue
		}
		families[ref.Family] = struct{}{}
	}
	return families
}

func laneRefScore(ref LaneRef, currentBranch string) int {
	score := 0
	if ref.Family == currentBranch {
		score += 100
	}
	switch ref.Side {
	case LaneLocal:
		score += 10
	case LaneRemote:
		score += 5
	}
	return score
}

func buildChildrenMap(commits []Node) map[string][]string {
	children := make(map[string][]string)
	for _, commit := range commits {
		for _, parent := range commit.Parents {
			if parent == "" {
				continue
			}
			children[parent] = append(children[parent], commit.Hash)
		}
	}
	return children
}

func ensureLaneSeeds(active []LaneRef, hash string, seeds []LaneRef, preferred bool, currentBranch string) []LaneRef {
	if hash == "" || len(seeds) == 0 {
		return active
	}
	filtered := make([]LaneRef, 0, len(seeds))
	for _, seed := range seeds {
		seed.Hash = hash
		if hasLaneRef(active, seed) {
			continue
		}
		filtered = append(filtered, seed)
	}
	if len(filtered) == 0 {
		return active
	}
	if len(active) == 0 {
		return append(active, filtered...)
	}
	prepend := preferred
	if !prepend && currentBranch != "" {
		for _, seed := range filtered {
			if seed.Family == currentBranch {
				prepend = true
				break
			}
		}
	}
	if prepend {
		return append(filtered, active...)
	}
	return append(active, filtered...)
}

func hasLaneRef(active []LaneRef, target LaneRef) bool {
	for _, ref := range active {
		if ref == target {
			return true
		}
	}
	return false
}

func firstParentSet(commits []Node, head string) map[string]bool {
	if head == "" {
		return nil
	}
	byHash := make(map[string]Node, len(commits))
	for _, commit := range commits {
		byHash[commit.Hash] = commit
	}
	preferred := make(map[string]bool)
	current := head
	for current != "" {
		if preferred[current] {
			break
		}
		preferred[current] = true
		commit, ok := byHash[current]
		if !ok || len(commit.Parents) == 0 {
			break
		}
		current = commit.Parents[0]
	}
	return preferred
}

func moveSelectableGraphPointer(current int, rows []Row, delta int) int {
	if len(rows) == 0 {
		return -1
	}
	if delta == 0 {
		return nearestSelectableGraphRow(rows, current, 1)
	}
	if current < 0 || current >= len(rows) {
		if delta > 0 {
			return nearestSelectableGraphRow(rows, 0, 1)
		}
		return nearestSelectableGraphRow(rows, len(rows)-1, -1)
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	if next >= len(rows) {
		next = len(rows) - 1
	}
	return nearestSelectableGraphRow(rows, next, sign(delta))
}

func nearestSelectableGraphRow(rows []Row, start, step int) int {
	if len(rows) == 0 {
		return -1
	}
	if step == 0 {
		step = 1
	}
	if start < 0 {
		start = 0
	}
	if start >= len(rows) {
		start = len(rows) - 1
	}
	for i := start; i >= 0 && i < len(rows); i += step {
		if rows[i].Commit.Hash != "" || rows[i].Graph != "" {
			return i
		}
	}
	if step > 0 {
		for i := start - 1; i >= 0; i-- {
			if rows[i].Commit.Hash != "" || rows[i].Graph != "" {
				return i
			}
		}
	} else {
		for i := start + 1; i < len(rows); i++ {
			if rows[i].Commit.Hash != "" || rows[i].Graph != "" {
				return i
			}
		}
	}
	return start
}

func chooseDisplayLane(active []LaneRef, matches []int, currentBranch string) int {
	if len(matches) == 0 {
		return 0
	}
	best := matches[0]
	bestScore := laneRefScore(active[best], currentBranch)
	for _, idx := range matches[1:] {
		score := laneRefScore(active[idx], currentBranch)
		if score > bestScore {
			best = idx
			bestScore = score
		}
	}
	return best
}

func choosePrimaryMatch(active []LaneRef, matches []int, currentBranch string) LaneRef {
	return active[chooseDisplayLane(active, matches, currentBranch)]
}

func prioritizeLaneRefs(active []LaneRef, currentBranch string, familyPriority map[string]int) []LaneRef {
	if len(active) <= 1 || currentBranch == "" {
		return active
	}
	ordered := append([]LaneRef(nil), active...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		leftRank := lanePriorityRank(left, currentBranch, familyPriority)
		rightRank := lanePriorityRank(right, currentBranch, familyPriority)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftSide := laneSidePriority(left.Side)
		rightSide := laneSidePriority(right.Side)
		if leftSide != rightSide {
			return leftSide < rightSide
		}
		return false
	})
	return ordered
}

func lanePriorityRank(ref LaneRef, currentBranch string, familyPriority map[string]int) int {
	if ref.Family == currentBranch {
		return 0
	}
	if rank, ok := familyPriority[ref.Family]; ok {
		return rank
	}
	return 1 << 20
}

func laneSidePriority(side LaneSide) int {
	switch side {
	case LaneLocal:
		return 0
	case LaneRemote:
		return 1
	default:
		return 2
	}
}

func advanceGraphLanes(active []LaneRef, matches []int, commit Node, currentBranch string, familyPriority map[string]int, preserveMatchedLanes bool) []LaneRef {
	if len(matches) == 0 {
		return append([]LaneRef(nil), active...)
	}
	primary := choosePrimaryMatch(active, matches, currentBranch)
	next := make([]LaneRef, 0, len(active)+len(commit.Parents))
	skipped := make(map[int]struct{}, len(matches))
	for _, idx := range matches {
		skipped[idx] = struct{}{}
	}
	inserted := false
	for idx, ref := range active {
		if _, ok := skipped[idx]; !ok {
			next = append(next, ref)
			continue
		}
		if inserted {
			continue
		}
		inserted = true
		if len(commit.Parents) == 0 {
			continue
		}
		next = append(next, LaneRef{
			Hash:   commit.Parents[0],
			Family: primary.Family,
			Side:   primary.Side,
		})
		for _, parent := range commit.Parents[1:] {
			if parent == "" {
				continue
			}
			next = append(next, LaneRef{Hash: parent, Side: LaneOther})
		}
	}
	return prioritizeLaneRefs(compactLaneRefs(next), currentBranch, familyPriority)
}

func compactLaneRefs(active []LaneRef) []LaneRef {
	if len(active) <= 1 {
		return active
	}
	seen := make(map[LaneRef]struct{}, len(active))
	compacted := make([]LaneRef, 0, len(active))
	for _, ref := range active {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		compacted = append(compacted, ref)
	}
	return compacted
}

func ensureLaneSeed(active []LaneRef, hash string, seed LaneRef, preferred bool, currentBranch string) []LaneRef {
	if hash == "" {
		return active
	}
	if idx := lastIndexOf(active, hash); idx >= 0 {
		return active
	}
	seed.Hash = hash
	switch {
	case len(active) == 0:
		return append(active, seed)
	case preferred || seed.Family == currentBranch:
		return append([]LaneRef{seed}, active...)
	default:
		return append(active, seed)
	}
}

func lastIndexOf(values []LaneRef, target string) int {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i].Hash == target {
			return i
		}
	}
	return -1
}

func laneMatches(active []LaneRef, hash string) []int {
	matches := make([]int, 0, 2)
	for i, ref := range active {
		if ref.Hash == hash {
			matches = append(matches, i)
		}
	}
	return matches
}
