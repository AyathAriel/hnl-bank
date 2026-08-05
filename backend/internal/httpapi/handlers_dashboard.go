package httpapi

import (
	"net/http"

	"github.com/hnl/bank-backend/internal/auth"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	data, err := s.banking.GetDashboard(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo cargar el dashboard.")
		return
	}
	writeJSON(w, http.StatusOK, data)
}
