package domain

import "testing"

func TestRemoveInvisibleRune(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no invisible characters",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "visible punctuation should remain",
			input:    "comeback v vlrnt???@??#@?#?",
			expected: "comeback v vlrnt???@??#@?#?",
		},
		{
			name:     "zero width space",
			input:    "hello\u200Bworld",
			expected: "helloworld",
		},
		{
			name:     "multiple zero width characters",
			input:    "test\u200C\u200D\u200Estring",
			expected: "teststring",
		},
		{
			name:     "control characters",
			input:    "line\u0001break\u0002test",
			expected: "linebreaktest",
		},
		{
			name:     "BOM character",
			input:    "\uFEFFhello world",
			expected: "hello world",
		},
		{
			name:     "mixed visible and invisible",
			input:    "normal\u2060text\u2061with\u2062invisible",
			expected: "normaltextwithinvisible",
		},
		{
			name:     "bidirectional controls",
			input:    "left\u202Ato\u202Bright",
			expected: "lefttoright",
		},
		{
			name:     "variation selectors",
			input:    "base\uFE00char\uFE0Ftest",
			expected: "basechartest",
		},
		{
			name:     "all invisible characters",
			input:    "\u200B\u200C\u200D\u200E\u200F",
			expected: "",
		},
		{
			name:     "numbers and letters remain",
			input:    "abc123\u200Bdef456",
			expected: "abc123def456",
		},
		{
			name:     "spaces remain",
			input:    "hello\u200B world\u200C !",
			expected: "hello world !",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeInvisibleRune(tt.input)
			if result != tt.expected {
				t.Errorf("removeInvisibleRune(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsInvisibleRune_VisibleCharacters(t *testing.T) {
	// Test that common visible characters are NOT considered invisible
	visibleChars := []rune{
		'a', 'Z', '0', '9',
		'?', '@', '#', '!', '.', ',',
		' ', '$', '%', '&', '*', '(', ')',
		'+', '-', '=', '/', '\\',
	}

	for _, r := range visibleChars {
		if isInvisibleRune(r) {
			t.Errorf("Visible character %c (U+%04X) should not be considered invisible", r, r)
		}
	}
}

func TestIsInvisibleRune_InvisibleCharacters(t *testing.T) {
	// Test that known invisible characters are correctly identified
	invisibleChars := []rune{
		'\u200B', // ZERO WIDTH SPACE
		'\u200C', // ZERO WIDTH NON-JOINER
		'\u200D', // ZERO WIDTH JOINER
		'\u200E', // LEFT-TO-RIGHT MARK
		'\u200F', // RIGHT-TO-LEFT MARK
		'\uFEFF', // ZERO WIDTH NO-BREAK SPACE
		'\u0001', // Control character
		'\u001F', // Control character
		'\u0080', // C1 control
		'\u2060', // WORD JOINER
	}

	for _, r := range invisibleChars {
		if !isInvisibleRune(r) {
			t.Errorf("Invisible character U+%04X should be considered invisible", r)
		}
	}
}
