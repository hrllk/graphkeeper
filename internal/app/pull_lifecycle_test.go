package app

import "testing"

func TestPullLifecycleIdentityOutcomeDistinguishesIgnoredAndActiveRequests(t *testing.T) {
	req := &pullRequest{ID: 7, Epoch: 11}

	for name, tc := range map[string]struct {
		id, epoch    uint64
		confirmStale bool
		wantKind     pullLifecycleOutcomeKind
	}{
		"matching request":   {id: 7, epoch: 11, wantKind: pullLifecycleActive},
		"request mismatch":   {id: 8, epoch: 11, wantKind: pullLifecycleIdentityIgnored},
		"epoch mismatch":     {id: 7, epoch: 12, wantKind: pullLifecycleIdentityIgnored},
		"stale confirmation": {id: 7, epoch: 11, confirmStale: true, wantKind: pullLifecycleIdentityIgnored},
		"missing request":    {id: 7, epoch: 11, wantKind: pullLifecycleIdentityIgnored},
	} {
		t.Run(name, func(t *testing.T) {
			active := req
			if name == "missing request" {
				active = nil
			}
			got := classifyPullLifecycleIdentity(active, tc.id, tc.epoch, tc.confirmStale)
			if got.Kind != tc.wantKind {
				t.Fatalf("kind = %v, want %v", got.Kind, tc.wantKind)
			}
			if tc.wantKind == pullLifecycleActive && (got.RequestID != req.ID || got.RepositoryEpoch != req.Epoch) {
				t.Fatalf("active identity = (%d, %d), want (%d, %d)", got.RequestID, got.RepositoryEpoch, req.ID, req.Epoch)
			}
		})
	}
}
