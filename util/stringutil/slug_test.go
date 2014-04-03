package stringutil

import (
	"testing"
)

func testSlug(t *testing.T, expected, original string, n int) {
	slug := SlugN(original, n)
	if expected != slug {
		t.Errorf("Unexpected slug for \"%s\", expecting \"%s\", got \"%s\"", original, expected, slug)
	}
}

func TestSlug(t *testing.T) {
	testSlug(t, "the-quick-brown-fox-jumps-over-the-lazy-dog", "The quick brown fox jumps over the lazy dog", -1)
	testSlug(t, "the-quick-brown-fox-jumps-over-the-lazy-dog", "The quick brown fox jumps over the lazy dog", 100)
	testSlug(t, "the", "The quick brown fox jumps over the lazy dog", 3)
	testSlug(t, "the", "The quick brown fox jumps over the lazy dog", 4)
	testSlug(t, "the-q", "The quick brown fox jumps over the lazy dog", 5)
	testSlug(t, "quita-del-37-5-para-los-grandes-depositos-del-banco-de-chipre", "Quita del 37,5% para los grandes depósitos del Banco de Chipre", -1)
	testSlug(t, "el-bng-pide-el-cese-de-feijoo-y-el-resto-de-la-oposicion-explicaciones", "El BNG pide el cese de Feijóo y el resto de la oposición, explicaciones", -1)
	testSlug(t, "el-papa-pide-una-solucion-politica-para-el-conflicto-en-siria", "El papa pide una “solución política” para el conflicto en Siria", -1)
}
