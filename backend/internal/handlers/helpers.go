package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5" 
)

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	type ErrorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

func mapDBError(err error) (int, string) {
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound, "Resource not found"
	}
	log.Printf("ERROR: Database operation failed: %v", err)
	return http.StatusInternalServerError, "Internal Server Error"
}

