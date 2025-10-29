package message_test

import (
	"testing"
	"twitchspam/internal/app/domain/message"
)

func TestTextWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		opts     []message.TextOption
		expected string
	}{
		// Базовые тесты без опций
		{
			name:     "no options",
			original: "Hello World!",
			opts:     []message.TextOption{},
			expected: "Hello World!",
		},
		{
			name:     "no options with punctuation",
			original: "Hello, World! How are you?",
			opts:     []message.TextOption{},
			expected: "Hello, World! How are you?",
		},

		// Тесты с message.LowerOption
		{
			name:     "lower option only",
			original: "Hello WORLD!",
			opts:     []message.TextOption{message.LowerOption},
			expected: "hello world!",
		},
		{
			name:     "lower option with mixed case",
			original: "HeLLo WoRLd!",
			opts:     []message.TextOption{message.LowerOption},
			expected: "hello world!",
		},

		// Тесты с RemovePunctuation
		{
			name:     "remove punctuation only",
			original: "Hello, World! How are you?",
			opts:     []message.TextOption{message.RemovePunctuationOption},
			expected: "Hello World How are you",
		},
		{
			name:     "remove punctuation with various symbols",
			original: "Test@email.com, phone: 123-456!",
			opts:     []message.TextOption{message.RemovePunctuationOption},
			expected: "Testemailcom phone 123456",
		},

		// Тесты с message.RemoveDuplicateLettersOption
		{
			name:     "remove duplicates only",
			original: "Hello   World!!",
			opts:     []message.TextOption{message.RemoveDuplicateLettersOption},
			expected: "Helo World!",
		},
		{
			name:     "remove duplicates with multiple repeats",
			original: "Booookkeeeper",
			opts:     []message.TextOption{message.RemoveDuplicateLettersOption},
			expected: "Bokeper",
		},

		// Комбинация message.LowerOption + RemovePunctuation
		{
			name:     "lower + remove punctuation",
			original: "Hello, WORLD! How ARE you?",
			opts:     []message.TextOption{message.LowerOption, message.RemovePunctuationOption},
			expected: "hello world how are you",
		},

		// Комбинация message.LowerOption + message.RemoveDuplicateLettersOption
		{
			name:     "lower + remove duplicates",
			original: "Hello   WORLDD!!",
			opts:     []message.TextOption{message.LowerOption, message.RemoveDuplicateLettersOption},
			expected: "helo world!",
		},

		// Комбинация RemovePunctuation + message.RemoveDuplicateLettersOption
		{
			name:     "remove punctuation + remove duplicates",
			original: "Hello,,, World!!!",
			opts:     []message.TextOption{message.RemovePunctuationOption, message.RemoveDuplicateLettersOption},
			expected: "Helo World",
		},

		// Все три опции вместе
		{
			name:     "all options",
			original: "Hello,,, WORLDD!!! How ARE you???",
			opts:     []message.TextOption{message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption},
			expected: "helo world how are you",
		},

		// Тесты с невидимыми символами
		{
			name:     "with zero width spaces",
			original: "Hello\u200BWorld\u200C!",
			opts:     []message.TextOption{},
			expected: "HelloWorld!",
		},
		{
			name:     "with invisible characters and options",
			original: "H\u200Bello\u200C, WORLDD!!!",
			opts:     []message.TextOption{message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption},
			expected: "helo world",
		},

		// Тесты с русско-английской транслитерацией
		{
			name:     "mixed cyrillic and latin",
			original: "Русский text",
			opts:     []message.TextOption{},
			expected: "Русский text",
		},
		{
			name:     "homoglyph conversion",
			original: "cупер", // первая 'c' английская, остальные русские
			opts:     []message.TextOption{},
			expected: "супер", // все русские после конвертации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			text := &message.Text{Original: tt.original}
			result := text.Text(tt.opts...)

			if result != tt.expected {
				t.Errorf("Text() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func BenchmarkNormalizeText_AllOptions(b *testing.B) {
	//nolint:dupword
	bigText := `!!! ВНИМАНИЕ! ВНИМАНИЕ! ВНИМАНИЕ !!!

Компания "РОМАШКА-ЛТД" представляет НОВЫЙ продукт 2024 года!!!
Сегодня - 15.03.2024 - специальное предложение!!! 

!!!СПИСОК команд для работы с системой:
!start - запуск процесса
!stop  - остановка процесса  
!status - проверка статуса
!help  - помощь

Цены СУПЕР-НИЗКИЕ: всего 999.99 руб. вместо 1,500.00 руб.!!!
Акция действует до 30/04/2024.

Примеры использования:
1. "Привет, мир!!!" -> нормализация текста
2. Hello World!!! -> транслитерация
3. ПРИВЕТ!!! -> нижний регистр
4. балалайкаааа -> удаление дубликатов

Контакты: 
Email: info@romashka.ru
Телефон: +7 (495) 123-45-67
Адрес: Москва, ул. Пушкина, д. 10, офис 5.

!!!ВАЖНО: 
Скидка 50% на ВСЕ товары категории "ПРЕМИУМ"!!!
Доставка БЕСПЛАТНАЯ при заказе от 5000 руб.

Технические характеристики:
• Процессор: Intel Core i7-12700K
• Память: 32 ГБ DDR4
• SSD: 1 ТБ NVMe
• Видеокарта: NVIDIA GeForce RTX 4070

Отзывы покупателей:
"Отличный товар!!! Быстрая доставка."
"Качество на высоте! Рекомендую!!!"
"Супер!!! Лучшее приобретение за последнее время."

!!!НЕ ПРОПУСТИТЕ!!! 
Только сегодня и только сейчас!!!
Уникальное предложение от "РОМАШКА-ЛТД"!!!

P.S. Следите за обновлениями в нашем телеграм-канале: @romashka_news
И подписывайтесь на рассылку!!!`
	text := &message.Text{Original: bigText}

	b.ResetTimer()
	for range b.N {
		text.Text(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption)
	}
}

func BenchmarkNormalizeText_NoOptions(b *testing.B) {
	//nolint:dupword
	bigText := `!!! ВНИМАНИЕ! ВНИМАНИЕ! ВНИМАНИЕ !!!

Компания "РОМАШКА-ЛТД" представляет НОВЫЙ продукт 2024 года!!!
Сегодня - 15.03.2024 - специальное предложение!!! 

!!!СПИСОК команд для работы с системой:
!start - запуск процесса
!stop  - остановка процесса  
!status - проверка статуса
!help  - помощь

Цены СУПЕР-НИЗКИЕ: всего 999.99 руб. вместо 1,500.00 руб.!!!
Акция действует до 30/04/2024.

Примеры использования:
1. "Привет, мир!!!" -> нормализация текста
2. Hello World!!! -> транслитерация
3. ПРИВЕТ!!! -> нижний регистр
4. балалайкаааа -> удаление дубликатов

Контакты: 
Email: info@romashka.ru
Телефон: +7 (495) 123-45-67
Адрес: Москва, ул. Пушкина, д. 10, офис 5.

!!!ВАЖНО: 
Скидка 50% на ВСЕ товары категории "ПРЕМИУМ"!!!
Доставка БЕСПЛАТНАЯ при заказе от 5000 руб.

Технические характеристики:
• Процессор: Intel Core i7-12700K
• Память: 32 ГБ DDR4
• SSD: 1 ТБ NVMe
• Видеокарта: NVIDIA GeForce RTX 4070

Отзывы покупателей:
"Отличный товар!!! Быстрая доставка."
"Качество на высоте! Рекомендую!!!"
"Супер!!! Лучшее приобретение за последнее время."

!!!НЕ ПРОПУСТИТЕ!!! 
Только сегодня и только сейчас!!!
Уникальное предложение от "РОМАШКА-ЛТД"!!!

P.S. Следите за обновлениями в нашем телеграм-канале: @romashka_news
И подписывайтесь на рассылку!!!`
	text := &message.Text{Original: bigText}

	b.ResetTimer()
	for range b.N {
		text.Text()
	}
}
