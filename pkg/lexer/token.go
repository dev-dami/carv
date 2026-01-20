package lexer

import "fmt"

type TokenType int

const (
	// Special tokens
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF
	TOKEN_NEWLINE
	TOKEN_COMMENT

	// Identifiers and literals
	TOKEN_IDENT  // variable, function, class names
	TOKEN_INT    // 123
	TOKEN_FLOAT  // 123.45
	TOKEN_STRING // "hello"
	TOKEN_CHAR   // 'c'
	TOKEN_TRUE   // true
	TOKEN_FALSE  // false
	TOKEN_NIL    // nil

	// Operators
	TOKEN_PLUS      // +
	TOKEN_MINUS     // -
	TOKEN_STAR      // *
	TOKEN_SLASH     // /
	TOKEN_PERCENT   // %
	TOKEN_CARET     // ^
	TOKEN_AMPERSAND // &
	TOKEN_VBAR      // |
	TOKEN_TILDE     // ~
	TOKEN_BANG      // !
	TOKEN_QUESTION  // ?

	// Comparison
	TOKEN_EQ  // ==
	TOKEN_NE  // !=
	TOKEN_LT  // <
	TOKEN_LE  // <=
	TOKEN_GT  // >
	TOKEN_GE  // >=
	TOKEN_AND // &&
	TOKEN_OR  // ||

	// Assignment
	TOKEN_ASSIGN       // =
	TOKEN_PLUS_EQ      // +=
	TOKEN_MINUS_EQ     // -=
	TOKEN_STAR_EQ      // *=
	TOKEN_SLASH_EQ     // /=
	TOKEN_PERCENT_EQ   // %=
	TOKEN_AMPERSAND_EQ // &=
	TOKEN_VBAR_EQ      // |=
	TOKEN_CARET_EQ     // ^=

	// Delimiters
	TOKEN_LPAREN    // (
	TOKEN_RPAREN    // )
	TOKEN_LBRACE    // {
	TOKEN_RBRACE    // }
	TOKEN_LBRACKET  // [
	TOKEN_RBRACKET  // ]
	TOKEN_COMMA     // ,
	TOKEN_DOT       // .
	TOKEN_COLON     // :
	TOKEN_SEMI      // ;
	TOKEN_ARROW     // ->
	TOKEN_FAT_ARROW // =>

	// Pipes (core to Carv)
	TOKEN_PIPE      // |>
	TOKEN_PIPE_BACK // <|

	// Concurrency
	TOKEN_LARROW // <-
	TOKEN_RARROW // ->  (reused for channel send)

	// Keywords
	TOKEN_FN        // fn
	TOKEN_LET       // let
	TOKEN_MUT       // mut
	TOKEN_CONST     // const
	TOKEN_IF        // if
	TOKEN_ELSE      // else
	TOKEN_MATCH     // match
	TOKEN_FOR       // for
	TOKEN_WHILE     // while
	TOKEN_LOOP      // loop
	TOKEN_BREAK     // break
	TOKEN_CONTINUE  // continue
	TOKEN_RETURN    // return
	TOKEN_CLASS     // class
	TOKEN_INTERFACE // interface
	TOKEN_IMPL      // impl
	TOKEN_PUB       // pub
	TOKEN_PRIV      // priv
	TOKEN_STATIC    // static
	TOKEN_SELF      // self
	TOKEN_SUPER     // super
	TOKEN_NEW       // new
	TOKEN_TYPE      // type
	TOKEN_AS        // as
	TOKEN_IN        // in
	TOKEN_IS        // is
	TOKEN_OK        // Ok
	TOKEN_ERR       // Err

	// Concurrency keywords
	TOKEN_SPAWN  // spawn
	TOKEN_ASYNC  // async
	TOKEN_AWAIT  // await
	TOKEN_CHAN   // chan
	TOKEN_SELECT // select
	TOKEN_SEND   // send
	TOKEN_RECV   // recv

	// Module
	TOKEN_IMPORT  // import
	TOKEN_EXPORT  // export
	TOKEN_MODULE  // module
	TOKEN_FROM    // from
	TOKEN_REQUIRE // require

	// String interpolation
	TOKEN_INTERP_STRING // f"hello {name}"

	// Types
	TOKEN_INT_TYPE    // int
	TOKEN_FLOAT_TYPE  // float
	TOKEN_BOOL_TYPE   // bool
	TOKEN_STRING_TYPE // string
	TOKEN_CHAR_TYPE   // char
	TOKEN_VOID_TYPE   // void
	TOKEN_ANY_TYPE    // any
)

