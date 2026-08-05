package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("Isabel2024!")
	if err != nil {
		t.Fatalf("HashPassword devolvió error: %v", err)
	}
	if hash == "Isabel2024!" {
		t.Fatal("el hash no debe ser igual a la contraseña en texto plano")
	}
	if !CheckPassword(hash, "Isabel2024!") {
		t.Error("CheckPassword debería aceptar la contraseña correcta")
	}
	if CheckPassword(hash, "otra-contraseña") {
		t.Error("CheckPassword no debería aceptar una contraseña incorrecta")
	}
}

func TestHashPasswordProducesDifferentHashesForSameInput(t *testing.T) {
	// bcrypt usa un salt aleatorio: dos hashes de la misma contraseña deben
	// ser distintos entre sí, aunque ambos la validen correctamente.
	h1, err := HashPassword("Miguel2024!")
	if err != nil {
		t.Fatalf("HashPassword devolvió error: %v", err)
	}
	h2, err := HashPassword("Miguel2024!")
	if err != nil {
		t.Fatalf("HashPassword devolvió error: %v", err)
	}
	if h1 == h2 {
		t.Error("dos hashes de la misma contraseña no deberían ser idénticos (falta el salt)")
	}
	if !CheckPassword(h1, "Miguel2024!") || !CheckPassword(h2, "Miguel2024!") {
		t.Error("ambos hashes deberían validar la contraseña original")
	}
}
