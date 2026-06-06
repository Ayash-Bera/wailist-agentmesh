package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/backend/internal/api/handlers"
)

func TestSignUpReturnsBadRequestOnEmptyEmail(t *testing.T) {
	d := &handlers.Deps{JWTSecret: "testsecret"}
	body, _ := json.Marshal(map[string]string{"email": "", "password": "validpassword"})
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.SignUp(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", w.Code)
	}
}

func TestSignUpReturnsBadRequestOnShortPassword(t *testing.T) {
	d := &handlers.Deps{JWTSecret: "testsecret"}
	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "password": "short"})
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.SignUp(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", w.Code)
	}
}
