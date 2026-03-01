package resolve

import "testing"

func TestSimilarityOrdering(t *testing.T) {
	exact := similarity("harp", "harp")
	prefix := similarity("har", "harp")
	fuzzy := similarity("hrp", "harp")
	bad := similarity("xyz", "harp")
	if !(exact > prefix && prefix > fuzzy && fuzzy > bad) {
		t.Fatalf("unexpected similarity ordering: exact=%.2f prefix=%.2f fuzzy=%.2f bad=%.2f", exact, prefix, fuzzy, bad)
	}
}

func TestShouldSuggestCorrection(t *testing.T) {
	matches := []Match{{Score: 0.80}, {Score: 0.60}}
	if !shouldSuggestCorrection(matches, 0.72, 0.10) {
		t.Fatalf("expected correction suggestion")
	}
	if shouldSuggestCorrection(matches, 0.90, 0.10) {
		t.Fatalf("expected no suggestion due to min score")
	}
	if shouldSuggestCorrection(matches, 0.72, 0.30) {
		t.Fatalf("expected no suggestion due to min gap")
	}
}
