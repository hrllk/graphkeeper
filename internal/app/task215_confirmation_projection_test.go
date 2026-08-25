package app

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/state"
)

func TestTask215BinaryProjectionRendererClassifierAgreement(t *testing.T) {
	m := model{status: state.New().WithConfirm(state.ActionStash, "Stash changes?", "Continue with the local cleanup?")}
	view := confirmView(m)
	want := map[string]confirmResult{
		"y":     {Decision: decisionAccept},
		"enter": {Decision: decisionAccept},
		"n":     {Decision: decisionCancel},
		"esc":   {Decision: decisionCancel},
		"x":     {Decision: decisionNoop},
	}
	popup := ansi.Strip(renderConfirmPopup(m, 80))
	if !strings.Contains(popup, "Continue with the local cleanup?") || !strings.Contains(popup, "y: stash  •  n: cancel") {
		t.Fatalf("binary renderer lost independent projection values: %q", popup)
	}
	for key, expected := range want {
		if got := classifyConfirmKey(view, key); got != expected {
			t.Fatalf("binary key %q: got %#v, want %#v", key, got, expected)
		}
		before := m
		gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		got := gotModel.(model)
		if key == "y" || key == "enter" {
			if cmd == nil || got.status.Mode != state.ModeLoading {
				t.Fatalf("binary affirmative %q did not reach existing command owner: status=%+v cmd=%v", key, got.status, cmd != nil)
			}
		} else if cmd != nil {
			t.Fatalf("binary non-affirmative %q returned command", key)
		}
		if key == "x" && !reflect.DeepEqual(got, before) {
			t.Fatalf("unknown binary key mutated model: got=%#v before=%#v", got, before)
		}
	}
}

func TestTask215DivergentProjectionRendererClassifierAgreement(t *testing.T) {
	view, ok := projectMergeConfirm(PullImpactSnapshot{CurrentRef: "main", UpstreamRef: "origin/main", Ahead: 2, Behind: 3}, PullImpactSet{
		Valid: true, MergeSummary: "merge body", RebaseSummary: "rebase body", MergeRisk: "merge risk", RebaseRisk: "rebase risk",
	}, false)
	if !ok {
		t.Fatal("valid divergent impact was rejected")
	}
	m := model{
		overlayState: overlayState{mergeConfirmView: &view},
		pullState:    pullState{pullIsFastForward: false},
		status:       state.New().WithConfirm(state.ActionPull, "Pull into main?", mergeConfirmBody(view, 72)),
	}
	popup := ansi.Strip(renderConfirmPopup(m, 80))
	for _, want := range []string{"Pull into main?", "merge body", "rebase body", "m: merge", "r: rebase", "esc: cancel"} {
		if !strings.Contains(popup, want) {
			t.Fatalf("divergent renderer missing %q: %q", want, popup)
		}
	}
	for key, expected := range map[string]confirmResult{
		"m":   {Decision: decisionChoice, ChoiceKey: choiceMerge},
		"r":   {Decision: decisionChoice, ChoiceKey: choiceRebase},
		"esc": {Decision: decisionCancel},
		"x":   {Decision: decisionNoop},
	} {
		if got := classifyConfirmKey(confirmView(m), key); got != expected {
			t.Fatalf("divergent key %q: got %#v, want %#v", key, got, expected)
		}
	}
}

func TestTask215DivergentRendererUsesProjectionOwnedCopy(t *testing.T) {
	base := confirmationProjection{
		Kind:          confirmChoiceKind,
		Title:         "Projection title",
		Detail:        "Projection detail",
		FooterText:    "Projection footer",
		CurrentBranch: "main",
		TargetRef:     "origin/main",
		CurrentOnly:   2,
		TargetOnly:    3,
		MergeText:     "merge summary",
		RebaseText:    "rebase summary",
		RiskText:      "risk summary",
	}
	for field, mutate := range map[string]func(*confirmationProjection){
		"title":  func(p *confirmationProjection) { p.Title = "Changed title" },
		"detail": func(p *confirmationProjection) { p.Detail = "Changed detail" },
		"footer": func(p *confirmationProjection) { p.FooterText = "Changed footer" },
	} {
		p := base
		mutate(&p)
		rendered := ansi.Strip(renderDivergentConfirmPopup(p, 80))
		want := map[string]string{"title": "Changed title", "detail": "Changed detail", "footer": "Changed footer"}[field]
		if !strings.Contains(rendered, want) {
			t.Fatalf("changed projection %s was not rendered: %q", field, rendered)
		}
	}

	stale := base
	stale.Disabled = true
	stale.DisabledText = "Changed stale detail"
	stale.FooterText = "stale footer must not approve"
	rendered := ansi.Strip(renderDivergentConfirmPopup(stale, 80))
	if !strings.Contains(rendered, "Changed stale detail") || !strings.Contains(rendered, "n: close") {
		t.Fatalf("stale projection did not preserve close-only copy/footer: %q", rendered)
	}
	if strings.Contains(rendered, "stale footer must not approve") || strings.Contains(rendered, "m: merge") || strings.Contains(rendered, "r: rebase") {
		t.Fatalf("stale renderer exposed affirmative footer: %q", rendered)
	}
}