var tokenNames = map[TokenType]string{
	TOKEN_ILLEGAL: "ILLEGAL",
	TOKEN_EOF:     "EOF",
	TOKEN_NEWLINE: "NEWLINE",
	TOKEN_COMMENT: "COMMENT",

	TOKEN_IDENT:  "IDENT",
	TOKEN_INT:    "INT",
	TOKEN_FLOAT:  "FLOAT",
	TOKEN_STRING: "STRING",
	TOKEN_CHAR:   "CHAR",
	TOKEN_TRUE:   "true",
	TOKEN_FALSE:  "false",
	TOKEN_NIL:    "nil",

	TOKEN_PLUS:      "+",
	TOKEN_MINUS:     "-",
	TOKEN_STAR:      "*",
	TOKEN_SLASH:     "/",
	TOKEN_PERCENT:   "%",
	TOKEN_CARET:     "^",
	TOKEN_AMPERSAND: "&",
	TOKEN_VBAR:      "|",
	TOKEN_TILDE:     "~",
	TOKEN_BANG:      "!",
	TOKEN_QUESTION:  "?",

	TOKEN_EQ:  "==",
	TOKEN_NE:  "!=",
	TOKEN_LT:  "<",
	TOKEN_LE:  "<=",
	TOKEN_GT:  ">",
	TOKEN_GE:  ">=",
	TOKEN_AND: "&&",
	TOKEN_OR:  "||",

	TOKEN_ASSIGN:       "=",
	TOKEN_PLUS_EQ:      "+=",
	TOKEN_MINUS_EQ:     "-=",
	TOKEN_STAR_EQ:      "*=",
	TOKEN_SLASH_EQ:     "/=",
	TOKEN_PERCENT_EQ:   "%=",
	TOKEN_AMPERSAND_EQ: "&=",
	TOKEN_VBAR_EQ:      "|=",
	TOKEN_CARET_EQ:     "^=",

	TOKEN_LPAREN:    "(",
	TOKEN_RPAREN:    ")",
	TOKEN_LBRACE:    "{",
	TOKEN_RBRACE:    "}",
	TOKEN_LBRACKET:  "[",
	TOKEN_RBRACKET:  "]",
	TOKEN_COMMA:     ",",
	TOKEN_DOT:       ".",
	TOKEN_COLON:     ":",
	TOKEN_SEMI:      ";",
	TOKEN_ARROW:     "->",
	TOKEN_FAT_ARROW: "=>",

	TOKEN_PIPE:      "|>",
	TOKEN_PIPE_BACK: "<|",

	TOKEN_LARROW: "<-",

	TOKEN_FN:        "fn",
	TOKEN_LET:       "let",
	TOKEN_MUT:       "mut",
	TOKEN_CONST:     "const",
	TOKEN_IF:        "if",
	TOKEN_ELSE:      "else",
	TOKEN_MATCH:     "match",
	TOKEN_FOR:       "for",
	TOKEN_WHILE:     "while",
	TOKEN_LOOP:      "loop",
	TOKEN_BREAK:     "break",
	TOKEN_CONTINUE:  "continue",
	TOKEN_RETURN:    "return",
	TOKEN_CLASS:     "class",
	TOKEN_INTERFACE: "interface",
	TOKEN_IMPL:      "impl",
	TOKEN_PUB:       "pub",
	TOKEN_PRIV:      "priv",
	TOKEN_STATIC:    "static",
	TOKEN_SELF:      "self",
	TOKEN_SUPER:     "super",
	TOKEN_NEW:       "new",
	TOKEN_TYPE:      "type",
	TOKEN_AS:        "as",
	TOKEN_IN:        "in",
	TOKEN_IS:        "is",
	TOKEN_OK:        "Ok",
	TOKEN_ERR:       "Err",

	TOKEN_SPAWN:  "spawn",
	TOKEN_ASYNC:  "async",
	TOKEN_AWAIT:  "await",
	TOKEN_CHAN:   "chan",
	TOKEN_SELECT: "select",
	TOKEN_SEND:   "send",
	TOKEN_RECV:   "recv",

	TOKEN_IMPORT:        "import",
	TOKEN_EXPORT:        "export",
	TOKEN_MODULE:        "module",
	TOKEN_FROM:          "from",
	TOKEN_REQUIRE:       "require",
	TOKEN_INTERP_STRING: "INTERP_STRING",

	TOKEN_INT_TYPE:    "int",
	TOKEN_FLOAT_TYPE:  "float",
	TOKEN_BOOL_TYPE:   "bool",
	TOKEN_STRING_TYPE: "string",
	TOKEN_CHAR_TYPE:   "char",
	TOKEN_VOID_TYPE:   "void",
	TOKEN_ANY_TYPE:    "any",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

var keywords = map[string]TokenType{
	"fn":        TOKEN_FN,
	"let":       TOKEN_LET,
	"mut":       TOKEN_MUT,
	"const":     TOKEN_CONST,
	"if":        TOKEN_IF,
	"else":      TOKEN_ELSE,
	"match":     TOKEN_MATCH,
	"for":       TOKEN_FOR,
	"while":     TOKEN_WHILE,
	"loop":      TOKEN_LOOP,
	"break":     TOKEN_BREAK,
	"continue":  TOKEN_CONTINUE,
	"return":    TOKEN_RETURN,
	"class":     TOKEN_CLASS,
	"interface": TOKEN_INTERFACE,
	"impl":      TOKEN_IMPL,
	"pub":       TOKEN_PUB,
	"priv":      TOKEN_PRIV,
	"static":    TOKEN_STATIC,
	"self":      TOKEN_SELF,
	"super":     TOKEN_SUPER,
	"new":       TOKEN_NEW,
	"type":      TOKEN_TYPE,
	"as":        TOKEN_AS,
	"in":        TOKEN_IN,
	"is":        TOKEN_IS,
	"Ok":        TOKEN_OK,
	"Err":       TOKEN_ERR,
	"spawn":     TOKEN_SPAWN,
	"async":     TOKEN_ASYNC,
	"await":     TOKEN_AWAIT,
	"chan":      TOKEN_CHAN,
	"select":    TOKEN_SELECT,
	"send":      TOKEN_SEND,
	"recv":      TOKEN_RECV,
	"import":    TOKEN_IMPORT,
	"export":    TOKEN_EXPORT,
	"module":    TOKEN_MODULE,
	"from":      TOKEN_FROM,
	"require":   TOKEN_REQUIRE,
	"true":      TOKEN_TRUE,
	"false":     TOKEN_FALSE,
	"nil":       TOKEN_NIL,
	"int":       TOKEN_INT_TYPE,
	"float":     TOKEN_FLOAT_TYPE,
	"bool":      TOKEN_BOOL_TYPE,
	"string":    TOKEN_STRING_TYPE,
	"char":      TOKEN_CHAR_TYPE,
	"void":      TOKEN_VOID_TYPE,
	"any":       TOKEN_ANY_TYPE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) Pos() string {
	return fmt.Sprintf("%d:%d", t.Line, t.Column)
}

func (t Token) IsKeyword() bool {
	return t.Type >= TOKEN_FN && t.Type <= TOKEN_ANY_TYPE
}
