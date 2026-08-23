package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestClassifyGitError(t *testing.T) {
	wrappedPermission := fmt.Errorf("push failed: %w", errors.New("Permission denied (publickey)"))
	cases := []struct {
		name string
		err  error
		want GitErrorCategory
	}{
		{"permission", errors.New("Permission denied (publickey)"), PermissionDenied},
		{"authentication", errors.New("Authentication failed for 'https://example.test/repo.git'"), PermissionDenied},
		{"non-fast-forward", errors.New("! [rejected] main -> main (non-fast-forward)"), NonFastForward},
		{"not-found", errors.New("remote ref not found"), NotFound},
		{"not-found-precedes-dirty-and-conflict", errors.New("remote ref not found; local changes would be overwritten by merge; merge conflict"), NotFound},
		{"dirty-worktree", errors.New("local changes would be overwritten by checkout"), DirtyWorktree},
		{"conflict", errors.New("merge conflict: CONFLICT (content)"), Conflict},
		{"unknown", errors.New("git returned an unrecognized diagnostic"), Unknown},
		{"wrapped", wrappedPermission, PermissionDenied},
		{"empty", errors.New(""), Unknown},
		{"nil", nil, Unknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyGitError(tc.err); got != tc.want {
				t.Fatalf("classifyGitError(%q) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestExecutePush(t *testing.T) {
	t.Run("ordinary push uses exact non-force non-upstream invocation and constructs success", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "config", "push.default", "current")
		runGit(t, fixture.root, "branch", "--unset-upstream")
		localHead := makeLocalCommit(t, fixture.root, "file.txt", "ordinary\n", "ordinary push")
		got := cmdResult(t, executePush(fixture.repo, "main", 40)).(executedMsg)
		if got.action != state.ActionPush || got.target != "main" || got.err != nil {
			t.Fatalf("ordinary push result = %+v", got)
		}
		if got.errorCategory != "" || got.operationErr != nil || got.statusErr != nil {
			t.Fatalf("successful ordinary push carried diagnostics/category: %+v", got)
		}
		if remoteHead := runGit(t, fixture.root, "ls-remote", fixture.remote, "refs/heads/main"); !strings.HasPrefix(remoteHead, localHead+"	") {
			t.Fatalf("ordinary push remote ref = %q, want %s", remoteHead, localHead)
		}
		if got.status.Branch != "main" || got.status.Upstream != "" {
			t.Fatalf("ordinary push changed upstream unexpectedly: %+v", got.status)
		}
	})

	t.Run("force push uses exact force invocation and constructs success", func(t *testing.T) {
		fixture := newCommandRepo(t)
		advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
		localHead := makeLocalCommit(t, fixture.root, "file.txt", "force\n", "force push")
		got := cmdResult(t, executeForcePush(fixture.repo, "main", 40)).(executedMsg)
		if got.action != state.ActionForcePush || got.target != "main" || got.err != nil {
			t.Fatalf("force push result = %+v", got)
		}
		if remoteHead := runGit(t, fixture.root, "ls-remote", fixture.remote, "refs/heads/main"); !strings.HasPrefix(remoteHead, localHead+"	") {
			t.Fatalf("force push remote ref = %q, want %s", remoteHead, localHead)
		}
		if got.status.Branch != "main" {
			t.Fatalf("force push changed branch unexpectedly: %+v", got.status)
		}
	})

	t.Run("set-upstream uses exact non-force upstream invocation and constructs success", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		localHead := makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature push")
		got := cmdResult(t, executePushSetUpstream(fixture.repo, "feature", 40)).(executedMsg)
		if got.action != state.ActionSetUpstream || got.target != "feature" || got.err != nil {
			t.Fatalf("set-upstream result = %+v", got)
		}
		if got.status.Upstream != "origin/feature" {
			t.Fatalf("set-upstream status = %+v", got.status)
		}
		if remoteHead := runGit(t, fixture.root, "ls-remote", fixture.remote, "refs/heads/feature"); !strings.HasPrefix(remoteHead, localHead+"	") {
			t.Fatalf("set-upstream remote ref = %q, want %s", remoteHead, localHead)
		}
		if got.status.Branch != "feature" {
			t.Fatalf("set-upstream changed branch unexpectedly: %+v", got.status)
		}
	})

	t.Run("operation error preserves raw error and category", func(t *testing.T) {
		fixture := newCommandRepo(t)
		advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
		makeLocalCommit(t, fixture.root, "file.txt", "rejected\n", "rejected push")
		got := cmdResult(t, executePush(fixture.repo, "main", 40)).(executedMsg)
		if got.err == nil || got.operationErr == nil || got.err != got.operationErr {
			t.Fatalf("operation error was not preserved as effective diagnostic: %+v", got)
		}
		if got.errorCategory != NonFastForward {
			t.Fatalf("operation category = %q, want %q; diagnostic=%v", got.errorCategory, NonFastForward, got.err)
		}
	})
}

func TestHandleExecutedPush(t *testing.T) {
	cases := []struct {
		name     string
		action   state.Action
		category GitErrorCategory
		errText  string
		message  string
		detail   string
		confirm  bool
	}{
		{"push permission", state.ActionPush, PermissionDenied, "Permission denied (publickey)", "Auth or permission error.", "Check credentials or network: Permission denied (publickey)", false},
		{"push non-fast-forward", state.ActionPush, NonFastForward, "! [rejected] main -> main (non-fast-forward)", "", "", true},
		{"push generic", state.ActionPush, Unknown, "original diagnostic", "Push failed.", "original diagnostic", false},
		{"force permission", state.ActionForcePush, PermissionDenied, "Permission denied (publickey)", "Auth or permission error.", "Check credentials or network: Permission denied (publickey)", false},
		{"force non-fast-forward", state.ActionForcePush, NonFastForward, "original diagnostic", "Push failed.", "original diagnostic", false},
		{"force generic", state.ActionForcePush, Unknown, "original diagnostic", "Push failed.", "original diagnostic", false},
		{"set-upstream permission", state.ActionSetUpstream, PermissionDenied, "Permission denied (publickey)", "Auth or permission error.", "Check credentials or network: Permission denied (publickey)", false},
		{"set-upstream non-fast-forward", state.ActionSetUpstream, NonFastForward, "original diagnostic", "Push failed.", "original diagnostic", false},
		{"set-upstream generic", state.ActionSetUpstream, Unknown, "original diagnostic", "Push failed.", "original diagnostic", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := model{status: state.New().WithLoading("pushing"), repositoryState: repositoryState{repoStatus: git.Status{Branch: "main", Head: "head", Upstream: "origin/main"}}}
			next, cmd := handleExecutedUpdate(m, executedMsg{
				action: tc.action, target: "main", err: errors.New(tc.errText), operationErr: errors.New(tc.errText), errorCategory: tc.category,
			})
			if cmd != nil {
				t.Fatalf("category %q returned command", tc.category)
			}
			got := next.(model)
			if tc.confirm {
				if got.status.Mode != state.ModeConfirm || got.status.Action != state.ActionForcePush {
					t.Fatalf("category %q did not route to force confirmation: %+v", tc.category, got.status)
				}
			} else if got.status.Mode != state.ModeBlocked || got.status.Message != tc.message || got.status.Detail != tc.detail {
				t.Fatalf("category %q route = mode=%q message=%q detail=%q", tc.category, got.status.Mode, got.status.Message, got.status.Detail)
			}
		})
	}
}

