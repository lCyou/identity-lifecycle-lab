// Package e2e_test drives the HTTP API end-to-end over a real network socket,
// exercising the same flow an external client would use, as opposed to the
// package-level unit/black-box tests under internal/*.
package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lCyou/identity-lifecycle-lab/internal/api"
	"github.com/lCyou/identity-lifecycle-lab/internal/dbtest"
	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// TestFullLifecycleOverHTTP は enrolled から archived までの全遷移を
// 実際のHTTPサーバー越しに実行し、途中の不正遷移が拒否されることと
// 監査ログが最終的に一貫していることを確認する。
func TestFullLifecycleOverHTTP(t *testing.T) {
	server := httptest.NewServer(api.NewRouter(identity.NewStore(dbtest.Open(t))))
	defer server.Close()

	created := decode[identity.Entity](t, postJSON(t, server.URL+"/entities", map[string]string{"name": "alice"}))
	if created.State != identity.StateEnrolled {
		t.Fatalf("want enrolled, got %s", created.State)
	}

	transitionsURL := server.URL + "/entities/" + created.ID + "/transitions"

	resp := postJSON(t, transitionsURL, map[string]string{"to": "issued", "actor": "registrar", "reason": "credentials issued"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = postJSON(t, transitionsURL, map[string]string{"to": "active", "actor": "registrar", "reason": "start use"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("activate: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = postJSON(t, transitionsURL, map[string]string{"to": "suspended", "actor": "fraud-system", "reason": "suspicious activity detected"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("suspend: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// suspended -> archived はいきなり許可されていない (suspended から行けるのは active か revoked のみ)
	resp = postJSON(t, transitionsURL, map[string]string{"to": "archived", "actor": "admin", "reason": "skip ahead"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("suspended->archived: want 409, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = postJSON(t, transitionsURL, map[string]string{"to": "revoked", "actor": "admin", "reason": "compromised credential"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("revoke: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// revoked -> active への直接復活は不可 (このリポジトリの中核ルール)
	resp = postJSON(t, transitionsURL, map[string]string{"to": "active", "actor": "admin", "reason": "attempt bypass"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("revoked->active: want 409, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = postJSON(t, transitionsURL, map[string]string{"to": "archived", "actor": "admin", "reason": "retention period elapsed"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("archive: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	histResp, err := http.Get(transitionsURL)
	if err != nil {
		t.Fatal(err)
	}
	hist := decode[[]identity.TransitionRecord](t, histResp)

	// 記録されるのは成功した遷移のみ: enrolled(初期) -> issued -> active -> suspended -> revoked -> archived
	if len(hist) != 6 {
		t.Fatalf("want 6 history entries, got %d: %+v", len(hist), hist)
	}
	if last := hist[len(hist)-1]; last.ToState != identity.StateArchived || last.Actor != "admin" {
		t.Fatalf("unexpected last history entry: %+v", last)
	}
}
