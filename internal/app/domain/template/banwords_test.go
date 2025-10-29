package template_test

import (
	"strings"
	"testing"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
)

func TestBanwords_CheckMessage(t *testing.T) {
	t.Parallel()
	bw := config.Banwords{
		Words: []string{
			"пидр", "нига", "pidr", "niga",
			"хач", "хача", "хачем", "хаче", "хачи", "хачей", "хачам", "хачами", "хачах",
			"хачик", "хачику", "хачике", "хачиках", "хачики",
			"жид", "жида", "жидом", "жиду", "жидов", "жиде", "жиды", "жидам", "жидами", "жидах",
			"жидик", "жидику", "жидике", "жидиках", "жидики",
			"хохол", "хохлы", "хохлу", "хохлам", "хохле", "хохлом", "хохлами", "хохлов", "хохлах",
			"хахол", "хахлы", "хахлу", "хахлам", "хахле", "хахлом", "хахлами", "хахлов", "хахлах",
			"русня", "русней", "русне", "русню", "руснёй", "русни", "руснями",
			"кацап", "кацапа", "кацапом", "кацапу", "кацапов", "кацапе", "кацапы",
			"кацапам", "кацапами", "кацапах", "кацапик", "кацапику", "кацапике", "кацапиках",
			"симп", "симпа", "симпом", "симпу", "симпов", "симпе", "симпы", "симпам", "симпами", "симпах",
		},
		ContainsWords: []string{
			"пидор", "пидар", "педик", "нигг", "негр", "хохляцк", "хахляцк",
			"русняв", "пендос", "пиндос", "черножоп", "чернажоп", "гомосек", "гомик",
			"инцел", "куколд", "faggot", "incel", "cuckold", "negr", "pidor", "pidar", "nigg",
		},
		CaseSensitiveWords: []string{
			"неоЖИДанно", "неоЖИДано", "НЕГРамотный",
			"асПИДОРАС", "асПИДОРАСЫ", "асПИДОР", "асПИДОРЫ",
			"ПЕДИКюр", "ХАЧу", "НИГЕРмания", "СИМПл",
		},
		ExcludeWords: []string{
			"неграмот", "винегрет", "негрони",
		},
	}

	bt := template.NewBanwords(bw)
	tests := []struct {
		name       string
		input      string
		wantBanned bool
	}{
		{"Прямое совпадение — пидр", "ты пидр", true},
		{"Частичное совпадение — пидор", "он пидорский", true},
		{"Case-sensitive совпадение", "это было неоЖИДанно", true},
		{"Case-sensitive несовпадение", "это было неожиданно", false},
		{"Исключение (винегрет)", "я люблю винегрет", false},
		{"Чистый текст", "привет, как дела", false},
		{"Английское совпадение (niga)", "you niga", true},
		{"Transliterated совпадение", "pidor", true},
		{"Пидарас совпадение", "Пидарас", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wordsOriginal := strings.Fields(tt.input)
			wordsLower := strings.Fields(strings.ToLower(tt.input))

			got := bt.CheckMessage(wordsOriginal, wordsLower)
			if got != tt.wantBanned {
				t.Errorf("CheckMessage(%q) = %v, want %v", tt.input, got, tt.wantBanned)
			}
		})
	}
}
