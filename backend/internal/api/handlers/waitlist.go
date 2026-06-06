package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/agentmesh/backend/internal/respond"
)

func (d *Deps) JoinWaitlist(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	if body.Email == "" || !strings.Contains(body.Email, "@") {
		respond.Error(w, http.StatusBadRequest, "valid email required")
		return
	}

	if err := d.Store.InsertWaitlistEmail(r.Context(), body.Email); err != nil {
		log.Printf("waitlist insert: %v", err)
		respond.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	respond.JSON(w, http.StatusCreated, map[string]string{"status": "joined"})
}
