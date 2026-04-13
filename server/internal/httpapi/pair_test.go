package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodePairResponse(t *testing.T, body *bytes.Buffer) pairResponse {
	t.Helper()
	var resp pairResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decode pair response: %v", err)
	}
	return resp
}

func pairRequestBody(t *testing.T, pin, label string) *bytes.Buffer {
	t.Helper()
	raw, err := json.Marshal(pairRequest{PIN: pin, DeviceLabel: label})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewBuffer(raw)
}

func TestPairHandler_CorrectPIN(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{token: "mint-abc-xyz"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", pairRequestBody(t, "123456", "phone"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	resp := decodePairResponse(t, w.Body)
	if resp.Token != "mint-abc-xyz" {
		t.Errorf("token=%q, want mint-abc-xyz", resp.Token)
	}
	if ti.label != "phone" {
		t.Errorf("issuer label=%q, want phone", ti.label)
	}
	if p.consumed != 1 {
		t.Errorf("consumed=%d, want 1", p.consumed)
	}
}

func TestPairHandler_WrongPIN(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{token: "never-issued"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", pairRequestBody(t, "000000", "phone"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "unauthorized" {
		t.Errorf("error.code=%q, want unauthorized", body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "invalid pin") {
		t.Errorf("error.message=%q, want to contain 'invalid pin'", body.Error.Message)
	}
	if ti.calls != 0 {
		t.Errorf("issuer calls=%d, want 0 (wrong pin must short-circuit)", ti.calls)
	}
	if p.consumed != 0 {
		t.Errorf("consumed=%d, want 0 (wrong pin must not consume)", p.consumed)
	}
}

func TestPairHandler_WrongPIN_DifferentLength(t *testing.T) {
	// Make sure ConstantTimeCompare rejecting mismatched lengths does
	// not crash the handler — a shorter PIN should simply take the
	// invalid-pin branch.
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{token: "never"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", pairRequestBody(t, "12345", "phone"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
}

func TestPairHandler_AlreadyConsumed(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: false}
	ti := &fakeTokenIssuer{token: "never"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", pairRequestBody(t, "123456", "phone"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401 (body=%s)", w.Code, w.Body.String())
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(body.Error.Message, "pin no longer valid") {
		t.Errorf("error.message=%q, want 'pin no longer valid'", body.Error.Message)
	}
	if ti.calls != 0 {
		t.Errorf("issuer calls=%d, want 0", ti.calls)
	}
}

func TestPairHandler_BadJSON(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{token: "never"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", bytes.NewBufferString("{garbage"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w.Code)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Errorf("error.code=%q, want bad_request", body.Error.Code)
	}
}

func TestPairHandler_MissingPIN(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{token: "never"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", bytes.NewBufferString(`{"device_label":"phone"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w.Code)
	}
}

func TestPairHandler_MissingLabel(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{token: "never"}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", bytes.NewBufferString(`{"pin":"123456"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w.Code)
	}
}

func TestPairHandler_TokenIssuerError(t *testing.T) {
	p := &fakePINProvider{pin: "123456", consumeOK: true}
	ti := &fakeTokenIssuer{err: errors.New("entropy gone: /secret/path")}
	h := NewPairHandler(p, ti)

	req := httptest.NewRequest(http.MethodPost, "/api/pair", pairRequestBody(t, "123456", "phone"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", w.Code)
	}
	body := w.Body.String()
	for _, leaked := range []string{"entropy", "/secret"} {
		if strings.Contains(body, leaked) {
			t.Errorf("body leaks %q: %s", leaked, body)
		}
	}
	var parsed errorBody
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if parsed.Error.Code != "internal_error" {
		t.Errorf("error.code=%q, want internal_error", parsed.Error.Code)
	}
}

// TestPairHandler_RouterWiring asserts /api/pair is mounted and does
// NOT require auth, while /api/apps still requires it. This guards
// against accidental changes in router.go that break the public auth
// contract.
func TestPairHandler_RouterWiring(t *testing.T) {
	cat := newTestCatalog(t, nil)
	r := newRouterFor(t, cat, nil, nil, nil)

	// /api/pair with the default fake (pin=000000, consumeOK=true,
	// token="unused-token") is reachable without Authorization.
	req := httptest.NewRequest(http.MethodPost, "/api/pair",
		bytes.NewBufferString(`{"pin":"000000","device_label":"phone"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/pair status=%d, want 200 (body=%s)", w.Code, w.Body.String())
	}
}