func TestHandleExecutedExcludedPush(t *testing.T) {
	cases := []struct {
		name   string
		action state.Action
		err    string
		want   string
	}{
		{"push tag auth substring", state.ActionPushTag, "Permission denied by remote", "Auth or permission error."},
		{"push tag category ignored", state.ActionPushTag, "unrelated tag diagnostic", "Push failed."},
		{"delete remote tag not found substring", state.ActionDeleteRemoteTag, "remote tag not found", "Remote tag not found."},
		{"delete remote tag category ignored", state.ActionDeleteRemoteTag, "unrelated remote tag diagnostic", "Push failed."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := model{status: state.New().WithLoading("working")}
			next, _ := handleExecutedUpdate(m, executedMsg{action: tc.action, target: "v1", err: errors.New(tc.err), errorCategory: PermissionDenied})
			got := next.(model)
			if got.status.Message != tc.want {
				t.Fatalf("legacy %s route message = %q, want %q", tc.action, got.status.Message, tc.want)
			}
		})
	}
}

func TestHandleExecutedStatusOnlyAndBothErrors(t *testing.T) {
	opErr := errors.New("! [rejected] main -> main (non-fast-forward)")
	statusErr := errors.New("status refresh diagnostic")
	cases := []struct {
		name      string
		operation error
		status    error
		effective error
		category  GitErrorCategory
		wantForce bool
	}{
		{"success", nil, nil, nil, "", false},
		{"operation only", opErr, nil, opErr, NonFastForward, true},
		{"status only", nil, statusErr, statusErr, Unknown, false},
		{"both errors", opErr, statusErr, opErr, NonFastForward, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// This is intentionally a boundary fixture: executePush currently calls
			// Repo.Push and Repo.Status directly, while Repo is concrete and has no
			// injectable operation/status seam. The next implementation step adds
			// these provenance fields to executedMsg; until then, a real command
			// cannot independently produce a status-only/both-error result here.
			msg := executedMsg{action: state.ActionPush, target: "main", err: tc.effective, operationErr: tc.operation, statusErr: tc.status, errorCategory: tc.category}
			if msg.err != tc.effective {
				t.Fatalf("effective detail = %v, want %v", msg.err, tc.effective)
			}
			if msg.operationErr != tc.operation {
				t.Fatalf("operation detail = %v, want %v", msg.operationErr, tc.operation)
			}
			if msg.statusErr != tc.status {
				t.Fatalf("status detail = %v, want %v", msg.statusErr, tc.status)
			}
			if msg.errorCategory != tc.category {
				t.Fatalf("category = %q, want %q", msg.errorCategory, tc.category)
			}
			m := model{status: state.New().WithLoading("pushing"), repositoryState: repositoryState{repoStatus: git.Status{Branch: "main", Head: "head", Upstream: "origin/main"}}}
			next, cmd := handleExecutedUpdate(m, msg)
			if cmd != nil {
				t.Fatalf("truth-table case %q returned command", tc.name)
			}
			got := next.(model)
			if (got.status.Mode == state.ModeConfirm) != tc.wantForce {
				t.Fatalf("truth-table case %q force confirmation=%v, want %v; status=%+v", tc.name, got.status.Mode == state.ModeConfirm, tc.wantForce, got.status)
			}
		})
	}
}

