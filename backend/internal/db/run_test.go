package db_test

import (
	"context"
	"testing"

	"github.com/agentmesh/backend/internal/models"
)

func TestRunAndLogs(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	wf, _ := store.CreateWorkflow(ctx, "RunTest", "dev")
	t.Cleanup(func() { store.DeleteWorkflow(ctx, wf.ID) })

	run, err := store.CreateRun(ctx, wf.ID, "manual", []byte(`{"message":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != models.RunStatusRunning {
		t.Fatal("expected running")
	}

	logEntry, err := store.InsertRunLog(ctx, models.RunLog{
		RunID: run.ID, StepIndex: 0,
		NodeID: "n1", NodeType: models.NodeTypeTrigger,
		Status: models.LogStatusRunning,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateRunLog(ctx, logEntry.ID, models.LogStatusSuccess, []byte(`"done"`), 42); err != nil {
		t.Fatal(err)
	}

	logs, err := store.GetRunLogs(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Status != models.LogStatusSuccess {
		t.Fatal("log not updated correctly")
	}
	if logs[0].DurationMs != 42 {
		t.Fatalf("want 42ms got %d", logs[0].DurationMs)
	}

	if err := store.FinishRun(ctx, run.ID, models.RunStatusSuccess); err != nil {
		t.Fatal(err)
	}
	got, _ := store.GetRun(ctx, run.ID)
	if got.Status != models.RunStatusSuccess {
		t.Fatal("run not finished")
	}
}

func TestAgentWallet(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	wf, _ := store.CreateWorkflow(ctx, "WalletTest", "dev")
	t.Cleanup(func() { store.DeleteWorkflow(ctx, wf.ID) })

	err := store.InsertAgentWallet(ctx, models.AgentWallet{
		WorkflowID:        wf.ID,
		AgentNodeID:       "agent1",
		Address:           "ALGO123456",
		EncryptedMnemonic: "enc-mnemonic",
		Network:           "testnet",
	})
	if err != nil {
		t.Fatal(err)
	}

	w, err := store.GetAgentWallet(ctx, wf.ID, "agent1")
	if err != nil {
		t.Fatal(err)
	}
	if w.Address != "ALGO123456" {
		t.Fatalf("want ALGO123456 got %s", w.Address)
	}
	if w.EncryptedMnemonic != "enc-mnemonic" {
		t.Fatal("mnemonic not persisted")
	}

	wallets, err := store.ListAgentWallets(ctx, wf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 1 {
		t.Fatalf("want 1 wallet got %d", len(wallets))
	}
}
