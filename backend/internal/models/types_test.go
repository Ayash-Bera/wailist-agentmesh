package models_test

import (
	"encoding/json"
	"testing"

	"github.com/agentmesh/backend/internal/models"
)

func TestWorkflowGraphRoundtrip(t *testing.T) {
	g := models.WorkflowGraph{
		Nodes: []models.WorkflowNode{
			{ID: "n1", Type: models.NodeTypeTrigger, Template: "chat"},
			{ID: "n2", Type: models.NodeTypeAgent, SystemPrompt: "You are helpful"},
		},
		Edges: []models.WorkflowEdge{
			{ID: "e1", From: "n1", To: "n2", Kind: models.EdgeKindFlow, ToPort: "in"},
		},
	}
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var g2 models.WorkflowGraph
	if err := json.Unmarshal(b, &g2); err != nil {
		t.Fatal(err)
	}
	if len(g2.Nodes) != 2 || len(g2.Edges) != 1 {
		t.Fatalf("got %d nodes %d edges", len(g2.Nodes), len(g2.Edges))
	}
	if g2.Nodes[1].SystemPrompt != "You are helpful" {
		t.Fatal("systemPrompt lost")
	}
}