func TestTask215FastForwardIsFOnlyAndHidesChoices(t *testing.T) {
	m := model{pullState: pullState{pullIsFastForward: true}, status: state.New().WithConfirm(state.ActionPull, "Fast-forward available.", "Fast-forward to the target commit.")}
	popup := ansi.Strip(renderConfirmPopup(m, 80))
	if !strings.Contains(popup, "f: fast-forward") || strings.Contains(popup, "enter: fast-forward") || strings.Contains(popup, "q: close") || strings.Contains(popup, "merge") || strings.Contains(popup, "rebase") {
		t.Fatalf("fast-forward footer contract mismatch: %q", popup)
	}
	for key, want := range map[string]confirmDecision{"f": decisionAccept, "enter": decisionNoop, "n": decisionNoop, "q": decisionNoop, "esc": decisionCancel, "m": decisionNoop, "r": decisionNoop} {
		if got := classifyConfirmKey(confirmView(m), key); got.Decision != want {
			t.Fatalf("fast-forward key %q: got %q, want %q", key, got.Decision, want)
		}
	}
}

func TestTask215StaleProjectionIsCloseOnlyAndCommandFree(t *testing.T) {
	view := mergeConfirmViewModel{CurrentBranch: "main", TargetRef: "origin/main", CurrentOnly: 2, TargetOnly: 3, ImpactKnown: true, MergeText: "old merge", RebaseText: "old rebase", Disabled: true, DisabledText: "Preview is stale. Refresh before continuing."}
	m := model{overlayState: overlayState{mergeConfirmView: &view}, pullState: pullState{pullConfirmStale: true}, status: state.New().WithConfirm(state.ActionPull, "Pull into main?", "old body")}
	popup := ansi.Strip(renderConfirmPopup(m, 80))
	if !strings.Contains(popup, "stale") || !strings.Contains(popup, "n: close") || strings.Contains(popup, "m/enter: merge") || strings.Contains(popup, "r: rebase") {
		t.Fatalf("stale renderer did not expose close-only copy/footer: %q", popup)
	}
	for _, key := range []string{"m", "r", "enter", "y", "x"} {
		if got := classifyConfirmKey(confirmView(m), key); got.Decision != decisionNoop {
			t.Fatalf("stale affirmative/unknown key %q was not a no-op: %#v", key, got)
		}
		gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if cmd != nil || !reflect.DeepEqual(gotModel.(model), m) {
			t.Fatalf("stale key %q mutated or created command: got=%#v cmd=%v", key, gotModel, cmd != nil)
		}
	}
	for _, key := range []string{"n", "esc"} {
		if got := classifyConfirmKey(confirmView(m), key); got.Decision != decisionCancel {
			t.Fatalf("stale close key %q: %#v", key, got)
		}
	}
}

