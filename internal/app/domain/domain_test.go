package domain_test

import (
	"strings"
	"testing"
	"twitchspam/internal/app/domain"
)

func TestJaccardHashSimilarity(t *testing.T) {
	t.Parallel()

	aText := "когда Шевчик душнит, он начинает быть похожим на моего Деда, но Деду 70 лет, а Вове 27 0 @Stintik"
	bText1 := "@Stintik когда Шевчик душнит, он начинает быть похожим на моего Деда, но Деду 70 лет, а Вове 27 0"
	bText2 := "@Stintik когда Шевчик душнит, он начинает быть похожим на моего Деда, но Деду 70 лет, а Вове 27 0 @Stintik"

	a := strings.Fields(aText)
	b1 := strings.Fields(bText1)
	b2 := strings.Fields(bText2)

	sim1 := domain.JaccardHashSimilarity(a, b1)
	sim2 := domain.JaccardHashSimilarity(a, b2)

	if sim1 <= 0.7 {
		t.Errorf("Expected similarity > 0.7, got %.3f", sim1)
	} else {
		t.Skipf("Jaccard similarity с b1: %.3f\n", sim1)
	}

	if sim2 <= 0.7 {
		t.Errorf("Expected similarity > 0.7, got %.3f", sim2)
	} else {
		t.Skipf("Jaccard similarity с b2: %.3f\n", sim2)
	}
}