func TestHandleExecutedForceConfirmation(t *testing.T) {
	sink := &recordingEventSink{}
	opErr := errors.New("! [rejected] main -> main (non-fast-forward)")
	m := model{status: state.New().WithLoading("pushing"), repositoryState: repositoryState{repoStatus: git.Status{Branch: "main", Head: "local-head", Upstream: "origin/main"}}, eventSink: sink}
	next, cmd := handleExecutedUpdate(m, executedMsg{
		action: state.ActionPush, target: "main", err: opErr, operationErr: opErr, errorCategory: NonFastForward,
	})
	if cmd != nil {
		t.Fatalf("force confirmation early return emitted command: %v", cmd)
	}
	got := next.(model)
	if got.status.Title != "Force push to origin/main?" || !strings.Contains(got.status.Detail, "overwrite origin/main history") {
		t.Fatalf("confirmation copy changed: %+v", got.status)
	}
	if len(sink.events) != 1 {
		t.Fatalf("events = %#v, want one push_force_confirmation event", sink.events)
	}
	event := sink.events[0]
	if event.Source != "app" || event.Name != "push_force_confirmation" {
		t.Fatalf("confirmation event identity = %#v", event)
	}
	wantFields := map[string]string{"action": "push", "target": "main", "error": opErr.Error()}
	if !reflect.DeepEqual(event.Fields, wantFields) {
		t.Fatalf("confirmation event fields = %#v, want %#v", event.Fields, wantFields)
	}
	if event.Fields["error"] != opErr.Error() {
		t.Fatalf("confirmation event did not preserve original operation diagnostic: %#v", event.Fields)
	}
}