func TestTask215ConfirmWidthsAreDeterministicAt40_60_80(t *testing.T) {
	view := mergeConfirmViewModel{CurrentBranch: "main", TargetRef: "origin/main", CurrentOnly: 12, TargetOnly: 34, ImpactKnown: true, MergeText: "Histories will be combined and a merge commit may be created.", RebaseText: "Local commits will be replayed onto origin/main.", RiskText: "Conflicts may occur."}
	for _, width := range []int{40, 60, 80} {
		first := ansi.Strip(renderMergeConfirmPopup(view, width))
		second := ansi.Strip(renderMergeConfirmPopup(view, width))
		if first != second {
			t.Fatalf("width %d fitting is nondeterministic", width)
		}
		if !strings.Contains(first, "merge") || !strings.Contains(first, "rebase") || !strings.Contains(first, "cancel") {
			t.Fatalf("width %d lost required footer content: %q", width, first)
		}
		for _, line := range strings.Split(first, "\n") {
			if len([]rune(line)) > width+8 { // popup chrome is outside the body-width fixture.
				t.Fatalf("width %d produced unbounded line %q", width, line)
			}
		}
	}
}

func TestTask215ProjectionReplacementDoesNotReuseOldMergeContent(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "old/target", CurrentOnly: 9, TargetOnly: 9, ImpactKnown: true, MergeText: "OLD MERGE", RebaseText: "OLD REBASE"}
	m := model{overlayState: overlayState{mergeConfirmView: &old}, status: state.New().WithConfirm(state.ActionPull, "old", "old")}
	fresh := PullImpactSnapshot{CurrentRef: "new", UpstreamRef: "new/target", Ahead: 1, Behind: 2}
	freshSet := PullImpactSet{Valid: true, MergeSummary: "NEW MERGE", RebaseSummary: "NEW REBASE"}
	if !applyMergeConfirmProjection(&m, fresh, freshSet, false) {
		t.Fatal("fresh projection was rejected")
	}
	popup := ansi.Strip(renderConfirmPopup(m, 80))
	if strings.Contains(popup, "OLD MERGE") || strings.Contains(popup, "OLD REBASE") || !strings.Contains(popup, "NEW MERGE") || !strings.Contains(popup, "NEW REBASE") {
		t.Fatalf("replacement retained stale content: %q", popup)
	}
}

func TestTask215InvalidPullImpactHasNoNormalProjection(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "origin/old", CurrentOnly: 1, TargetOnly: 2, ImpactKnown: true, MergeText: "OLD MERGE", RebaseText: "OLD REBASE"}
	m := model{overlayState: overlayState{mergeConfirmView: &old}, status: state.New().WithConfirm(state.ActionPull, "old", "old")}
	if ok := applyMergeConfirmProjection(&m, PullImpactSnapshot{CurrentRef: "main", UpstreamRef: "origin/main", Ahead: -1, Behind: 2}, PullImpactSet{Valid: true}, false); ok {
		t.Fatal("invalid impact produced a normal projection")
	}
	if rendered := ansi.Strip(renderConfirmPopup(m, 80)); strings.Contains(rendered, "OLD MERGE") || strings.Contains(rendered, "OLD REBASE") {
		t.Fatalf("invalid impact reused prior divergent content: %q", rendered)
	}
}

func TestTask215PullWorkflowCompletionClearsProjection(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "origin/old", ImpactKnown: true, MergeText: "OLD MERGE", RebaseText: "OLD REBASE"}
	m := model{
		overlayState: overlayState{mergeConfirmView: &old},
		pullState:    pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}},
		status:       state.New().WithConfirm(state.ActionPull, "old", "old"),
	}
	m.pullConfirmInput = &pullConfirmInput{MergeText: "OLD MERGE"}
	next, cmd := m.Update(pullWorkflowMsg{result: PullWorkflowResult{
		OperationRequestID: 7, OperationEpoch: 3, RefreshRequestID: 8, RefreshEpoch: 4,
		Execute: PullExecutionResult{Mode: PullModeMerge, Succeeded: true},
		Refresh: ReadSnapshotResult{RequestID: 8, RepositoryEpoch: 4},
	}})
	if cmd != nil {
		t.Fatalf("workflow emitted command: %v", cmd)
	}
	got := next.(model)
	if got.pullConfirmInput != nil || got.mergeConfirmView != nil {
		t.Fatalf("successful workflow retained confirmation projection: input=%#v view=%#v", got.pullConfirmInput, got.mergeConfirmView)
	}
}

