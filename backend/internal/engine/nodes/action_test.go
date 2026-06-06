package nodes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/backend/internal/engine"
	"github.com/agentmesh/backend/internal/engine/nodes"
	"github.com/agentmesh/backend/internal/models"
)

func TestWebhookAction(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	node := models.WorkflowNode{ID: "a1", Type: models.NodeTypeAction, Template: "webhook", URL: srv.URL}
	rc := engine.NewRunContext("r1", []byte(`{"message":"test payload"}`))
	_, err := nodes.ExecuteAction(context.Background(), node, rc)
	if err != nil {
		t.Fatal(err)
	}
	if received == nil {
		t.Fatal("webhook not called")
	}
}

func TestLogAction(t *testing.T) {
	node := models.WorkflowNode{ID: "a2", Type: models.NodeTypeAction, Template: "log"}
	rc := engine.NewRunContext("r1", []byte(`"hello"`))
	result, err := nodes.ExecuteAction(context.Background(), node, rc)
	if err != nil {
		t.Fatal(err)
	}
	if result != "logged" {
		t.Fatalf("want 'logged' got %v", result)
	}
}
