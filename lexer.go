package have

import (
	"fmt"
	"unicode"

	goscanner "go/scanner"
	gotoken "go/token"
)

type TokenType int

type Token struct {
	Type   TokenType
	Offset int
	Value  interface{}
}

// Tells if a token is any of the comparison operators.
func (t *Token) IsCompOp() bool {
	switch t.Type {
	case TOKEN_EQUALS, TOKEN_NEQUALS,
		TOKEN_LT, TOKEN_GT,
		TOKEN_EQ_LT, TOKEN_EQ_GT:
		return true
	}
	return false
}

// Tells if a token is any of the order operators.
func (t *Token) IsOrderOp() bool {
	switch t.Type {
	case TOKEN_LT, TOKEN_GT,
		TOKEN_EQ_LT, TOKEN_EQ_GT:
		return true
	}
	return false
}

// Tells if operator's operands can only be boolean.
func (t *Token) IsLogicalOp() bool {
	switch t.Type {
	case TOKEN_AND, TOKEN_OR:
		return true
	}
	return false
}

//go:generate stringer -type=TokenType
const (
	TOKEN_EOF          TokenType = iota + 1
	TOKEN_INDENT                 // indent - []rune of whitespace characters
	TOKEN_FOR                    // the "for" keyword
	TOKEN_WORD                   // alphanumeric word, starts witn a letter
	TOKEN_ASSIGN                 // =
	TOKEN_EQUALS                 // ==
	TOKEN_NEQUALS                // !=
	TOKEN_GT                     // >
	TOKEN_LT                     // <
	TOKEN_EQ_LT                  // <=
	TOKEN_EQ_GT                  // >=
	TOKEN_NEGATE                 // !
	TOKEN_NUM                    // general token for all number literals
	TOKEN_STR                    // string literal
	TOKEN_DOT                    // .
	TOKEN_LPARENTH               // (
	TOKEN_RPARENTH               // )
	TOKEN_LBRACKET               // [
	TOKEN_RBRACKET               // ]
	TOKEN_LBRACE                 // {
	TOKEN_RBRACE                 // }
	TOKEN_PLUS                   // +
	TOKEN_PLUS_ASSIGN            // +=
	TOKEN_INCREMENT              // ++
	TOKEN_MINUS                  // -
	TOKEN_MINUS_ASSIGN           // -=
	TOKEN_DECREMENT              // --
	TOKEN_VAR                    // the "var" keyword
	TOKEN_IF                     // the "if" keyword
	TOKEN_ELSE                   // the "else" keyword
	TOKEN_ELIF                   // the "elif" keyword
	TOKEN_SWITCH                 // the "switch" keyword
	TOKEN_CASE                   // the "case" keyword
	TOKEN_RETURN                 // the "return" keyword
	TOKEN_TRUE                   // the "true" keyword
	TOKEN_FALSE                  // the "false" keyword
	TOKEN_STRUCT                 // the "struct" keyword
	TOKEN_MAP                    // the "map" keyword
	TOKEN_FUNC                   // the "func" keyword
	TOKEN_TYPE                   // the "type" keyword
	TOKEN_IN                     // the "in" keyword
	TOKEN_PASS                   // the "pass" keyword
	TOKEN_PACKAGE                // the "package" keyword
	TOKEN_BREAK                  // the "break" keyword
	TOKEN_CONTINUE               // the "continue" keyword
	TOKEN_FALLTHROUGH            // the "fallthrough" keyword
	TOKEN_GOTO                   // the "goto" keyword
	TOKEN_INTERFACE              // the "interface" keyword
	TOKEN_NIL                    // the "nil" keyword
	TOKEN_MUL                    // *
	TOKEN_DIV                    // /
	TOKEN_MUL_ASSIGN             // *=
	TOKEN_DIV_ASSIGN             // /=
	TOKEN_SHL                    // <<
	TOKEN_SHR                    // >>
	TOKEN_SEND                   // <-
	TOKEN_COMMA                  // ,
	TOKEN_COLON                  // :
	TOKEN_SEMICOLON              // ;
	TOKEN_AMP                    // &
	TOKEN_PIPE                   // |
	TOKEN_PERCENT                // %
	TOKEN_AND                    // &&
	TOKEN_OR                     // ||
)