func TestTask215PullWorkflowRefreshIdentityIgnoredClearsProjection(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "origin/old", ImpactKnown: true, MergeText: "OLD MERGE"}
	m := model{
		overlayState: overlayState{mergeConfirmView: &old},
		pullState:    pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}},
	}
	m.pullConfirmInput = &pullConfirmInput{MergeText: "OLD MERGE"}
	next, cmd := m.Update(pullWorkflowMsg{result: PullWorkflowResult{
		OperationRequestID: 7, OperationEpoch: 3, RefreshRequestID: 8, RefreshEpoch: 4,
		Execute: PullExecutionResult{Mode: PullModeMerge, Succeeded: true},
		Refresh: ReadSnapshotResult{RequestID: 99, RepositoryEpoch: 4},
	}})
	if cmd != nil {
		t.Fatalf("identity-ignored workflow emitted command: %v", cmd)
	}
	got := next.(model)
	if got.pullConfirmInput != nil || got.mergeConfirmView != nil {
		t.Fatalf("identity-ignored workflow retained confirmation projection: input=%#v view=%#v", got.pullConfirmInput, got.mergeConfirmView)
	}
}

func TestTask215PullToastDoneClearsProjection(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "origin/old", ImpactKnown: true, MergeText: "OLD MERGE"}
	m := model{
		overlayState: overlayState{mergeConfirmView: &old},
		pullState:    pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}},
	}
	m.pullConfirmInput = &pullConfirmInput{MergeText: "OLD MERGE"}
	next, cmd := m.Update(pullToastDoneMsg{requestID: 7, requestEpoch: 3})
	if cmd != nil {
		t.Fatalf("toast emitted command: %v", cmd)
	}
	got := next.(model)
	if got.pullConfirmInput != nil || got.mergeConfirmView != nil {
		t.Fatalf("toast retained confirmation projection: input=%#v view=%#v", got.pullConfirmInput, got.mergeConfirmView)
	}
}

func TestTask215PullExecutionTransitionClearsProjection(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "origin/old", ImpactKnown: true, MergeText: "OLD MERGE"}
	m := model{
		overlayState: overlayState{mergeConfirmView: &old},
		pullState:    pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}},
		status:       state.New().WithConfirm(state.ActionPull, "Pull", "Continue"),
	}
	m.pullConfirmInput = &pullConfirmInput{MergeText: "OLD MERGE"}
	m.status.CanExecute = true
	next, _ := m.handleOutcomePreviewExecute()
	got := next.(model)
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("pull execution did not enter loading: %+v", got.status)
	}
	if got.pullConfirmInput != nil || got.mergeConfirmView != nil {
		t.Fatalf("execution transition retained confirmation projection: input=%#v view=%#v", got.pullConfirmInput, got.mergeConfirmView)
	}
}

func TestTask215TopLevelPullCommandPayloadIdentity(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head", Upstream: "origin/main", UpstreamOID: "upstream", Ahead: 1, Behind: 2, TrackingKnown: true, TrackingFresh: true}
	for _, tc := range []struct {
		key  string
		mode PullMode
	}{
		{key: "m", mode: PullModeMerge},
		{key: "r", mode: PullModeRebase},
	} {
		t.Run(tc.key, func(t *testing.T) {
			pull := &compositionPullFake{validation: PullValidationResult{Valid: true, Authorized: true, AuthorizedBaseline: baseline}, execution: PullExecutionResult{Succeeded: true}}
			readResult := ReadSnapshotResult{ErrorKind: ReadErrorNone, Snapshot: ReadSnapshot{Freshness: baseline}}
			read := &compositionReadFake{results: []ReadSnapshotResult{readResult, readResult}}
			m := model{
				repositoryState: repositoryState{repositoryEpoch: 3},
				pullState:       pullState{pullIsFastForward: false, activePullRequest: &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true}},
				overlayState:    overlayState{mergeConfirmView: &mergeConfirmViewModel{CurrentBranch: "main", TargetRef: "origin/main", CurrentOnly: 1, TargetOnly: 2, ImpactKnown: true, MergeText: "merge", RebaseText: "rebase"}},
				pull:            pull, repositoryRead: read,
				status: state.New().WithConfirm(state.ActionPull, "Pull into main?", "review"),
			}
			gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			if cmd == nil {
				t.Fatal("affirmative pull key returned no command")
			}
			if _, ok := cmd().(pullWorkflowMsg); !ok {
				t.Fatalf("pull command returned unexpected message")
			}
			got := gotModel.(model)
			if len(pull.executes) != 1 {
				t.Fatalf("execute count=%d, want 1", len(pull.executes))
			}
			req := pull.executes[0]
			if req.Mode != tc.mode || req.RequestID != 1 || req.RepositoryEpoch != 3 || !req.Authorized || req.AuthorizedBaseline != baseline {
				t.Fatalf("command payload changed: %#v", req)
			}
			if got.status.Action != state.ActionMerge && tc.mode == PullModeMerge && tc.key == "m" {
				// The existing handler intentionally uses pull progress copy; payload ownership is the assertion here.
				t.Fatal("merge choice did not enter the existing merge-owned transition")
			}
		})
	}
	m := model{pullState: pullState{pullIsFastForward: false, activePullRequest: &pullRequest{ID: 7}}, status: state.New().WithConfirm(state.ActionPull, "Pull", "review")}
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || got.(model).activePullRequest != nil {
		t.Fatalf("cancel unexpectedly created command or preserved active request")
	}
}

