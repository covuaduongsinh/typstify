package vietnamese

import (
	"testing"
)

// typeWord simulates typing characters one by one and applying Telex transformations.
func typeWord(input string) string {
	var current string
	for _, ch := range input {
		if transformed, ok := TransformWordTelex(current, ch); ok {
			current = transformed
		} else {
			current += string(ch)
		}
	}
	return current
}

func TestTelexBasicVowels(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"aa", "â"},
		{"aw", "ă"},
		{"ee", "ê"},
		{"oo", "ô"},
		{"ow", "ơ"},
		{"uw", "ư"},
		{"dd", "đ"},
		{"DD", "Đ"},
		{"Dd", "Đ"},
		{"w", "ư"},
		{"W", "Ư"},
	}

	for _, tt := range tests {
		got := typeWord(tt.input)
		if got != tt.expected {
			t.Errorf("typeWord(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestTelexTones(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"as", "á"},
		{"af", "à"},
		{"ar", "ả"},
		{"ax", "ã"},
		{"aj", "ạ"},
		{"az", "a"},
		{"ees", "ế"},
		{"eef", "ề"},
		{"eer", "ể"},
		{"eex", "ễ"},
		{"eej", "ệ"},
		{"aws", "ắ"},
		{"awf", "ằ"},
		{"awr", "ẳ"},
		{"awx", "ẵ"},
		{"awj", "ặ"},
		{"oos", "ố"},
		{"oof", "ồ"},
		{"oor", "ổ"},
		{"oox", "ỗ"},
		{"ooj", "ộ"},
		{"ows", "ớ"},
		{"owf", "ờ"},
		{"owr", "ở"},
		{"owx", "ỡ"},
		{"owj", "ợ"},
		{"uws", "ứ"},
		{"uwf", "ừ"},
		{"uwr", "ử"},
		{"uwx", "ữ"},
		{"uwj", "ự"},
	}

	for _, tt := range tests {
		got := typeWord(tt.input)
		if got != tt.expected {
			t.Errorf("typeWord(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestTelexWords(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tiengs", "tiếng"},
		{"vietj", "việt"},
		{"tawng", "tăng"},
		{"kichs", "kích"},
		{"thuwocs", "thước"},
		{"banf", "bàn"},
		{"cowf", "cờ"},
		{"owr", "ở"},
		{"duowis", "dưới"},
		{"bawfng", "bằng"},
		{"vowis", "với"},
		{"treen", "trên"},
		{"cho", "cho"},
		{"ddoongf", "đồng"},
		{"booj", "bộ"},
		{"nguwowif", "người"},
		{"hoaf", "hoà"},
		{"hoas", "hoá"},
		{"thuys", "thuý"},
		{"thuyr", "thuỷ"},
		{"toans", "toán"},
		{"duyeetj", "duyệt"},
	}

	for _, tt := range tests {
		got := typeWord(tt.input)
		if got != tt.expected {
			t.Errorf("typeWord(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestTelexCapitalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Vietj", "Việt"},
		{"VIETJ", "VIỆT"},
		{"DDOONGF", "ĐỒNG"},
		{"BOOJ", "BỘ"},
		{"Thuowcs", "Thước"},
	}

	for _, tt := range tests {
		got := typeWord(tt.input)
		if got != tt.expected {
			t.Errorf("typeWord(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestTelexDoubleEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ddd", "dd"},
		{"aaa", "aa"},
		{"ass", "as"},
	}

	for _, tt := range tests {
		got := typeWord(tt.input)
		if got != tt.expected {
			t.Errorf("typeWord(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func typeSentence(input string) string {
	var fullText string
	for _, ch := range input {
		start, lastWord := ExtractLastWord(fullText)
		if transformed, ok := TransformWordTelex(lastWord, ch); ok {
			runes := []rune(fullText)
			fullText = string(runes[:start]) + transformed
		} else {
			fullText += string(ch)
		}
	}
	return fullText
}

func TestTelexSentence(t *testing.T) {
	input := "tawng kichs thuwocs 2 banf cowf owr duowis bawfng vowis banf cowf owr treen cho ddoongf booj"
	expected := "tăng kích thước 2 bàn cờ ở dưới bằng với bàn cờ ở trên cho đồng bộ"
	got := typeSentence(input)
	if got != expected {
		t.Errorf("typeSentence(%q) = %q; want %q", input, got, expected)
	}
}