type Lexer struct {
	// Characters not processed yet.
	buf []rune
	// Stack of opened indents.
	indentsStack []int
	// We don't want to emit indent tokens for blank lines,
	// so we need to postpone indent tokens for a while.
	tokenIndent *Token
	// How many characters we've processed.
	skipped int
	// Offset of currently processed token.
	curTokenPos int
}

func NewLexer(buf []rune) *Lexer {
	return &Lexer{buf: buf, indentsStack: []int{}}
}

// Advance lexer's buffer by skipping whitespace, except newlines.
func (l *Lexer) skipWhiteChars() []rune {
	i := 0
	for i < len(l.buf) && (unicode.IsSpace(l.buf[i]) && l.buf[i] != '\n') {
		i++
	}
	whitespace := l.buf[:i]
	l.skipBy(i)
	return whitespace
}

// Read an alphanumeric word from the buffer, advancing it.
func (l *Lexer) scanWord() []rune {
	i := 0
	for i < len(l.buf) && (unicode.IsLetter(l.buf[i]) || unicode.IsNumber(l.buf[i])) {
		i++
	}

	result := l.buf[:i]
	l.skipBy(i)
	return result
}

// Advance lexer's buffer by one character.
func (l *Lexer) skip() {
	l.skipped++
	l.buf = l.buf[1:]
}

// Advance lexer's buffer by N characters.
func (l *Lexer) skipBy(n int) {
	l.skipped += n
	l.buf = l.buf[n:]
}

// Tells if we've reached the end of the buffer.
func (l *Lexer) isEnd() bool {
	return len(l.buf) == 0
}

// Check which token is currently at the beginning of the buffer.
// Can be used to decide between tokens with the same beginning, e.g.
// "=", "==", "=<", ">=".
// Returns the first token matched, NOT the longest one, so order matters.
// E.g. instead of "=", "=="; rather use "==", "=".
func (l *Lexer) checkAlt(alts ...string) (alt string, ok bool) {
	for _, alt := range alts {
		if string(l.buf[:len(alt)]) == alt {
			l.skipBy(len(alt))
			return alt, true
		}
	}
	return "", false
}

func (l *Lexer) loadEscapedString() (string, error) {
	if len(l.buf) == 0 || l.buf[0] != '"' {
		return "", fmt.Errorf("String literal has to start with a double quote")
	}

	l.skip()

	i := 0
	for ; i < len(l.buf); i++ {
		switch l.buf[i] {
		case '\\':
			i++
			if i == len(l.buf) {
				return "", fmt.Errorf("Unexpected file end - middle of a string literal")
			}
		case '"':
			s := string(l.buf[:i])
			l.skipBy(i + 1)
			return s, nil
		}
	}
	return "", fmt.Errorf("Unterminated string literal")
}

func (l *Lexer) newToken(typ TokenType, val interface{}) *Token {
	return &Token{Type: typ, Offset: l.curTokenPos, Value: val}
}

// A convenience wrapper for newToken, handy in situations when a token
// is created just to be immediately returned with a nil error.
func (l *Lexer) retNewToken(typ TokenType, val interface{}) (*Token, error) {
	return l.newToken(typ, val), nil
}

func (l *Lexer) scanGoToken() (token gotoken.Token, lit string, err error) {
	// TODO: We shouldn't be setting everything up from scratch every time.

	fs := gotoken.NewFileSet()
	code := make([]byte, len(l.buf))

	// TODO: Don't use []rune, if Golang doesn't need it neither do we and it leads
	// to stuff like this.
	for i := 0; i < len(l.buf); i++ {
		code[i] = byte(l.buf[i])
	}

	f := fs.AddFile("", fs.Base(), len(code))
	s := &goscanner.Scanner{}

	errorHandler := func(pos gotoken.Position, msg string) {
		err = fmt.Errorf("Scanner error: %s, %s", pos, msg)
	}

	s.Init(f, []byte(code), errorHandler, 0)
	_, tok, lit := s.Scan()
	l.skipBy(len(lit))

	return tok, lit, err
}

