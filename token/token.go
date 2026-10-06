// Package token defines the lexical tokens of the Turtle language.
package token

type Type string

type Token struct {
	Type    Type
	Literal string
	Line    int
	// SpaceBefore is true when whitespace (or a comment, or the start of
	// a line) comes right before this token. It tells "nums get -1" (a
	// negative argument) from "a b - 1" (subtraction).
	SpaceBefore bool
}

const (
	ILLEGAL Type = "ILLEGAL"
	EOF     Type = "EOF"

	IDENT  Type = "IDENT"
	INT    Type = "INT"
	FLOAT  Type = "FLOAT"
	STRING Type = "STRING"

	ASSIGN   Type = "="
	PLUS     Type = "+"
	MINUS    Type = "-"
	ASTERISK Type = "*"
	SLASH    Type = "/"
	DIV      Type = "DIV" // "7 div 2": division keeping the whole part
	PERCENT  Type = "%"

	LT     Type = "<"
	GT     Type = ">"
	LE     Type = "<="
	GE     Type = ">="
	EQ     Type = "=="
	NOT_EQ Type = "!="
	AND    Type = "&&"
	OR     Type = "||"
	BANG   Type = "!"

	INCR Type = "++"
	DECR Type = "--"

	COMMA  Type = ","
	PERIOD Type = "."
	COLON  Type = ":"
	SEMI   Type = ";"

	LBRACKET Type = "["
	RBRACKET Type = "]"
	LPAREN   Type = "("
	RPAREN   Type = ")"

	QUESTION Type = "?"

	// Keywords
	TRUE     Type = "TRUE"
	FALSE    Type = "FALSE"
	NONE     Type = "NONE"
	SHOW     Type = "SHOW"
	IF       Type = "IF"
	ELSE     Type = "ELSE"
	DEF      Type = "DEF"
	END      Type = "END"
	LOOP     Type = "LOOP"
	RETURN   Type = "RETURN"
	LIST     Type = "LIST"
	SET      Type = "SET"
	MAP      Type = "MAP"
	IMPORT   Type = "IMPORT"
	SYS      Type = "SYS"
	TO       Type = "TO"
	FROM     Type = "FROM"
	AT       Type = "AT"
	OF       Type = "OF"
	IS       Type = "IS"
	ADD      Type = "ADD"
	CHANGE   Type = "CHANGE"
	REMOVE   Type = "REMOVE"
	DELETE   Type = "DELETE"
	SORT     Type = "SORT"
	REVERSE  Type = "REVERSE"
	INSERT   Type = "INSERT"
	MIN      Type = "MIN"
	MAX      Type = "MAX"
	LENGTH   Type = "LENGTH"
	READ     Type = "READ"
	WRITE    Type = "WRITE"
	APPEND   Type = "APPEND"
	DIR      Type = "DIRECTORY"
	BREAK    Type = "BREAK"
	CONTINUE Type = "CONTINUE"
	GIVES    Type = "GIVES"
	IN       Type = "IN"
	ASSEMBLE Type = "ASSEMBLE"
	SAFE     Type = "SAFE"
	HANDLE   Type = "HANDLE"
	FAIL     Type = "FAIL"
	WARN     Type = "WARN"
)

// LiteralBrace stands in a STRING token's Literal for an escaped \{, so
// the parser can tell it from a {name} interpolation. It never appears in
// a Turtle value: the parser turns it back into '{'.
const LiteralBrace = '\x00'

var keywords = map[string]Type{
	"true":      TRUE,
	"false":     FALSE,
	"none":      NONE,
	"show":      SHOW,
	"if":        IF,
	"else":      ELSE,
	"def":       DEF,
	"end":       END,
	"loop":      LOOP,
	"return":    RETURN,
	"list":      LIST,
	"set":       SET,
	"map":       MAP,
	"import":    IMPORT,
	"sys":       SYS,
	"to":        TO,
	"from":      FROM,
	"at":        AT,
	"of":        OF,
	"is":        IS,
	"add":       ADD,
	"change":    CHANGE,
	"remove":    REMOVE,
	"delete":    DELETE,
	"sort":      SORT,
	"reverse":   REVERSE,
	"insert":    INSERT,
	"min":       MIN,
	"max":       MAX,
	"length":    LENGTH,
	"read":      READ,
	"write":     WRITE,
	"append":    APPEND,
	"directory": DIR,
	"break":     BREAK,
	"continue":  CONTINUE,
	"gives":     GIVES,
	"in":        IN,
	"assemble":  ASSEMBLE,
	"safe":      SAFE,
	"handle":    HANDLE,
	"fail":      FAIL,
	"warn":      WARN,
	"div":       DIV,
}

// LookupIdent returns the keyword Type for literal, or IDENT if it isn't
// a reserved word.
func LookupIdent(literal string) Type {
	if tok, ok := keywords[literal]; ok {
		return tok
	}
	return IDENT
}
