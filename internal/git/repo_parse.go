package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func parseBranchMetadataLine(line string) (branchName string, upstream string, tracking BranchTracking, ok bool) {
	parts := strings.Split(strings.TrimSpace(line), "|")
	if len(parts) != 3 {
		return "", "", BranchTracking{}, false
	}
	branchName = strings.TrimSpace(parts[0])
	if branchName == "" {
		return "", "", BranchTracking{}, false
	}
	upstream = strings.TrimSpace(parts[1])
	if upstream != "" && strings.TrimSpace(parts[2]) == "" {
		return "", "", BranchTracking{}, false
	}
	trackingInfo, trackingOK := parseTrackingInfoStrict(parts[2])
	if !trackingOK {
		return "", "", BranchTracking{}, false
	}
	tracking = trackingInfo
	if strings.TrimSpace(parts[2]) == "[gone]" {
		upstream = ""
	}
	return branchName, upstream, tracking, true
}

func parseBranchUpstreamLine(line string) (branchName string, upstream string, ok bool) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) == 0 {
		return "", "", false
	}
	branchName = strings.TrimSpace(parts[0])
	if branchName == "" {
		return "", "", false
	}
	if len(parts) > 1 {
		upstream = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 && strings.Contains(parts[2], "gone") {
		upstream = ""
	}
	return branchName, upstream, true
}

func parseTrackingInfo(track string) (ahead, behind int) {
	tracking, _ := parseTrackingInfoStrict(track)
	return tracking.Ahead, tracking.Behind
}

func parseTrackingInfoStrict(track string) (tracking BranchTracking, ok bool) {
	track = strings.TrimSpace(track)
	if track == "" {
		return BranchTracking{}, true
	}
	if len(track) < 2 || track[0] != '[' || track[len(track)-1] != ']' || strings.ContainsAny(track[1:len(track)-1], "[]") {
		return BranchTracking{}, false
	}
	track = track[1 : len(track)-1]
	if strings.TrimSpace(track) == "" {
		return BranchTracking{}, false
	}
	parts := strings.Split(track, ",")
	seenAhead, seenBehind := false, false
	for _, part := range parts {
		fields := strings.Fields(part)
		if len(fields) == 1 && fields[0] == "gone" {
			if len(parts) != 1 {
				return BranchTracking{}, false
			}
			continue
		}
		if len(fields) != 2 || (fields[0] != "ahead" && fields[0] != "behind") {
			return BranchTracking{}, false
		}
		count, err := strconv.Atoi(fields[1])
		if err != nil || count < 0 {
			return BranchTracking{}, false
		}
		if fields[0] == "ahead" {
			if seenAhead {
				return BranchTracking{}, false
			}
			seenAhead = true
			tracking.Ahead = count
		} else {
			if seenBehind {
				return BranchTracking{}, false
			}
			seenBehind = true
			tracking.Behind = count
		}
	}
	return tracking, true
}

func (r *Repo) graphCommits(ctx context.Context, localBranches []string, branchUpstreams map[string]string, limit int) ([]GraphCommit, error) {
	graphRefs := graphRefs(localBranches, branchUpstreams)
	if len(graphRefs) == 0 {
		return nil, nil
	}
	args := graphLogArgs(graphRefs, limit)
	lines, err := r.gitRawLines(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseGraphCommitLines(lines), nil
}

func parseGraphCommitLines(lines []string) []GraphCommit {
	commits := make([]GraphCommit, 0, len(lines))
	for _, line := range lines {
		nul := strings.IndexRune(line, '\x00')
		if nul < 0 {
			graph := strings.TrimRight(line, "\r\n")
			if strings.TrimSpace(graph) != "" {
				commits = append(commits, GraphCommit{Graph: graph})
			}
			continue
		}
		graph := line[:nul]
		parts := strings.SplitN(line[nul+1:], "\x1f", 6)
		if len(parts) < 6 {
			continue
		}
		entry := GraphCommit{Graph: graph, Hash: strings.TrimSpace(parts[0])}
		if parents := strings.TrimSpace(parts[1]); parents != "" {
			entry.Parents = strings.Fields(parents)
		}
		if relativeAge := strings.TrimSpace(parts[2]); relativeAge != "" {
			entry.RelativeAge = relativeAge
		}
		if author := strings.TrimSpace(parts[3]); author != "" {
			entry.Author = author
		}
		if decorations := strings.TrimSpace(parts[4]); decorations != "" {
			entry.Decorations = splitDecorations(decorations)
		}
		entry.Subject = strings.TrimSpace(parts[5])
		if entry.Hash != "" {
			commits = append(commits, entry)
		}
	}
	return commits
}

func graphRefs(localBranches []string, branchUpstreams map[string]string) []string {
	refs := make([]string, 0, len(localBranches)+len(branchUpstreams)+1)
	seen := make(map[string]struct{}, len(localBranches)+len(branchUpstreams)+1)
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, ok := seen[ref]; ok {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	for _, branch := range localBranches {
		add(branch)
		if upstream := branchUpstreams[branch]; upstream != "" {
			add(upstream)
		}
	}
	add("HEAD")
	return refs
}

func graphLogArgs(refs []string, limit int) []string {
	args := []string{
		"log",
		"--graph",
		"--decorate=short",
		"--decorate-refs=HEAD",
		"--decorate-refs=refs/heads/*",
		"--decorate-refs=refs/remotes/*",
		"--topo-order",
		"--format=%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s",
	}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	args = append(args, refs...)
	return args
}

func filterRemoteBranches(remoteBranches []string) []string {
	filtered := make([]string, 0, len(remoteBranches))
	for _, branch := range remoteBranches {
		if strings.HasSuffix(branch, "/HEAD") {
			continue
		}
		filtered = append(filtered, branch)
	}
	return filtered
}
