package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/hnl/bank-backend/internal/auth"
)

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(corsMiddleware(s.cfg.CORSOrigin))
	r.Use(maxBodySize(1 << 20)) // 1 MiB: de sobra para cualquier request legítimo de esta API

	authLimiter := newIPRateLimiter(2, 10)        // ~2 req/s por IP, ráfaga de 10: registro/login
	txLimiter := newIPRateLimiter(5, 20)          // ~5 req/s por IP, ráfaga de 20: depósito/retiro/transferencia

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Use(authLimiter.Middleware)
			r.Post("/register", s.handleRegister)
			r.Post("/login", s.handleLogin)
			r.Post("/2fa/verify", s.handleVerify2FA)
		})

		// El WebSocket de notificaciones autentica el token por query param
		// (el navegador no permite headers personalizados al abrir un WS),
		// por lo que va fuera del grupo con auth.Middleware y valida el
		// token dentro de su propio handler.
		r.Get("/ws", s.handleWebSocket)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(s.cfg.JWTSecret, s.revocation.IsRevoked))

			r.Post("/auth/logout", s.handleLogout)
			r.Get("/auth/2fa/status", s.handle2FAStatus)
			r.Post("/auth/2fa/setup", s.handle2FASetup)
			r.Post("/auth/2fa/enable", s.handle2FAEnable)
			r.Post("/auth/2fa/disable", s.handle2FADisable)

			r.Get("/accounts", s.handleListAccounts)
			r.Get("/accounts/{number}", s.handleGetAccount)
			r.Get("/accounts/{number}/balance-history", s.handleBalanceHistory)

			r.Group(func(r chi.Router) {
				r.Use(txLimiter.Middleware)
				r.Post("/transactions/deposit", s.handleDeposit)
				r.Post("/transactions/withdraw", s.handleWithdraw)
				r.Post("/transactions/transfer", s.handleTransfer)
			})
			r.Get("/transactions", s.handleHistory)
			r.Get("/transactions/export", s.handleExportHistory)

			r.Get("/dashboard", s.handleDashboard)

			r.Post("/chat", s.handleChat)
		})
	})

	return r
}
