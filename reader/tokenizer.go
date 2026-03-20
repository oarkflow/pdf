package reader

import (
	"fmt"
	"strconv"
	"strings"
)

// TokenType identifies the kind of PDF token.
type TokenType int

const (
	TokenInteger    TokenType = iota
	TokenReal
	TokenString
	TokenHexString
	TokenName
	TokenKeyword
	TokenArrayBegin // [
	TokenArrayEnd   // ]
	TokenDictBegin  // <<
	TokenDictEnd    // >>
	TokenEOF
)

// Token represents a single lexical token from a PDF byte stream.
type Token struct {
	Type  TokenType
	Value string
	Int   int64
	Real  float64
	Pos   int64 // byte offset where the token starts
}

// Tokenizer is a PDF lexer that reads tokens from raw PDF bytes.
type Tokenizer struct {
	data []byte
	pos  int
}

// NewTokenizer creates a new Tokenizer over the given data.
func NewTokenizer(data []byte) *Tokenizer {
	return &Tokenizer{data: data}
}

// Pos returns the current byte offset.
func (t *Tokenizer) Pos() int {
	return t.pos
}

// Seek sets the current byte offset.
func (t *Tokenizer) Seek(pos int) {
	t.pos = pos
}

// Peek returns the next token without advancing.
func (t *Tokenizer) Peek() (Token, error) {
	saved := t.pos
	tok, err := t.Next()
	t.pos = saved
	return tok, err
}

func isWhitespace(c byte) bool {
	switch c {
	case 0, 9, 10, 12, 13, 32:
		return true
	}
	return false
}

func isDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

func isRegular(c byte) bool {
	return !isWhitespace(c) && !isDelimiter(c)
}

// skipWhitespaceAndComments advances past whitespace and % comments.
func (t *Tokenizer) skipWhitespaceAndComments() {
	for t.pos < len(t.data) {
		c := t.data[t.pos]
		if isWhitespace(c) {
			t.pos++
			continue
		}
		if c == '%' {
			// Skip until end of line.
			for t.pos < len(t.data) && t.data[t.pos] != '\n' && t.data[t.pos] != '\r' {
				t.pos++
			}
			continue
		}
		break
	}
}

// Next reads and returns the next token.
func (t *Tokenizer) Next() (Token, error) {
	t.skipWhitespaceAndComments()

	if t.pos >= len(t.data) {
		return Token{Type: TokenEOF, Pos: int64(t.pos)}, nil
	}

	startPos := int64(t.pos)
	c := t.data[t.pos]

	switch c {
	case '[':
		t.pos++
		return Token{Type: TokenArrayBegin, Value: "[", Pos: startPos}, nil
	case ']':
		t.pos++
		return Token{Type: TokenArrayEnd, Value: "]", Pos: startPos}, nil
	case '<':
		if t.pos+1 < len(t.data) && t.data[t.pos+1] == '<' {
			t.pos += 2
			return Token{Type: TokenDictBegin, Value: "<<", Pos: startPos}, nil
		}
		return t.readHexString(startPos)
	case '>':
		if t.pos+1 < len(t.data) && t.data[t.pos+1] == '>' {
			t.pos += 2
			return Token{Type: TokenDictEnd, Value: ">>", Pos: startPos}, nil
		}
		return Token{}, fmt.Errorf("unexpected '>' at offset %d", t.pos)
	case '(':
		return t.readLiteralString(startPos)
	case '/':
		return t.readName(startPos)
	}

	// Number or keyword.
	return t.readNumberOrKeyword(startPos)
}