func TestHandleExecutedPushCategoryWithoutOperationDiagnostic(t *testing.T) {
	m := model{status: state.New().WithLoading("pushing"), repositoryState: repositoryState{repoStatus: git.Status{Branch: "main"}}}
	next, cmd := handleExecutedUpdate(m, executedMsg{
		action: state.ActionPush, target: "main", err: errors.New("category-only diagnostic"), errorCategory: NonFastForward,
	})
	if cmd != nil {
		t.Fatalf("category-only diagnostic returned command: %v", cmd)
	}
	got := next.(model)
	if got.status.Mode != state.ModeBlocked || got.status.Message != "Push failed." || got.status.Detail != "category-only diagnostic" {
		t.Fatalf("category-only diagnostic route = %+v, want blocked push failure", got.status)
	}
}

var _ = strings.Contains

func TestExecutePushCommandBoundary(t *testing.T) {
	cases := []struct {
		name         string
		invoke       func(*git.Repo, string, int) tea.Cmd
		action       state.Action
		wantForce    bool
		wantUpstream bool
	}{
		{"ordinary", executePush, state.ActionPush, false, false},
		{"force", executeForcePush, state.ActionForcePush, true, false},
		{"set-upstream", executePushSetUpstream, state.ActionSetUpstream, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotBranch string
			var gotForce, gotUpstream bool
			original := pushStatusOps
			pushStatusOps = pushStatusOperations{
				push: func(_ context.Context, _ *git.Repo, branch string, force, setUpstream bool) (string, error) {
					gotBranch, gotForce, gotUpstream = branch, force, setUpstream
					return "pushed", nil
				},
				status: func(_ context.Context, _ *git.Repo, limit int) (git.Status, error) {
					if limit != 40 {
						t.Fatalf("status limit = %d, want 40", limit)
					}
					return git.Status{Branch: "main", Head: "head"}, nil
				},
			}
			defer func() { pushStatusOps = original }()

			result := cmdResult(t, tc.invoke(nil, "main", 40))
			msg, ok := result.(executedMsg)
			if !ok {
				t.Fatalf("result type = %T, want executedMsg", result)
			}
			if gotBranch != "main" || gotForce != tc.wantForce || gotUpstream != tc.wantUpstream {
				t.Fatalf("push args = branch %q, force %v, upstream %v; want branch main, force %v, upstream %v", gotBranch, gotForce, gotUpstream, tc.wantForce, tc.wantUpstream)
			}
			if msg.action != tc.action || msg.target != "main" || msg.err != nil || msg.operationErr != nil || msg.statusErr != nil {
				t.Fatalf("result = %+v", msg)
			}
		})
	}
}

func TestExecutePushCommandBoundaryErrorProvenance(t *testing.T) {
	opErr := errors.New("operation diagnostic")
	statusErr := errors.New("status diagnostic")
	variants := []struct {
		name   string
		invoke func(*git.Repo, string, int) tea.Cmd
	}{
		{"ordinary", executePush},
		{"force", executeForcePush},
		{"set-upstream", executePushSetUpstream},
	}
	cases := []struct {
		name      string
		operation error
		status    error
		effective error
		category  GitErrorCategory
	}{
		{"success", nil, nil, nil, ""},
		{"operation only", opErr, nil, opErr, Unknown},
		{"status only", nil, statusErr, statusErr, Unknown},
		{"both errors", opErr, statusErr, opErr, Unknown},
	}
	for _, variant := range variants {
		for _, tc := range cases {
			t.Run(variant.name+"/"+tc.name, func(t *testing.T) {
				original := pushStatusOps
				pushStatusOps = pushStatusOperations{
					push:   func(context.Context, *git.Repo, string, bool, bool) (string, error) { return "", tc.operation },
					status: func(context.Context, *git.Repo, int) (git.Status, error) { return git.Status{}, tc.status },
				}
				defer func() { pushStatusOps = original }()

				msg := cmdResult(t, variant.invoke(nil, "main", 40)).(executedMsg)
				if msg.err != tc.effective || msg.operationErr != tc.operation || msg.statusErr != tc.status || msg.errorCategory != tc.category {
					t.Fatalf("result = %+v, want effective=%v operation=%v status=%v category=%q", msg, tc.effective, tc.operation, tc.status, tc.category)
				}
			})
		}
	}
}
