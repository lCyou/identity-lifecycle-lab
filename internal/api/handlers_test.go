package api_test

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

func doJSON(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestEntityLifecycleViaAPI(t *testing.T) {
	router := api.NewRouter(identity.NewStore(dbtest.Open(t)))

	createRec := doJSON(t, router, http.MethodPost, "/entities", map[string]string{"name": "alice"})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create entity: want 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created identity.Entity
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.State != identity.StateEnrolled {
		t.Fatalf("want enrolled, got %s", created.State)
	}

	okRec := doJSON(t, router, http.MethodPost, "/entities/"+created.ID+"/transitions",
		map[string]string{"to": "issued", "actor": "registrar", "reason": "issue credentials"})
	if okRec.Code != http.StatusOK {
		t.Fatalf("transition: want 200, got %d: %s", okRec.Code, okRec.Body.String())
	}

	// issued -> revoked は許可された遷移(issued->active)ではないため 409 になるはず
	conflictRec := doJSON(t, router, http.MethodPost, "/entities/"+created.ID+"/transitions",
		map[string]string{"to": "revoked", "actor": "registrar", "reason": "skip ahead"})
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("invalid transition: want 409, got %d: %s", conflictRec.Code, conflictRec.Body.String())
	}

	histRec := doJSON(t, router, http.MethodGet, "/entities/"+created.ID+"/transitions", nil)
	if histRec.Code != http.StatusOK {
		t.Fatalf("history: want 200, got %d", histRec.Code)
	}
	var hist []identity.TransitionRecord
	if err := json.Unmarshal(histRec.Body.Bytes(), &hist); err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 { // 初期登録 + issued (拒否された遷移は記録されない)
		t.Fatalf("want 2 history entries, got %d", len(hist))
	}
}

func TestGetUnknownEntityReturns404(t *testing.T) {
	router := api.NewRouter(identity.NewStore(dbtest.Open(t)))

	rec := doJSON(t, router, http.MethodGet, "/entities/does-not-exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestCreateEntityRequiresName(t *testing.T) {
	router := api.NewRouter(identity.NewStore(dbtest.Open(t)))

	rec := doJSON(t, router, http.MethodPost, "/entities", map[string]string{"name": ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}