func (t *Tokenizer) readLiteralString(startPos int64) (Token, error) {
	t.pos++ // skip opening '('
	var buf strings.Builder
	depth := 1
	for t.pos < len(t.data) {
		c := t.data[t.pos]
		if c == '\\' {
			t.pos++
			if t.pos >= len(t.data) {
				return Token{}, fmt.Errorf("unexpected end of string escape at offset %d", t.pos)
			}
			esc := t.data[t.pos]
			t.pos++
			switch esc {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case '(':
				buf.WriteByte('(')
			case ')':
				buf.WriteByte(')')
			case '\\':
				buf.WriteByte('\\')
			case '\r':
				// line continuation: \<CR> or \<CR><LF>
				if t.pos < len(t.data) && t.data[t.pos] == '\n' {
					t.pos++
				}
			case '\n':
				// line continuation
			default:
				// Octal escape.
				if esc >= '0' && esc <= '7' {
					oct := int(esc - '0')
					for i := 0; i < 2 && t.pos < len(t.data) && t.data[t.pos] >= '0' && t.data[t.pos] <= '7'; i++ {
						oct = oct*8 + int(t.data[t.pos]-'0')
						t.pos++
					}
					buf.WriteByte(byte(oct))
				} else {
					buf.WriteByte(esc)
				}
			}
			continue
		}
		if c == '(' {
			depth++
			buf.WriteByte(c)
			t.pos++
			continue
		}
		if c == ')' {
			depth--
			if depth == 0 {
				t.pos++
				return Token{Type: TokenString, Value: buf.String(), Pos: startPos}, nil
			}
			buf.WriteByte(c)
			t.pos++
			continue
		}
		buf.WriteByte(c)
		t.pos++
	}
	return Token{}, fmt.Errorf("unterminated string starting at offset %d", startPos)
}

func (t *Tokenizer) readHexString(startPos int64) (Token, error) {
	t.pos++ // skip '<'
	var buf strings.Builder
	for t.pos < len(t.data) {
		c := t.data[t.pos]
		if c == '>' {
			t.pos++
			// Decode hex pairs.
			hex := buf.String()
			hex = strings.ReplaceAll(hex, " ", "")
			hex = strings.ReplaceAll(hex, "\n", "")
			hex = strings.ReplaceAll(hex, "\r", "")
			hex = strings.ReplaceAll(hex, "\t", "")
			return Token{Type: TokenHexString, Value: hex, Pos: startPos}, nil
		}
		buf.WriteByte(c)
		t.pos++
	}
	return Token{}, fmt.Errorf("unterminated hex string starting at offset %d", startPos)
}

func (t *Tokenizer) readName(startPos int64) (Token, error) {
	t.pos++ // skip '/'
	var buf strings.Builder
	for t.pos < len(t.data) {
		c := t.data[t.pos]
		if isWhitespace(c) || isDelimiter(c) {
			break
		}
		if c == '#' && t.pos+2 < len(t.data) {
			hi := unhex(t.data[t.pos+1])
			lo := unhex(t.data[t.pos+2])
			if hi >= 0 && lo >= 0 {
				buf.WriteByte(byte(hi<<4 | lo))
				t.pos += 3
				continue
			}
		}
		buf.WriteByte(c)
		t.pos++
	}
	return Token{Type: TokenName, Value: buf.String(), Pos: startPos}, nil
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

func (t *Tokenizer) readNumberOrKeyword(startPos int64) (Token, error) {
	start := t.pos
	for t.pos < len(t.data) && isRegular(t.data[t.pos]) {
		t.pos++
	}
	word := string(t.data[start:t.pos])
	if word == "" {
		return Token{}, fmt.Errorf("unexpected byte 0x%02x at offset %d", t.data[start], start)
	}

	// Try integer.
	if i, err := strconv.ParseInt(word, 10, 64); err == nil {
		return Token{Type: TokenInteger, Value: word, Int: i, Pos: startPos}, nil
	}

	// Try real.
	if f, err := strconv.ParseFloat(word, 64); err == nil {
		// Only treat as real if it contains a dot or exponent.
		if strings.ContainsAny(word, ".eE") {
			return Token{Type: TokenReal, Value: word, Real: f, Pos: startPos}, nil
		}
	}

	// Keyword.
	return Token{Type: TokenKeyword, Value: word, Pos: startPos}, nil
}