func TestTask215MutationOwnership(t *testing.T) {
	view, ok := projectMergeConfirm(PullImpactSnapshot{CurrentRef: "main", UpstreamRef: "origin/main", Ahead: 1, Behind: 2}, PullImpactSet{Valid: true, MergeSummary: "merge", RebaseSummary: "rebase"}, false)
	if !ok {
		t.Fatal("valid impact rejected")
	}
	before := view
	for _, key := range []string{"m", "enter", "r", "n", "esc", "x"} {
		_ = classifyConfirmKey(confirmViewModel{Kind: confirmChoiceKind, Action: state.ActionPull}, key)
	}
	if !reflect.DeepEqual(view, before) {
		t.Fatal("projection/classifier layer mutated its input")
	}
}

func TestTask215ClearPullConfirmProjectionRemovesRenderedAndClassifiedContent(t *testing.T) {
	old := mergeConfirmViewModel{CurrentBranch: "old", TargetRef: "origin/old", CurrentOnly: 1, TargetOnly: 2, ImpactKnown: true, MergeText: "OLD MERGE", RebaseText: "OLD REBASE"}
	m := model{
		overlayState: overlayState{mergeConfirmView: &old},
		pullState:    pullState{pullConfirmStale: false, pullIsFastForward: false},
		status:       state.New().WithConfirm(state.ActionPull, "old", "old"),
	}
	m.pullConfirmInput = &pullConfirmInput{CurrentBranch: "old", TargetRef: "origin/old", ImpactKnown: true, MergeText: "OLD MERGE", RebaseText: "OLD REBASE"}
	clearPullConfirmProjection(&m)
	if m.pullConfirmInput != nil || m.mergeConfirmView != nil {
		t.Fatalf("clear left pull projection installed: input=%#v view=%#v", m.pullConfirmInput, m.mergeConfirmView)
	}
	got := confirmView(m)
	if !reflect.DeepEqual(got, confirmViewModel{}) {
		t.Fatalf("cleared confirmation unexpectedly classified as %#v", got)
	}
	if rendered := ansi.Strip(renderConfirmPopup(m, 80)); strings.Contains(rendered, "OLD MERGE") || strings.Contains(rendered, "OLD REBASE") {
		t.Fatalf("cleared confirmation rendered old content: %q", rendered)
	}
}

func TestTask215ConfirmCancelClearsPullProjection(t *testing.T) {
	view := mergeConfirmViewModel{CurrentBranch: "main", TargetRef: "origin/main", CurrentOnly: 1, TargetOnly: 2, ImpactKnown: true, MergeText: "merge", RebaseText: "rebase"}
	m := model{
		overlayState: overlayState{mergeConfirmView: &view},
		pullState:    pullState{pullConfirmStale: false, pullIsFastForward: false, activePullRequest: &pullRequest{ID: 1}},
		status:       state.New().WithConfirm(state.ActionPull, "Pull", "review"),
	}
	m.pullConfirmInput = &pullConfirmInput{CurrentBranch: "main", TargetRef: "origin/main", ImpactKnown: true, MergeText: "merge", RebaseText: "rebase"}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil || got.pullConfirmInput != nil || got.mergeConfirmView != nil {
		t.Fatalf("cancel retained pull projection: input=%#v view=%#v cmd=%v", got.pullConfirmInput, got.mergeConfirmView, cmd != nil)
	}
}
