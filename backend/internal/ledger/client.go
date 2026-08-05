// Package ledger encapsula toda la interacción con TigerBeetle: creación de
// cuentas, transferencias con partida doble y consulta de balances/historial.
// Es la única capa del sistema que conoce el SDK de TigerBeetle; el resto del
// backend (internal/banking) solo ve estos métodos de alto nivel.
package ledger

import (
	"errors"
	"fmt"
	"net"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

// Ledger de contabilidad usado por todas las cuentas (dataset de prueba es 100% USD).
const LedgerUSD uint32 = 1

// Códigos de cuenta.
const (
	CodeExternalAccount uint16 = 1
	CodeChecking        uint16 = 1001
	CodeSavings         uint16 = 1002
	CodeInvestment      uint16 = 1003
)

// Códigos de transferencia (para auditoría/clasificación dentro de TigerBeetle).
const (
	CodeOpeningBalance uint16 = 1
	CodeDeposit        uint16 = 2
	CodeWithdrawal     uint16 = 3
	CodeTransfer       uint16 = 4
)

// ExternalAccountID identifica la cuenta de control que representa fondos que
// entran o salen del banco (depósitos externos y retiros hacia el exterior).
// Es un ID fijo y bajo para que nunca choque con los IDs (basados en tiempo,
// vía tb.ID()) generados para cuentas de clientes.
var ExternalAccountID = tb.ToUint128(1)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountNotFound   = errors.New("account not found")
)

// Client envuelve el cliente nativo de TigerBeetle.
type Client struct {
	tb tb.Client
}

func NewClient(clusterID uint64, addresses []string) (*Client, error) {
	resolved, err := resolveAddresses(addresses)
	if err != nil {
		return nil, fmt.Errorf("resolving tigerbeetle addresses: %w", err)
	}

	c, err := tb.NewClient(tb.ToUint128(clusterID), resolved)
	if err != nil {
		return nil, fmt.Errorf("connecting to tigerbeetle: %w", err)
	}
	return &Client{tb: c}, nil
}

// resolveAddresses convierte hostnames (ej. "tigerbeetle", el nombre del
// servicio en docker-compose) a IPs literales: el cliente nativo de
// TigerBeetle solo acepta direcciones "host:puerto" con IP literal, no
// resuelve DNS por sí mismo.
func resolveAddresses(addresses []string) ([]string, error) {
	resolved := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			// No es "host:puerto" (ej. un puerto suelto como "3000"); se deja tal cual.
			resolved = append(resolved, addr)
			continue
		}
		if net.ParseIP(host) != nil {
			resolved = append(resolved, addr)
			continue
		}
		ips, err := net.LookupHost(host)
		if err != nil || len(ips) == 0 {
			return nil, fmt.Errorf("could not resolve host %q: %w", host, err)
		}
		resolved = append(resolved, net.JoinHostPort(ips[0], port))
	}
	return resolved, nil
}

func (c *Client) Close() {
	c.tb.Close()
}

// AccountTypeToCode mapea el tipo de cuenta de dominio al código TigerBeetle.
func AccountTypeToCode(accountType string) uint16 {
	switch accountType {
	case "savings":
		return CodeSavings
	case "investment":
		return CodeInvestment
	default:
		return CodeChecking
	}
}
