package ledger

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// AmountPattern es el formato aceptado para montos en las peticiones de la API:
// un entero opcionalmente seguido de hasta 2 decimales. Ej: "100", "100.5", "100.50".
var AmountPattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

// DecimalStringToCents convierte un monto decimal (ej. "100.50") a centavos (10050),
// sin pasar por punto flotante, para evitar errores de redondeo en dinero.
func DecimalStringToCents(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if !AmountPattern.MatchString(amount) {
		return 0, fmt.Errorf("invalid amount format: %q", amount)
	}

	parts := strings.SplitN(amount, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}

	var cents int64
	if len(parts) == 2 {
		frac := parts[1]
		for len(frac) < 2 {
			frac += "0"
		}
		fracVal, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount: %w", err)
		}
		cents = fracVal
	}

	total := whole*100 + cents
	if total <= 0 {
		return 0, fmt.Errorf("amount must be greater than zero")
	}
	return total, nil
}

// CentsToDecimalString formatea centavos como un monto decimal legible, ej. 10050 -> "100.50".
func CentsToDecimalString(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// FloatToCents convierte un float64 (como los del dataset de prueba JSON) a centavos.
// Solo se usa durante el seeding, donde la fuente son floats de un archivo JSON externo.
func FloatToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}
