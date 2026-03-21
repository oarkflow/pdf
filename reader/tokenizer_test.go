package reader

import (
	"testing"
)

func TestTokenizeInteger(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"42", 42},
		{"-10", -10},
		{"0", 0},
		{"999999", 999999},
	}
	for _, tt := range tests {
		tok := NewTokenizer([]byte(tt.input))
		token, err := tok.Next()
		if err != nil {
			t.Fatalf("input %q: %v", tt.input, err)
		}
		if token.Type != TokenInteger {
			t.Errorf("input %q: type = %d, want TokenInteger", tt.input, token.Type)
		}
		if token.Int != tt.want {
			t.Errorf("input %q: Int = %d, want %d", tt.input, token.Int, tt.want)
		}
	}
}

func TestTokenizeReal(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"3.14", 3.14},
		{"-1.5", -1.5},
		{"0.0", 0.0},
	}
	for _, tt := range tests {
		tok := NewTokenizer([]byte(tt.input))
		token, err := tok.Next()
		if err != nil {
			t.Fatal(err)
		}
		if token.Type != TokenReal {
			t.Errorf("input %q: type = %d, want TokenReal", tt.input, token.Type)
		}
		if token.Real != tt.want {
			t.Errorf("input %q: Real = %f, want %f", tt.input, token.Real, tt.want)
		}
	}
}

func TestTokenizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Type", "Type"},
		{"/", ""},
		{"/A#20B", "A B"},
	}
	for _, tt := range tests {
		tok := NewTokenizer([]byte(tt.input))
		token, err := tok.Next()
		if err != nil {
			t.Fatal(err)
		}
		if token.Type != TokenName {
			t.Errorf("input %q: type = %d, want TokenName", tt.input, token.Type)
		}
		if token.Value != tt.want {
			t.Errorf("input %q: Value = %q, want %q", tt.input, token.Value, tt.want)
		}
	}
}

func TestTokenizeLiteralString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"(Hello)", "Hello"},
		{"()", ""},
		{"(a\\(b\\)c)", "a(b)c"},
		{"(line\\nfeed)", "line\nfeed"},
		{"(nested(paren)ok)", "nested(paren)ok"},
		{"(esc\\\\back)", "esc\\back"},
		{"(tab\\there)", "tab\there"},
	}
	for _, tt := range tests {
		tok := NewTokenizer([]byte(tt.input))
		token, err := tok.Next()
		if err != nil {
			t.Fatalf("input %q: %v", tt.input, err)
		}
		if token.Type != TokenString {
			t.Errorf("input %q: type = %d, want TokenString", tt.input, token.Type)
		}
		if token.Value != tt.want {
			t.Errorf("input %q: Value = %q, want %q", tt.input, token.Value, tt.want)
		}
	}
}

func TestTokenizeHexString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<48656C6C6F>", "48656C6C6F"},
		{"<>", ""},
		{"<AB CD>", "ABCD"},
	}
	for _, tt := range tests {
		tok := NewTokenizer([]byte(tt.input))
		token, err := tok.Next()
		if err != nil {
			t.Fatal(err)
		}
		if token.Type != TokenHexString {
			t.Errorf("input %q: type = %d, want TokenHexString", tt.input, token.Type)
		}
		if token.Value != tt.want {
			t.Errorf("input %q: Value = %q, want %q", tt.input, token.Value, tt.want)
		}
	}
}

func TestTokenizeArrayDelimiters(t *testing.T) {
	tok := NewTokenizer([]byte("[1 2]"))
	t1, _ := tok.Next()
	if t1.Type != TokenArrayBegin {
		t.Error("expected [")
	}
	tok.Next() // 1
	tok.Next() // 2
	t4, _ := tok.Next()
	if t4.Type != TokenArrayEnd {
		t.Error("expected ]")
	}
}

func TestTokenizeDictDelimiters(t *testing.T) {
	tok := NewTokenizer([]byte("<< /Type /Catalog >>"))
	t1, _ := tok.Next()
	if t1.Type != TokenDictBegin {
		t.Error("expected <<")
	}
	tok.Next() // /Type
	tok.Next() // /Catalog
	t4, _ := tok.Next()
	if t4.Type != TokenDictEnd {
		t.Error("expected >>")
	}
}

func TestTokenizeKeywords(t *testing.T) {
	for _, kw := range []string{"true", "false", "null", "obj", "endobj", "stream", "endstream", "xref", "trailer"} {
		tok := NewTokenizer([]byte(kw))
		token, _ := tok.Next()
		if token.Type != TokenKeyword {
			// true/false/null are integers? No, they parse as keywords
			if token.Type != TokenKeyword {
				t.Errorf("%q: type = %d, want TokenKeyword", kw, token.Type)
			}
		}
	}
}

func TestTokenizeComments(t *testing.T) {
	tok := NewTokenizer([]byte("% this is a comment\n42"))
	token, _ := tok.Next()
	if token.Type != TokenInteger || token.Int != 42 {
		t.Errorf("expected 42 after comment, got type=%d val=%q", token.Type, token.Value)
	}
}

func TestTokenizeEOF(t *testing.T) {
	tok := NewTokenizer([]byte(""))
	token, _ := tok.Next()
	if token.Type != TokenEOF {
		t.Error("expected EOF")
	}
}

func TestTokenizeWhitespace(t *testing.T) {
	tok := NewTokenizer([]byte("  \t\n\r  42"))
	token, _ := tok.Next()
	if token.Type != TokenInteger || token.Int != 42 {
		t.Error("whitespace not skipped properly")
	}
}

func TestTokenizePeek(t *testing.T) {
	tok := NewTokenizer([]byte("42 100"))
	peeked, _ := tok.Peek()
	if peeked.Int != 42 {
		t.Error("peek should return 42")
	}
	actual, _ := tok.Next()
	if actual.Int != 42 {
		t.Error("next after peek should still return 42")
	}
}

func TestTokenizeSeek(t *testing.T) {
	tok := NewTokenizer([]byte("hello 42"))
	tok.Seek(6)
	token, _ := tok.Next()
	if token.Type != TokenInteger || token.Int != 42 {
		t.Error("seek did not work")
	}
}

func TestTokenizeOctalEscape(t *testing.T) {
	tok := NewTokenizer([]byte("(\\101)")) // \101 = 'A'
	token, _ := tok.Next()
	if token.Value != "A" {
		t.Errorf("octal escape: got %q, want A", token.Value)
	}
}

func TestTokenizeUnterminatedString(t *testing.T) {
	tok := NewTokenizer([]byte("(unterminated"))
	_, err := tok.Next()
	if err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestTokenizeUnterminatedHexString(t *testing.T) {
	tok := NewTokenizer([]byte("<ABCD"))
	_, err := tok.Next()
	if err == nil {
		t.Error("expected error for unterminated hex string")
	}
}

func TestTokenizePos(t *testing.T) {
	tok := NewTokenizer([]byte("   42"))
	if tok.Pos() != 0 {
		t.Error("initial pos should be 0")
	}
	token, _ := tok.Next()
	if token.Pos != 3 {
		t.Errorf("token pos = %d, want 3", token.Pos)
	}
}
