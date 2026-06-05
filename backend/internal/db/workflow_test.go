package db_test

import (
	"context"
	"testing"

	"github.com/agentmesh/backend/internal/models"
)

func TestWorkflowCRUD(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	wf, err := store.CreateWorkflow(ctx, "Test WF", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if wf.Name != "Test WF" {
		t.Fatalf("want 'Test WF' got %q", wf.Name)
	}
	if wf.Status != models.WorkflowStatusDraft {
		t.Fatalf("want draft got %s", wf.Status)
	}

	wf2, err := store.GetWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wf2.ID != wf.ID {
		t.Fatal("id mismatch")
	}

	graph := models.WorkflowGraph{
		Nodes: []models.WorkflowNode{{ID: "n1", Type: models.NodeTypeTrigger}},
		Edges: []models.WorkflowEdge{},
	}
	updated, err := store.UpdateWorkflow(ctx, wf.ID, "Renamed", graph)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed" {
		t.Fatal("name not updated")
	}
	if len(updated.Nodes) != 1 {
		t.Fatal("nodes not saved")
	}

	list, err := store.ListWorkflows(ctx, "dev")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range list {
		if w.ID == wf.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("workflow not in list")
	}

	if err := store.DeleteWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetWorkflow(ctx, wf.ID)
	if err == nil {
		t.Fatal("expected not found error after delete")
	}
}
