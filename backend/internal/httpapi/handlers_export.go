package httpapi

import (
	"encoding/csv"
	"net/http"

	"github.com/hnl/bank-backend/internal/auth"
)

// handleExportHistory exporta el historial de transacciones del usuario como CSV descargable.
func (s *Server) handleExportHistory(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	accountNumber := r.URL.Query().Get("account")

	// Tope razonable para una exportación puntual desde el navegador; el
	// historial paginado (GET /api/transactions) sigue siendo la vía normal
	// para navegar grandes volúmenes de movimientos.
	const exportLimit = 5000
	txs, _, err := s.banking.GetHistory(r.Context(), userID, accountNumber, 1, exportLimit)
	if err != nil {
		mapBankingError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="historial-hnl-bank.csv"`)
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"fecha", "tipo", "descripcion", "cuenta_origen", "cuenta_destino", "monto_usd", "estado"})
	for _, t := range txs {
		_ = writer.Write([]string{
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			string(t.Type),
			t.Description,
			t.FromAccountNumber,
			t.ToAccountNumber,
			t.Amount,
			string(t.Status),
		})
	}
	writer.Flush()
}