func (l *Lexer) fromGoToken(token gotoken.Token, lit string) (*Token, error) {
	switch token {
	case gotoken.INT, gotoken.FLOAT, gotoken.IMAG:
		// TODO: Don't lump everything together
		return l.retNewToken(TOKEN_NUM, lit)
	}
	return nil, fmt.Errorf("Unexpected Go token: %s", token)
}

func (l *Lexer) Next() (*Token, error) {
	l.curTokenPos = l.skipped

	if l.isEnd() || l.buf[0] != '\n' {
		if l.tokenIndent != nil {
			t := l.tokenIndent
			l.tokenIndent = nil
			return t, nil
		}
	}

	if l.isEnd() {
		return l.retNewToken(TOKEN_EOF, nil)
	}

	ch := l.buf[0]

	switch {
	case ch == '\n':
		l.skip()
		indent := string(l.skipWhiteChars())
		l.tokenIndent = l.newToken(TOKEN_INDENT, indent)
		return l.Next()
	case unicode.IsSpace(ch):
		l.skipWhiteChars()
		return l.Next()
	case unicode.IsLetter(ch):
		word := l.scanWord()
		switch s := string(word); s {
		case "for":
			return l.retNewToken(TOKEN_FOR, nil)
		case "pass":
			return l.retNewToken(TOKEN_PASS, nil)
		case "package":
			return l.retNewToken(TOKEN_PACKAGE, nil)
		case "var":
			return l.retNewToken(TOKEN_VAR, nil)
		case "if":
			return l.retNewToken(TOKEN_IF, nil)
		case "else":
			return l.retNewToken(TOKEN_ELSE, nil)
		case "elif":
			return l.retNewToken(TOKEN_ELIF, nil)
		case "switch":
			return l.retNewToken(TOKEN_SWITCH, nil)
		case "case":
			return l.retNewToken(TOKEN_CASE, nil)
		case "return", "ret":
			return l.retNewToken(TOKEN_RETURN, nil)
		case "true":
			return l.retNewToken(TOKEN_TRUE, nil)
		case "false":
			return l.retNewToken(TOKEN_FALSE, nil)
		case "struct":
			return l.retNewToken(TOKEN_STRUCT, nil)
		case "interface":
			return l.retNewToken(TOKEN_INTERFACE, nil)
		case "map":
			return l.retNewToken(TOKEN_MAP, nil)
		case "func":
			return l.retNewToken(TOKEN_FUNC, nil)
		case "type":
			return l.retNewToken(TOKEN_TYPE, nil)
		case "break":
			return l.retNewToken(TOKEN_BREAK, nil)
		case "continue":
			return l.retNewToken(TOKEN_CONTINUE, nil)
		case "fallthrough":
			return l.retNewToken(TOKEN_FALLTHROUGH, nil)
		case "goto":
			return l.retNewToken(TOKEN_GOTO, nil)
		case "nil":
			return l.retNewToken(TOKEN_NIL, nil)
		default:
			return l.retNewToken(TOKEN_WORD, s)
		}
	case ch == '=':
		alt, _ := l.checkAlt("==", "=")
		switch alt {
		case "=":
			return l.retNewToken(TOKEN_ASSIGN, alt)
		case "==":
			return l.retNewToken(TOKEN_EQUALS, alt)
		}
	case ch == '!':
		alt, _ := l.checkAlt("!=", "!")
		switch alt {
		case "!=":
			return l.retNewToken(TOKEN_NEQUALS, alt)
		case "!":
			return l.retNewToken(TOKEN_NEGATE, alt)
		}
	case ch == '+':
		alt, _ := l.checkAlt("++", "+=", "+")
		switch alt {
		case "+":
			return l.retNewToken(TOKEN_PLUS, alt)
		case "+=":
			return l.retNewToken(TOKEN_PLUS_ASSIGN, alt)
		case "++":
			return l.retNewToken(TOKEN_INCREMENT, alt)
		}
	case ch == '-':
		alt, _ := l.checkAlt("--", "-=", "-")
		switch alt {
		case "-":
			return l.retNewToken(TOKEN_MINUS, alt)
		case "-=":
			return l.retNewToken(TOKEN_MINUS_ASSIGN, alt)
		case "--":
			return l.retNewToken(TOKEN_DECREMENT, alt)
		}
	case ch == '<':
		alt, _ := l.checkAlt("<<", "<-", "<=", "<")
		switch alt {
		case "<":
			return l.retNewToken(TOKEN_LT, alt)
		case "<-":
			return l.retNewToken(TOKEN_SEND, alt)
		case "<<":
			return l.retNewToken(TOKEN_SHL, alt)
		case "<=":
			return l.retNewToken(TOKEN_EQ_LT, alt)
		}
	case ch == '>':
		alt, _ := l.checkAlt(">>", ">=", ">")
		switch alt {
		case ">":
			return l.retNewToken(TOKEN_GT, alt)
		case ">>":
			return l.retNewToken(TOKEN_SHR, alt)
		case ">=":
			return l.retNewToken(TOKEN_EQ_GT, alt)
		}
	case unicode.IsNumber(ch):
		gotok, lit, err := l.scanGoToken()
		if err != nil {
			return nil, err
		}
		return l.fromGoToken(gotok, lit)
	case ch == '"':
		str, err := l.loadEscapedString()
		if err != nil {
			return nil, err
		}
		return l.retNewToken(TOKEN_STR, str)
	case ch == '(':
		l.skip()
		return l.retNewToken(TOKEN_LPARENTH, nil)
	case ch == ')':
		l.skip()
		return l.retNewToken(TOKEN_RPARENTH, nil)
	case ch == '[':
		l.skip()
		return l.retNewToken(TOKEN_LBRACKET, nil)
	case ch == ']':
		l.skip()
		return l.retNewToken(TOKEN_RBRACKET, nil)
	case ch == '{':
		l.skip()
		return l.retNewToken(TOKEN_LBRACE, nil)
	case ch == '}':
		l.skip()
		return l.retNewToken(TOKEN_RBRACE, nil)
	case ch == '.':
		l.skip()
		return l.retNewToken(TOKEN_DOT, nil)
	case ch == '*':
		alt, _ := l.checkAlt("*=", "*")
		switch alt {
		case "*":
			return l.retNewToken(TOKEN_MUL, alt)
		case "*=":
			return l.retNewToken(TOKEN_MUL_ASSIGN, alt)
		}
	case ch == '/':
		alt, _ := l.checkAlt("/=", "/")
		switch alt {
		case "/":
			return l.retNewToken(TOKEN_DIV, alt)
		case "/=":
			return l.retNewToken(TOKEN_DIV_ASSIGN, alt)
		}
	case ch == ',':
		l.skip()
		return l.retNewToken(TOKEN_COMMA, nil)
	case ch == ';':
		l.skip()
		return l.retNewToken(TOKEN_SEMICOLON, nil)
	case ch == ':':
		l.skip()
		return l.retNewToken(TOKEN_COLON, nil)
	case ch == '%':
		l.skip()
		return l.retNewToken(TOKEN_PERCENT, "%")
	case ch == '&':
		alt, _ := l.checkAlt("&&", "&")
		switch alt {
		case "&&":
			return l.retNewToken(TOKEN_AND, alt)
		case "&":
			return l.retNewToken(TOKEN_AMP, alt)
		}
	case ch == '|':
		alt, _ := l.checkAlt("||", "|")
		switch alt {
		case "||":
			return l.retNewToken(TOKEN_OR, alt)
		case "|":
			return l.retNewToken(TOKEN_PIPE, alt)
		}
	}

	return nil, fmt.Errorf("Don't know what to do, '%c'", ch)
}
