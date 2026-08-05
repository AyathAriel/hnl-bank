package ledger

import "testing"

func TestDecimalStringToCents(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"100", 10000, false},
		{"100.5", 10050, false},
		{"100.50", 10050, false},
		{"0.01", 1, false},
		{"1234567.89", 123456789, false},
		{"0", 0, true},          // debe ser > 0
		{"-5", 0, true},         // negativo no es un formato válido
		{"5.999", 0, true},      // más de 2 decimales
		{"abc", 0, true},        // no numérico
		{"5.", 0, true},         // punto sin decimales
		{"", 0, true},           // vacío
	}

	for _, c := range cases {
		got, err := DecimalStringToCents(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("DecimalStringToCents(%q): esperaba error, no hubo (got=%d)", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("DecimalStringToCents(%q): error inesperado: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("DecimalStringToCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestCentsToDecimalString(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{10000, "100.00"},
		{10050, "100.50"},
		{1, "0.01"},
		{0, "0.00"},
		{-1050, "-10.50"},
	}

	for _, c := range cases {
		got := CentsToDecimalString(c.in)
		if got != c.want {
			t.Errorf("CentsToDecimalString(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFloatToCents(t *testing.T) {
	cases := []struct {
		in   float64
		want int64
	}{
		{100.50, 10050},
		{32354.53, 3235453},
		{0.1, 10},
		{0.29, 29}, // guarda contra el clásico error de redondeo binario (0.29*100 = 28.999...)
	}

	for _, c := range cases {
		got := FloatToCents(c.in)
		if got != c.want {
			t.Errorf("FloatToCents(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestMoneyRoundTrip verifica que convertir centavos a string y de vuelta a
// centavos sea siempre una operación de ida y vuelta exacta (crítico para no
// perder ni un centavo en cada depósito/retiro/transferencia).
func TestMoneyRoundTrip(t *testing.T) {
	amounts := []int64{0 + 1, 100, 10050, 999999999, 1}
	for _, cents := range amounts {
		s := CentsToDecimalString(cents)
		back, err := DecimalStringToCents(s)
		if err != nil {
			t.Fatalf("round-trip de %d falló al parsear %q: %v", cents, s, err)
		}
		if back != cents {
			t.Errorf("round-trip de %d dio %d (via %q)", cents, back, s)
		}
	}
}
