package have

import (
	"fmt"
	"reflect"
	"testing"
)

func testTokens(t *testing.T, input []rune, output []*Token) {
	l := NewLexer(input)
	for _, expected := range output {
		token, err := l.Next()
		if err != nil {
			fmt.Printf("Non-nil error %v\n", err)
			t.Fail()
		}
		if !reflect.DeepEqual(token, expected) {
			fmt.Printf("Received %v instead of %v\n", token, expected)
			t.Fail()
		}
	}
}

func TestIndents(t *testing.T) {
	testTokens(t, []rune(""), []*Token{&Token{TOKEN_EOF, nil}})
	testTokens(t, []rune("\n  for"), []*Token{
		&Token{TOKEN_NEWSCOPE, nil},
		&Token{TOKEN_FOR, nil},
		&Token{TOKEN_ENDSCOPE, nil},
		&Token{TOKEN_EOF, nil}})

	s := `
  for test
    for
    frog
`

	testTokens(t, []rune(s), []*Token{
		&Token{TOKEN_NEWSCOPE, nil},
		&Token{TOKEN_FOR, nil},
		&Token{TOKEN_WORD, "test"},
		&Token{TOKEN_NEWSCOPE, nil},
		&Token{TOKEN_FOR, nil},
		&Token{TOKEN_WORD, "frog"},
		&Token{TOKEN_ENDSCOPE, nil},
		&Token{TOKEN_ENDSCOPE, nil},
		&Token{TOKEN_EOF, nil},
	})

	testTokens(t, []rune("for"), []*Token{
		&Token{TOKEN_FOR, nil},
		&Token{TOKEN_EOF, nil}})
}
