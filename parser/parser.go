// Package parser turns a token stream into a Turtle AST.
//
// Convention used throughout this file: every parseXStatement function
// advances curToken to the first token of whatever follows it before
// returning (so ParseProgram/parseBlockUntil never call nextToken after
// a statement). Expression parsers use the opposite, standard Pratt
// convention: they leave curToken on their own last consumed token, so
// an enclosing parser can inspect peekToken to decide what comes next.
package parser

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/syntax"
	"Turtle/token"
)

const (
	LOWEST int = iota
	METHOD     // "r is x at m args .": the statement's own method, after its value
	OR
	AND
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	INDEX // "x at m": a method works on the value right before it
)

// SENTENCEARG is how far a sentence call's argument reaches without
// brackets: through arithmetic (nums get i + 1 is get[nums, i + 1]), but
// not past a comparison, && or ||, which work on the call's result, as in
// most languages: s has "a" && ok is (s has "a") && ok, a solve b == x is
// (a solve b) == x. A give function still takes the rest of its line.
const SENTENCEARG = LESSGREATER

var precedences = map[token.Type]int{
	token.AT:       METHOD,
	token.OR:       OR,
	token.AND:      AND,
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.LE:       LESSGREATER,
	token.GE:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.ASTERISK: PRODUCT,
	token.SLASH:    PRODUCT,
	token.DIV:      PRODUCT,
	token.PERCENT:  PRODUCT,
}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

type Parser struct {
	l *lexer.Lexer

	prevToken token.Token
	curToken  token.Token
	peekToken token.Token
	buf       []token.Token

	errors []string
	errs   []Error
	// multilineString is the line a "..." text that ran over several lines
	// started on: usually a missing closing quote, which a later error
	// mentions.
	multilineString int
	multilinePos    int
	multilineEnd    int // the line its closing quote is on

	// inBrackets is true while parsing a comma-separated list inside
	// [...] (call arguments, list/set/map literals). There a sentence-style
	// call takes only one argument, so the list's own commas aren't
	// swallowed: check["x", t find "W", 7] is check["x", find[t, "W"], 7].
	inBrackets bool

	// inIsReceiver is true while parsing the receiver of
	// "r is x at m args ." at its top level, where the statement itself
	// handles "at" and its unbracketed args.
	inIsReceiver bool

	// inFieldObject is true while parsing the value after "field of",
	// where only get[...] and slice[...] reach into it (methodPrecedence).
	inFieldObject bool

	// randomImported is set by "import random": from then on, "random"
	// followed by a kind of value (random list of 5 integers) is a
	// random-value sentence (see random.go). Elsewhere random is an
	// ordinary name.
	randomImported bool

	// logImported is set by "import log": a line starting with log and a
	// value (or a level word) is a log statement (see parseLogStatement).
	logImported bool

	// testImported is set by "import test": check, verify and validate
	// begin statements from then on (see testlib.go).
	testImported bool

	// linearImported is set by "import linear": matrix [...] makes a
	// matrix from then on (see linear.go).
	linearImported bool

	// stopWords end the value being parsed (see pushStops), and
	// inVerifyValue keeps "at least" / "at most" from being read as a
	// method call on the values being verified.
	stopWords map[string]int
	// theories are the theories read so far, by word: from then on their
	// phrases are read (see theory.go); laterTheories is where each theory
	// in the file is, so a use above its theory can say so.
	theories      map[string]*theorySpec
	laterTheories map[string]int
	// ModuleDir is the folder imports are found in (the program's), so
	// an imported file's theories are known and their phrases read; ""
	// when there's no file to look in. importing is the files being read
	// for their theories, so two files importing each other end.
	ModuleDir string
	importing map[string]bool
	unlisted  map[string]string // theories an import list leaves out, and the file they're in
	private   map[string]string // imported files' private (~) theories, and the file they're in
	// TestFile is set for a test file run by turtle test: it reads the
	// private (~) theories of the files it imports too, to test them.
	TestFile bool
	// SelfLibrary is the library a standard-library file belongs to:
	// its own "import data" is data's Go half, not its theories again.
	SelfLibrary   string
	inVerifyValue int

	// lines is the source split into lines, for statements' text.
	lines []string

	// inShape is true while parsing a value inside a shape (from A to B),
	// where a name followed by "rounded" isn't a sentence-style call.
	inShape bool

	// hereUses counts "here" in the scroll step being parsed (see
	// scroll.go): a step naming here gets the value there, not first.
	hereUses int

	// warnings are spots that parse but probably don't mean what they
	// seem to (see Warnings).
	warnings []Error

	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	p.prefixParseFns = map[token.Type]prefixParseFn{
		token.IDENT:  p.parseIdentifier,
		token.INT:    p.parseIntegerLiteral,
		token.FLOAT:  p.parseFloatLiteral,
		token.STRING: p.parseStringLiteral,
		token.RAWSTRING: func() ast.Expression {
			return p.maybeSentence(&ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal})
		},
		token.TRUE:     p.parseBoolean,
		token.FALSE:    p.parseBoolean,
		token.NONE:     p.parseNone,
		token.MINUS:    p.parsePrefixExpression,
		token.BANG:     p.parsePrefixExpression,
		token.LPAREN:   p.parseGroupedExpression,
		token.LIST:     p.parseListLiteral,
		token.SET:      p.parseSetLiteral,
		token.MAP:      p.parseMapLiteral,
		token.SORT:     p.parseSortModuleExpression,
		token.MIN:      p.parseMinMaxLength,
		token.MAX:      p.parseMinMaxLength,
		token.LENGTH:   p.parseMinMaxLength,
		token.CHANGE:   p.parseChangeExpression,
		token.LBRACKET: p.parseBracketFunctionLiteral,
		token.SCROLL:   p.parseScrollExpression,
	}
	p.infixParseFns = map[token.Type]infixParseFn{
		token.PLUS:     p.parseInfixExpression,
		token.MINUS:    p.parseInfixExpression,
		token.ASTERISK: p.parseInfixExpression,
		token.SLASH:    p.parseInfixExpression,
		token.DIV:      p.parseInfixExpression,
		token.PERCENT:  p.parseInfixExpression,
		token.LT:       p.parseInfixExpression,
		token.GT:       p.parseInfixExpression,
		token.LE:       p.parseInfixExpression,
		token.GE:       p.parseInfixExpression,
		token.EQ:       p.parseInfixExpression,
		token.NOT_EQ:   p.parseInfixExpression,
		token.AND:      p.parseInfixExpression,
		token.OR:       p.parseInfixExpression,
		token.AT:       p.parseMethodCallExpression,
		token.IDENT:    p.parseTypeCheck, // only "x type Order" (see typeCheckAhead)
	}

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return p.errors }

// Enable turns on a library's sentences as "import <name>" would, for a
// builtin library's own Turtle code (which can't import itself).
func (p *Parser) Enable(name string) {
	switch name {
	case "random":
		p.randomImported = true
	case "test":
		p.testImported = true
	case "log":
		p.logImported = true
	case "linear":
		p.linearImported = true
	}
}

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errorAt(p.curToken.Line, p.curToken.Pos, format, args...)
}

func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	if len(p.buf) > 0 {
		p.peekToken = p.buf[0]
		p.buf = p.buf[1:]
	} else {
		p.peekToken = p.l.NextToken()
	}
}

// peekN returns the token n positions ahead of curToken; peekN(0) is
// curToken itself, peekN(1) is peekToken.
func (p *Parser) peekN(n int) token.Token {
	if n == 0 {
		return p.curToken
	}
	if n == 1 {
		return p.peekToken
	}
	for len(p.buf) < n-1 {
		p.buf = append(p.buf, p.l.NextToken())
	}
	return p.buf[n-2]
}

func (p *Parser) curTokenIs(t token.Type) bool  { return p.curToken.Type == t }
func (p *Parser) peekTokenIs(t token.Type) bool { return p.peekToken.Type == t }

func (p *Parser) expectPeek(t token.Type) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.expectError(t)
	return false
}

func (p *Parser) peekPrecedence() int {
	// A text over several lines ends below where it starts: an operator
	// right after its closing quote is on its line.
	if p.peekToken.Line != p.endLine(p.curToken) {
		return LOWEST
	}
	// A method works on the value right before it, like Python's
	// a.invert(): "a at invert == b at invert" compares the two inverted
	// maps, "a" + "b" at upper is "aB", and ("a" + "b") at upper is "AB".
	if p.peekTokenIs(token.AT) {
		return p.methodPrecedence()
	}
	if p.typeCheckAhead() {
		return EQUALS
	}
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// methodPrecedence is how tightly the "at" in peekToken binds.
//
// After "of", only get[...] and slice[...] reach into the value: in
// "title of books at get[0]" the get picks a book, while in "title of b
// at upper" upper works on the title (the field expression ends first).
//
// In "r is x at m args ." the statement takes unbracketed args itself,
// so a method followed by an argument ends the value there.
func (p *Parser) methodPrecedence() int {
	if p.inVerifyValue > 0 && (p.peekN(2).Literal == "least" || p.peekN(2).Literal == "most") {
		return LOWEST // "verify rolls at least 2 ...": the rule, not a method
	}
	if p.inFieldObject {
		// get and slice pick from the value after "of", bracketed or with
		// one bare argument: title of books at get[0], title of books at get 0.
		if isIndexMethod(p.peekN(2)) && (p.peekN(3).Type == token.LBRACKET && p.peekN(3).Line == p.peekToken.Line || p.argumentStartsAt(3)) {
			return INDEX
		}
		return METHOD
	}
	if p.inIsReceiver && p.peekN(3).Type != token.LBRACKET && p.argumentStartsAt(3) {
		return METHOD
	}
	return INDEX
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.curToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// requirePeriod consumes a trailing '.' and advances past it. Used by
// every statement kind confirmed to require one: show, data-structure
// operations, and "is" assignments.
// isPeriod ends an is line, which is a sentence: it ends with a period,
// as show and add do. Missing at the end of the line, the error says so,
// and that = needs none.
func (p *Parser) isPeriod() bool {
	if !p.peekTokenIs(token.PERIOD) && (p.peekTokenIs(token.EOF) || p.peekToken.Line != p.endLine(p.curToken)) {
		p.errorAt(p.endLine(p.curToken), p.curToken.End, "a line with is ends with '.' (or write name = value, which needs none)")
		return false
	}
	return p.requirePeriod()
}

func (p *Parser) requirePeriod() bool {
	if !p.expectPeek(token.PERIOD) {
		return false
	}
	p.nextToken()
	return true
}

// ---- top level ---------------------------------------------------------

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	p.scanTheories()
	for !p.curTokenIs(token.EOF) {
		before := p.curToken
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		if p.curToken == before {
			p.nextToken() // guarantee forward progress past a malformed statement
		}
	}
	if p.testImported {
		linkValidateExamples(program.Statements)
	}
	return program
}

// parseBlockUntil parses statements starting just after curToken (the
// block's own opening token) until stop() reports true. It never
// consumes whatever satisfies stop(); the caller inspects/consumes it.
func (p *Parser) parseBlockUntil(stop func() bool) *ast.BlockStatement {
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = false // a block inside brackets ([x] give ...) is ordinary code
	block := &ast.BlockStatement{Token: p.curToken}
	p.nextToken()
	for !stop() && !p.curTokenIs(token.EOF) {
		before := p.curToken
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		if p.curToken == before {
			p.nextToken()
		}
	}
	return block
}

func (p *Parser) parseStatement() ast.Statement {
	// "max = 5", "list is ...": a reserved word used as a variable name.
	// One clear error instead of a cascade from parsing it as a keyword.
	if p.isReservedWord(p.curToken) && (p.peekTokenIs(token.ASSIGN) || p.peekTokenIs(token.IS)) && p.peekToken.Line == p.curToken.Line {
		p.reservedNameError(p.curToken, "a variable")
		p.skipLine()
		return nil
	}
	if p.startsMatrix() && (p.peekTokenIs(token.ASSIGN) || p.peekTokenIs(token.IS)) && p.peekToken.Line == p.curToken.Line {
		p.matrixNameError(p.curToken, "a variable")
		p.skipLine()
		return nil
	}
	switch p.curToken.Type {
	case token.SHOW:
		return p.parseShowStatement()
	case token.WARN:
		// warn "..." . is show for stderr (system's).
		if stmt, ok := p.parseShowStatement().(*ast.ShowStatement); ok {
			stmt.Stderr = true
			return stmt
		}
		return nil
	case token.RETURN:
		return p.parseReturnStatement()
	case token.BREAK:
		stmt := &ast.BreakStatement{Token: p.curToken}
		p.nextToken()
		return stmt
	case token.CONTINUE:
		stmt := &ast.ContinueStatement{Token: p.curToken}
		p.nextToken()
		return stmt
	case token.SAFE:
		if p.isSafeEnd() {
			p.errorf("'safe [end]' needs a 'safe' block and a 'handle [...] error .' line above it")
			p.skipLine()
			return nil
		}
		return p.parseSafeStatement()
	case token.HANDLE:
		p.errorf("'handle' needs a 'safe' above it, starting the code it protects")
		p.skipLine()
		return nil
	case token.FAIL:
		return p.parseFailStatement()
	case token.DEF:
		return p.parseFunctionDef()
	case token.ASSEMBLE:
		return p.parseAssembleStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.SYS:
		return p.parseSysStatement()
	case token.ADD:
		return p.parseAddStatement()
	case token.CHANGE:
		return p.parseChangeStatement()
	case token.REMOVE:
		return p.parseRemoveStatement()
	case token.DELETE:
		return p.parseDeleteStatement()
	case token.SORT:
		if p.sortModuleCall() {
			return p.parseIdentifierLeadStatement()
		}
		return p.parseSortStatement()
	case token.REVERSE:
		return p.parseReverseStatement()
	case token.INSERT:
		return p.parseInsertStatement()
	case token.MIN, token.MAX, token.LENGTH:
		return p.parseMinMaxLengthStatement()
	case token.LBRACKET:
		return p.parseBracketStatement()
	case token.IDENT:
		return p.parseIdentifierLeadStatement()
	case token.SCROLL:
		// "scroll rows into table_write["a.csv", here] .": run for its effect.
		tok := p.curToken
		expr := p.parseExpression(LOWEST)
		if !p.requirePeriod() {
			return nil
		}
		return &ast.ExpressionStatement{Token: tok, Expression: expr}
	default:
		p.statementStartError()
		p.nextToken()
		return nil
	}
}

func (p *Parser) parseBracketStatement() ast.Statement {
	switch p.peekToken.Type {
	case token.LOOP:
		return p.parseLoopStatement()
	case token.READ:
		return p.parseFileReadStatement()
	case token.WRITE:
		return p.parseFileWriteStatement(false)
	case token.APPEND:
		return p.parseFileWriteStatement(true)
	case token.DIR:
		return p.parseDirectoryStatement()
	case token.IF:
		return p.parseNestedIfStatement()
	default:
		p.errorf("a line can't start with '[' followed by %s; blocks start [loop][, [read], [write], [append] or [directory]", describe(p.peekToken))
		p.nextToken()
		return nil
	}
}

// ---- assignment / input / is / bare call --------------------------------

func (p *Parser) parseIdentifierLeadStatement() ast.Statement {
	if p.startsTheory() {
		return p.parseTheoryStatement()
	}
	if p.theoryOnKeyword() {
		p.skipToTheoryEnd()
		return nil
	}
	if p.atTheoryEnd() {
		p.errorf("theory [end] without a theory to end")
		p.skipLine()
		return nil
	}
	// A phrase on its own line: "tally 1 in nums ."
	if p.startsPhrase() && !p.peekTokenIs(token.ASSIGN) && !p.peekTokenIs(token.IS) {
		tok := p.curToken
		call := p.parseTheoryCall()
		p.nextToken()
		if p.curTokenIs(token.PERIOD) && p.curToken.Line == tok.Line {
			p.nextToken()
		}
		return &ast.ExpressionStatement{Token: tok, Expression: call}
	}
	if p.usedBeforeTheory() {
		p.skipLine()
		return nil
	}
	if p.curToken.Literal == "diagnose" && (p.peekTokenIs(token.EOF) || p.peekToken.Line != p.curToken.Line) {
		return p.parseDiagnoseBlock()
	}
	if p.startsTestStatement() {
		return p.parseTestStatement()
	}
	// "log ... ." after import log (log is otherwise an ordinary name).
	if p.logImported && p.curToken.Literal == "log" && (p.peekTokenIs(token.WARN) || p.peekStartsArgument() && !p.peekTokenIs(token.LBRACKET)) {
		return p.parseLogStatement()
	}
	// "put 99 to nums at 2 ." (put is otherwise an ordinary name).
	if p.curToken.Literal == "put" && p.peekStartsArgument() && !p.peekTokenIs(token.LBRACKET) {
		return p.parseAtIndexStatement(ast.OpPut)
	}
	if p.peekTokenIs(token.OF) && p.peekToken.Line == p.curToken.Line {
		return p.parseFieldStatement()
	}
	if w := p.curToken.Literal; (w == "for" || w == "while") && p.peekTokenIs(token.IDENT) || w == "while" && p.peekTokenIs(token.LPAREN) {
		p.statementStartError() // a loop from another language
		p.nextToken()
		return nil
	}
	if p.peekTokenIs(token.ASSIGN) {
		return p.parseAssignOrInputStatement()
	}
	if p.peekTokenIs(token.IS) {
		return p.parseIsStatement()
	}
	if p.peekTokenIs(token.LBRACKET) || p.isQualifiedName() {
		tok := p.curToken
		expr := p.parseExpression(LOWEST)
		p.nextToken()
		if p.curTokenIs(token.PERIOD) {
			p.nextToken()
		}
		if call, ok := expr.(*ast.CallExpression); ok {
			return &ast.CallStatement{Token: tok, Call: call}
		}
		return &ast.ExpressionStatement{Token: tok, Expression: expr}
	}
	// "nums at add 4", "m at put[9, 1, 2]": a method call on its own line
	// is a statement, as a function call is (the method does its work; any
	// value it gives back is dropped). Only a method call: a line that's
	// just a value, "nums at get 0 + 1", is still a mistake.
	// On its own line a method takes its arguments as the is-statement
	// does, unbracketed and separated by commas: m at put 9, 1, 2 .
	if m := p.peekN(2); p.curTokenIs(token.IDENT) && p.peekTokenIs(token.AT) && p.peekToken.Line == p.curToken.Line &&
		(m.Type == token.IDENT || slices.Contains(syntax.Methods, m.Literal)) {
		tok := p.curToken
		expr := p.parseMethodStatementValue()
		if _, ok := expr.(*ast.MethodCallExpression); ok && (p.peekTokenIs(token.PERIOD) || p.peekToken.Line != p.endLine(p.curToken) || p.peekTokenIs(token.EOF)) {
			p.nextToken()
			if p.curTokenIs(token.PERIOD) {
				p.nextToken()
			}
			return &ast.ExpressionStatement{Token: tok, Expression: expr}
		}
		p.errorAt(tok.Line, tok.Pos, "this line is a value, not a statement: to keep it, name it (r = ...); to see it, show it (show ... .)")
		p.skipLine()
		return nil
	}
	p.statementStartError()
	p.nextToken()
	return nil
}

// parseAssignOrInputStatement parses "name = expr", and input:
// "name = ?" or "name = ? \"prompt\"", either with an optional closing
// period. curToken is the name; peek is '='.
func (p *Parser) parseAssignOrInputStatement() ast.Statement {
	tok := p.curToken
	p.notPrivate(tok, "a variable")
	name := p.curToken.Literal
	p.nextToken() // name -> '='

	if p.peekTokenIs(token.QUESTION) {
		p.nextToken()             // '=' -> '?'
		var prompt ast.Expression // none: read without a prompt
		if p.peekToken.Line == p.curToken.Line && !p.peekTokenIs(token.PERIOD) && !p.peekTokenIs(token.EOF) {
			if !p.expectPeek(token.STRING) {
				return nil
			}
			prompt = p.stringExpression(p.curToken)
		}
		p.nextToken()
		if p.curTokenIs(token.PERIOD) && p.curToken.Line == p.prevToken.Line {
			p.nextToken()
		}
		return &ast.InputStatement{Token: tok, Name: name, Prompt: prompt}
	}

	p.nextToken() // '=' -> first token of value
	val := p.parseExpression(LOWEST)
	p.nextToken()
	// A closing period is fine, as after a call: "x = rows keep r give
	// ... ." reads like the sentence it is.
	if p.curTokenIs(token.PERIOD) && p.curToken.Line == p.prevToken.Line {
		p.nextToken()
	}
	return &ast.AssignStatement{Token: tok, Name: name, Value: val}
}

// parseIsStatement parses "name is expr ." and
// "name is receiver at method arg, arg ." curToken is name; peek is IS.
func (p *Parser) parseIsStatement() ast.Statement {
	tok := p.curToken
	p.notPrivate(tok, "a variable")
	name := p.curToken.Literal
	p.nextToken() // name -> IS
	p.nextToken() // IS -> first token of receiver
	// METHOD, not LOWEST: the statement form "r is x at m arg, arg ." parses
	// its own unbracketed args below, so the receiver mustn't consume "at".
	p.inIsReceiver = true
	receiver := p.parseExpression(METHOD)
	p.inIsReceiver = false

	if p.peekTokenIs(token.AT) {
		p.nextToken() // -> AT
		p.nextToken() // -> method name token
		method := p.curToken.Literal
		var args []ast.Expression
		if !p.peekTokenIs(token.PERIOD) && !p.peekTokenIs(token.EOF) && p.peekToken.Line == p.curToken.Line {
			p.nextToken()
			args = append(args, p.parseExpression(LOWEST))
			for p.peekTokenIs(token.COMMA) {
				p.nextToken()
				p.nextToken()
				args = append(args, p.parseExpression(LOWEST))
			}
		}
		if !p.isPeriod() {
			return nil
		}
		// The method works on the value right before it, as everywhere:
		// "r is "a" + "b" at upper ." is "a" + ("b" at upper), and
		// "r is !s at contains "a" ." negates the method call, not s.
		value := attachMethod(receiver, func(e ast.Expression) ast.Expression {
			return &ast.MethodCallExpression{Token: tok, Receiver: e, Method: method, Arguments: args}
		})
		return &ast.AssignStatement{Token: tok, Name: name, Value: value}
	}

	if !p.isPeriod() {
		return nil
	}
	return &ast.AssignStatement{Token: tok, Name: name, Value: receiver}
}

// parseMethodStatementValue is a method call standing as a statement,
// parsed as the value of "r is x at m args ." is: curToken starts the
// receiver.
func (p *Parser) parseMethodStatementValue() ast.Expression {
	tok := p.curToken
	p.inIsReceiver = true
	receiver := p.parseExpression(METHOD)
	p.inIsReceiver = false
	if !p.peekTokenIs(token.AT) {
		return receiver
	}
	p.nextToken() // -> AT
	p.nextToken() // -> method name
	method := p.curToken.Literal
	var args []ast.Expression
	if !p.peekTokenIs(token.PERIOD) && !p.peekTokenIs(token.EOF) && p.peekToken.Line == p.curToken.Line {
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
		for p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			args = append(args, p.parseExpression(LOWEST))
		}
	}
	return attachMethod(receiver, func(e ast.Expression) ast.Expression {
		return &ast.MethodCallExpression{Token: tok, Receiver: e, Method: method, Arguments: args}
	})
}

// logLevels are the words that can follow log.
var logLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

// parseLogStatement parses "log [debug|info|warn|error] <expr>, ... .".
func (p *Parser) parseLogStatement() ast.Statement {
	tok := p.curToken
	level := "info"
	if (p.peekTokenIs(token.WARN) || p.peekTokenIs(token.IDENT) && logLevels[p.peekToken.Literal]) && p.peekN(2).Line == p.peekToken.Line && p.peekN(2).Type != token.PERIOD {
		p.nextToken()
		level = p.curToken.Literal
	}
	if p.peekTokenIs(token.PERIOD) {
		p.errorf("log %s: say what to log, e.g. log \"started\" .", level)
		return nil
	}
	p.nextToken()
	exprs := []ast.Expression{p.parseExpression(LOWEST)}
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		exprs = append(exprs, p.parseExpression(LOWEST))
	}
	if !p.requirePeriod() {
		return nil
	}
	return &ast.LogStatement{Token: tok, Level: level, Expressions: exprs}
}

func (p *Parser) parseShowStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()
	exprs := []ast.Expression{p.parseExpression(LOWEST)}
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		exprs = append(exprs, p.parseExpression(LOWEST))
	}
	if !p.requirePeriod() {
		return nil
	}
	return &ast.ShowStatement{Token: tok, Expressions: exprs}
}

// parseReturnStatement handles "return <expr>" and a bare "return" (the
// last token on its line), which returns none.
func (p *Parser) parseReturnStatement() ast.Statement {
	tok := p.curToken
	if p.peekTokenIs(token.EOF) || p.peekToken.Line != tok.Line {
		p.nextToken()
		return &ast.ReturnStatement{Token: tok}
	}
	p.nextToken()
	val := p.parseExpression(LOWEST)
	p.nextToken()
	p.endScroll(val)
	return &ast.ReturnStatement{Token: tok, Value: val}
}

// ---- safe / handle / fail ---------------------------------------------

// errorKinds are the names a handle statement can list; they match the
// kinds the evaluator gives its runtime errors.
var errorKinds = []string{"file", "number", "math", "index", "key", "name", "type", "json", "date", "http", "sql", "csv", "test", "pattern", "crypt", "schedule", "config", "server", "scroll", "linear", "custom"}

// parseSafeStatement parses
//
//	safe
//	    <statements>
//	handle [kind, ...] name .
//	    <statements, run only on an error>
//	safe [end]
func (p *Parser) parseSafeStatement() ast.Statement {
	tok := p.curToken
	body := p.parseBlockUntil(func() bool { return p.curTokenIs(token.HANDLE) })
	if !p.curTokenIs(token.HANDLE) {
		p.errorf("'safe' on line %d needs a 'handle [...] error .' line to close it", tok.Line)
		return nil
	}
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	var kinds []string
	for !p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		k := p.curToken.Literal
		if !slices.Contains(errorKinds, k) {
			p.errorf("%q isn't a kind of error; handle can list %s, or [] for any error",
				k, strings.Join(errorKinds, ", "))
		}
		kinds = append(kinds, k)
		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	if p.isReservedWord(p.peekToken) {
		p.reservedNameError(p.peekToken, "an error variable")
		p.nextToken()
	} else if !p.expectPeek(token.IDENT) {
		return nil
	}
	p.notPrivate(p.curToken, "an error variable")
	name := p.curToken.Literal
	if !p.expectPeek(token.PERIOD) {
		return nil
	}
	handler := p.parseBlockUntil(p.isSafeEnd)
	if !p.isSafeEnd() {
		p.errorf("'safe' on line %d needs a 'safe [end]' after its handle code", tok.Line)
		return nil
	}
	p.nextToken() // SAFE -> '['
	p.nextToken() // '[' -> END
	p.nextToken() // END -> ']'
	p.nextToken() // past ']'
	return &ast.SafeStatement{Token: tok, Body: body, Kinds: kinds, Name: name, Handler: handler}
}

// isSafeEnd reports whether curToken starts "safe [end]".
func (p *Parser) isSafeEnd() bool {
	return p.curTokenIs(token.SAFE) &&
		p.peekN(1).Type == token.LBRACKET &&
		p.peekN(2).Type == token.END &&
		p.peekN(3).Type == token.RBRACKET
}

// parseFailStatement handles "fail <expr>". Like return, no period.
func (p *Parser) parseFailStatement() ast.Statement {
	tok := p.curToken
	if p.peekTokenIs(token.EOF) || p.peekToken.Line != tok.Line {
		p.errorf("'fail' needs a message, e.g. fail \"not enough money\"")
		p.nextToken()
		return nil
	}
	p.nextToken()
	val := p.parseExpression(LOWEST)
	p.nextToken()
	return &ast.FailStatement{Token: tok, Value: val}
}

func (p *Parser) parseImportStatement() ast.Statement {
	tok := p.curToken
	// "sort" names the sort library here, not the sort statement.
	if p.peekTokenIs(token.SORT) {
		p.peekToken.Type = token.IDENT
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	path := p.curToken.Literal
	if path == "random" {
		p.randomImported = true
	}
	if path == "test" {
		p.testImported = true
	}
	if path == "log" {
		p.logImported = true
	}
	if path == "linear" {
		p.linearImported = true
	}
	// import lib/utils: a module in a subfolder, path segments joined by '/'.
	for p.peekTokenIs(token.SLASH) && p.peekToken.Line == tok.Line {
		p.nextToken() // -> '/'
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		path += "/" + p.curToken.Literal
	}
	var names []string
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line {
		p.nextToken() // -> '['
		if p.peekTokenIs(token.RBRACKET) {
			p.errorf("import %s [] lists nothing to import — list names, or drop the brackets to import everything", path)
			p.nextToken() // -> ']'
			p.nextToken()
			return nil
		}
		for {
			if !p.expectPeek(token.IDENT) {
				return nil
			}
			names = append(names, p.curToken.Literal)
			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken()
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
	}
	p.importTheories(tok, path, names)
	p.nextToken()
	return &ast.ImportStatement{Token: tok, Path: path, Names: names}
}

// parseSysStatement is trivial: the lexer already captured the entire
// raw rest-of-line as this token's Literal (see lexer.captureRestOfLine),
// so no further tokenizing of arbitrary shell syntax is needed here.
func (p *Parser) parseSysStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()
	return &ast.SysStatement{Token: tok, Command: tok.Literal}
}

// ---- data-structure "sentence" statements ------------------------------

func (p *Parser) parseAddStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()
	val := p.parseExpression(LOWEST)
	if !p.expectPeek(token.TO) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	target := p.curToken.Literal
	var key ast.Expression
	if p.peekTokenIs(token.AT) && p.peekToken.Line == p.curToken.Line {
		// "add 12 to ages at "Cy" .": a map's key.
		p.nextToken()
		p.nextToken()
		key = p.parseExpression(LOWEST)
	}
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpAdd, Target: target, Value: val, Index: key}
}

// parseChangeExpression parses "change <expr> to <type>" as an expression,
// usable anywhere (assigned to a new variable, nested in a larger
// expression, passed as an argument, ...). curToken is CHANGE.
func (p *Parser) parseChangeExpression() ast.Expression {
	tok := p.curToken
	p.nextToken()
	src := p.parseExpression(LOWEST)
	if !p.expectPeek(token.TO) {
		return nil
	}
	p.nextToken()
	typeName := p.curToken.Literal
	return &ast.ChangeExpression{Token: tok, Source: src, TypeName: typeName}
}

// parseChangeStatement parses "change <ident> to <type> ." as a standalone
// statement: it mutates <ident> in place, so the source must be a plain
// identifier (like the target of "add ... to <ident> ."), not an
// arbitrary expression.
func (p *Parser) parseChangeStatement() ast.Statement {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expectPeek(token.TO) {
		return nil
	}
	p.nextToken()
	typeName := p.curToken.Literal
	if !p.requirePeriod() {
		return nil
	}
	return &ast.AssignStatement{
		Token: tok, Name: name,
		Value: &ast.ChangeExpression{Token: tok, Source: &ast.Identifier{Token: tok, Value: name}, TypeName: typeName},
	}
}

func (p *Parser) parseRemoveStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()
	val := p.parseExpression(LOWEST)
	if !p.expectPeek(token.FROM) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	target := p.curToken.Literal
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpRemove, Target: target, Value: val}
}

func (p *Parser) parseDeleteStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()
	val := p.parseExpression(LOWEST)
	if !p.expectPeek(token.FROM) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	target := p.curToken.Literal
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpDelete, Target: target, Value: val}
}

func (p *Parser) parseSortStatement() ast.Statement {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	target := p.curToken.Literal
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpSort, Target: target}
}

func (p *Parser) parseReverseStatement() ast.Statement {
	tok := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	target := p.curToken.Literal
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpReverse, Target: target}
}

func (p *Parser) parseInsertStatement() ast.Statement {
	return p.parseAtIndexStatement(ast.OpInsert)
}

// parseAtIndexStatement parses "insert <expr> to <ident> at <expr> ." and
// "put <expr> to <ident> at <expr> .".
func (p *Parser) parseAtIndexStatement(kind ast.OpKind) ast.Statement {
	tok := p.curToken
	p.nextToken()
	val := p.parseExpression(LOWEST)
	if !p.expectPeek(token.TO) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	target := p.curToken.Literal
	if !p.expectPeek(token.AT) {
		return nil
	}
	p.nextToken()
	idx := p.parseExpression(LOWEST)
	var idx2 ast.Expression
	if kind == ast.OpPut && p.peekTokenIs(token.COMMA) { // a matrix: row, column
		p.nextToken()
		p.nextToken()
		idx2 = p.parseExpression(LOWEST)
	}
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: kind, Target: target, Value: val, Index: idx, Index2: idx2}
}

func (p *Parser) parseMinMaxLengthStatement() ast.Statement {
	tok := p.curToken
	expr := p.parseMinMaxLength()
	if !p.requirePeriod() {
		return nil
	}
	return &ast.ExpressionStatement{Token: tok, Expression: expr, Print: true}
}

// ---- function definitions ------------------------------------------

func (p *Parser) parseFunctionDef() ast.Statement {
	tok := p.curToken
	if p.isReservedWord(p.peekToken) {
		// Report it, then keep parsing as if it were a name so the body
		// doesn't produce a cascade of follow-on errors.
		p.reservedNameError(p.peekToken, "a function")
		p.nextToken()
	} else if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if p.linearImported && name == "matrix" {
		p.matrixNameError(p.curToken, "a function")
	}
	if p.testImported && testWords[name] {
		p.errorf("%s is a test word in a file that imports test, so it can't name a function here; rename it (a function called %s from another file can be used as module %s[...])", name, name, name)
	}
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	var params []string
	if !p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		p.checkName(p.curToken, "a parameter")
		params = append(params, p.curToken.Literal)
		for p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			p.checkName(p.curToken, "a parameter")
			params = append(params, p.curToken.Literal)
		}
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	body := p.parseBlockUntil(p.isDefEnd)
	if !p.curTokenIs(token.DEF) {
		p.unclosedError(tok, fmt.Sprintf("the function %s", name), "def [end]")
		return nil
	}
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	if !p.expectPeek(token.END) {
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	p.nextToken()
	return &ast.FunctionDefStatement{Token: tok, Name: name, Parameters: params, Body: body}
}

func (p *Parser) isDefEnd() bool {
	return p.curTokenIs(token.DEF) &&
		p.peekN(1).Type == token.LBRACKET &&
		p.peekN(2).Type == token.END &&
		p.peekN(3).Type == token.RBRACKET
}

// ---- if / else if / else --------------------------------------------
//
// Real syntax uses reversed brackets: `if ] cond [ ... else if ] cond [
// ... else ] ... if [end]`. One nested level is written by prefixing
// each clause keyword with an extra literal '[': `[if ] cond [`,
// `[else if ] cond [`, `[else ]`, with no separate closing marker of its
// own — it implicitly ends the moment a clause at the *same or
// shallower* bracket depth is seen. This generalizes to arbitrary depth
// via ordinary recursion: entering a nested chain just means consuming
// one more leading '[' than its parent; the parent's own bare
// continuation tokens (no leading '[') are what make an inner chain
// return control upward, however many levels deep it is.

func (p *Parser) isIfChainStop() bool {
	if p.curTokenIs(token.ELSE) {
		return true
	}
	if p.isIfEnd() {
		return true
	}
	if p.curTokenIs(token.LBRACKET) && p.peekTokenIs(token.ELSE) {
		return true
	}
	return false
}

// isIfEnd reports whether curToken starts the closing "if [end]" marker,
// which — like "def [end]" — is four tokens: IF LBRACKET END RBRACKET.
func (p *Parser) isIfEnd() bool {
	return p.curTokenIs(token.IF) &&
		p.peekN(1).Type == token.LBRACKET &&
		p.peekN(2).Type == token.END &&
		p.peekN(3).Type == token.RBRACKET
}

// parseIfHeaderAndBody parses "if ] cond [" plus its body. curToken must
// be IF on entry.
func (p *Parser) parseIfHeaderAndBody() *ast.IfClause {
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	p.nextToken()
	cond := p.parseExpression(LOWEST)
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	body := p.parseBlockUntil(p.isIfChainStop)
	return &ast.IfClause{Condition: cond, Body: body}
}

// parseIfChain parses one full if/else-if/else sequence. If nested is
// true, curToken is IF but this chain was entered via a leading '['
// already consumed by the caller, and the chain must NOT consume a
// closing "if [end]" of its own — it returns as soon as a same-or-
// shallower-depth clause token appears, leaving it for the caller.
func (p *Parser) parseIfChain(tok token.Token, nested bool) ast.Statement {
	clause := p.parseIfHeaderAndBody()
	if clause == nil {
		return nil
	}
	clauses := []*ast.IfClause{clause}

	for {
		if p.curTokenIs(token.LBRACKET) && p.peekTokenIs(token.ELSE) {
			p.nextToken() // consume '[' -> ELSE
		} else if !p.curTokenIs(token.ELSE) {
			break
		}
		// curToken == ELSE
		if p.peekTokenIs(token.IF) {
			p.nextToken() // ELSE -> IF
			c := p.parseIfHeaderAndBody()
			if c == nil {
				return nil
			}
			clauses = append(clauses, c)
			continue
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		body := p.parseBlockUntil(p.isIfChainStop)
		clauses = append(clauses, &ast.IfClause{Condition: nil, Body: body})
		break
	}

	if !nested {
		if !p.isIfEnd() {
			p.unclosedError(tok, "this if", "if [end]")
			return nil
		}
		p.nextToken() // IF -> '['
		p.nextToken() // '[' -> END
		p.nextToken() // END -> ']'
		p.nextToken() // advance past ']'
	}
	return &ast.IfStatement{Token: tok, Clauses: clauses}
}

func (p *Parser) parseIfStatement() ast.Statement {
	return p.parseIfChain(p.curToken, false)
}

func (p *Parser) parseNestedIfStatement() ast.Statement {
	tok := p.curToken // '['
	p.nextToken()     // '[' -> IF
	return p.parseIfChain(tok, true)
}

// ---- loops -------------------------------------------------------------

func (p *Parser) isLoopEnd() bool {
	return p.curTokenIs(token.LBRACKET) &&
		p.peekN(1).Type == token.LOOP &&
		p.peekN(2).Type == token.RBRACKET &&
		p.peekN(3).Type == token.LBRACKET &&
		p.peekN(4).Type == token.END
}

func (p *Parser) parseLoopStatement() ast.Statement {
	tok := p.curToken // '['
	p.nextToken()     // '[' -> LOOP
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	p.nextToken() // -> first token of header

	var kind ast.LoopKind
	var init, post ast.Statement
	var cond, iterable ast.Expression
	var vars []string
	if p.isEachHeader() {
		kind = ast.LoopEach
		vars, iterable = p.parseEachHeader()
		for _, v := range vars {
			p.notPrivate(token.Token{Literal: v, Line: tok.Line, Pos: tok.Pos}, "a loop variable")
		}
	} else {
		kind, init, cond, post = p.parseLoopHeader()
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	body := p.parseBlockUntil(p.isLoopEnd)
	if !p.curTokenIs(token.LBRACKET) {
		p.unclosedError(tok, "this loop", "[loop][end]")
		return nil
	}
	if !p.expectPeek(token.LOOP) {
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	if !p.expectPeek(token.END) {
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	p.nextToken()
	return &ast.LoopStatement{Token: tok, Kind: kind, Init: init, Condition: cond, Post: post, Vars: vars, Iterable: iterable, Body: body}
}

// isEachHeader reports whether the loop header at curToken is the
// for-each form: "x in <expr>" or "a, b in <expr>".
func (p *Parser) isEachHeader() bool {
	if p.isReservedWord(p.curToken) && (p.peekTokenIs(token.IN) || p.peekTokenIs(token.COMMA)) {
		p.reservedNameError(p.curToken, "a loop variable")
		return true // parse it as for-each anyway, so nothing cascades
	}
	if !p.curTokenIs(token.IDENT) {
		return false
	}
	if p.peekTokenIs(token.IN) {
		return true
	}
	if p.peekTokenIs(token.COMMA) && p.peekN(3).Type == token.IN && p.isReservedWord(p.peekN(2)) {
		p.reservedNameError(p.peekN(2), "a loop variable")
		return true
	}
	return p.peekTokenIs(token.COMMA) && p.peekN(2).Type == token.IDENT && p.peekN(3).Type == token.IN
}

func (p *Parser) parseEachHeader() ([]string, ast.Expression) {
	vars := []string{p.curToken.Literal}
	if p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		vars = append(vars, p.curToken.Literal)
	}
	p.nextToken() // -> IN
	p.nextToken() // -> first token of the collection expression
	return vars, p.parseExpression(LOWEST)
}

// headerHasSemicolon reports whether the loop header starting at
// curToken (already positioned just past the header's opening '[')
// contains a top-level ';', which distinguishes the C-style
// "init; cond; post" form from the bare-condition while-style form.
func (p *Parser) headerHasSemicolon() bool {
	depth := 0
	for i := 0; i < 512; i++ {
		t := p.peekN(i)
		if t.Type == token.EOF {
			return false
		}
		if t.Type == token.LBRACKET {
			depth++
		}
		if t.Type == token.RBRACKET {
			if depth == 0 {
				return false
			}
			depth--
		}
		if t.Type == token.SEMI && depth == 0 {
			return true
		}
	}
	return false
}

func (p *Parser) parseLoopHeader() (ast.LoopKind, ast.Statement, ast.Expression, ast.Statement) {
	if p.headerHasSemicolon() {
		init := p.parseAssignOrInputStatement() // advances past itself, landing on ';'
		if !p.curTokenIs(token.SEMI) {
			p.errorf("a counting loop is written [loop][i = 0; i < 10; i = i + 1]: expected ';' after the start, found %s", describe(p.curToken))
			return ast.LoopCStyle, init, nil, nil
		}
		p.nextToken() // ';' -> first token of condition
		cond := p.parseExpression(LOWEST)
		if !p.expectPeek(token.SEMI) {
			return ast.LoopCStyle, init, cond, nil
		}
		p.nextToken() // ';' -> first token of post clause
		post := p.parsePostClause()
		return ast.LoopCStyle, init, cond, post
	}
	cond := p.parseExpression(LOWEST)
	return ast.LoopWhile, nil, cond, nil
}

// parsePostClause parses "i++", "i--", or "i = expr" and leaves curToken
// on its own last token (expression-style), since the caller still needs
// to check peek for the header's closing ']'.
func (p *Parser) parsePostClause() ast.Statement {
	tok := p.curToken
	if p.curTokenIs(token.IDENT) && (p.peekTokenIs(token.INCR) || p.peekTokenIs(token.DECR)) {
		name := p.curToken.Literal
		op := "+"
		if p.peekToken.Type == token.DECR {
			op = "-"
		}
		p.nextToken() // name -> ++/--
		return &ast.AssignStatement{
			Token: tok, Name: name,
			Value: &ast.InfixExpression{
				Token: tok, Operator: op,
				Left:  &ast.Identifier{Token: tok, Value: name},
				Right: &ast.IntegerLiteral{Token: tok, Value: 1},
			},
		}
	}
	name := p.curToken.Literal
	p.nextToken() // name -> '='
	p.nextToken() // '=' -> first token of value
	val := p.parseExpression(LOWEST)
	return &ast.AssignStatement{Token: tok, Name: name, Value: val}
}

// ---- files --------------------------------------------------------------

// parsePathExpression parses a file path starting at curToken: any
// expression that builds one (folder + "/" + name + ".txt", names at
// get[0], a call), a quoted string, a variable, or a bareword like
// "file.txt" / "data/in.csv" reconstructed from the IDENT/PERIOD/SLASH/
// MINUS/INT tokens the lexer split it into (or just "." on its own, for
// [directory]).
func (p *Parser) parsePathExpression() ast.Expression {
	tok := p.curToken
	if p.pathIsExpression() {
		return p.parseExpression(LOWEST)
	}
	if p.curTokenIs(token.STRING) {
		return p.stringExpression(tok)
	}
	if p.curTokenIs(token.RAWSTRING) {
		return &ast.StringLiteral{Token: tok, Value: tok.Literal}
	}
	if tok.Type == token.IDENT && !p.peekTokenIs(token.PERIOD) && !p.peekTokenIs(token.SLASH) && !p.peekTokenIs(token.MINUS) {
		// A lone word: a variable holding the path if one exists when this
		// runs, else the literal filename (see Interpreter.evalPath).
		return &ast.Identifier{Token: tok, Value: tok.Literal}
	}
	var sb strings.Builder
	sb.WriteString(tok.Literal)
	for p.peekTokenIs(token.PERIOD) || p.peekTokenIs(token.SLASH) || p.peekTokenIs(token.MINUS) {
		p.nextToken()
		sb.WriteString(p.curToken.Literal)
		if p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.INT) {
			p.nextToken()
			sb.WriteString(p.curToken.Literal)
		}
	}
	return &ast.StringLiteral{Token: tok, Value: sb.String()}
}

// pathIsExpression reports whether the path at curToken is an expression:
// a word or string followed, on its line, by +, [, at or of. Anything else
// reads as before, so data/out-1.csv is still a file name, not a sum.
func (p *Parser) pathIsExpression() bool {
	if !p.curTokenIs(token.IDENT) && !p.curTokenIs(token.STRING) && !p.curTokenIs(token.RAWSTRING) {
		return false
	}
	if p.peekToken.Line != p.curToken.Line {
		return false
	}
	switch p.peekToken.Type {
	case token.PLUS, token.LBRACKET, token.AT, token.OF:
		return true
	}
	return false
}

func (p *Parser) parseFileReadStatement() ast.Statement {
	tok := p.curToken // '['
	p.nextToken()     // '[' -> READ
	p.nextToken()     // READ -> ']'
	if !p.curTokenIs(token.RBRACKET) {
		p.errorf("expected ']' after [read, found %s: write [read] file to name [end]", describe(p.curToken))
		return nil
	}
	p.nextToken() // -> first token of path
	path := p.parsePathExpression()
	if !p.expectPeek(token.TO) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	p.notPrivate(p.curToken, "a variable")
	varName := p.curToken.Literal
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	if !p.expectPeek(token.END) {
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	p.nextToken()
	return &ast.FileReadStatement{Token: tok, File: path, Var: varName}
}

func (p *Parser) parseFileWriteStatement(isAppend bool) ast.Statement {
	tok := p.curToken // '['
	p.nextToken()     // '[' -> WRITE/APPEND
	p.nextToken()     // -> ']'
	if !p.curTokenIs(token.RBRACKET) {
		p.errorf("expected ']' after [write or [append, found %s: write [write] file ... [end]", describe(p.curToken))
		return nil
	}
	p.nextToken() // -> first token of path
	path := p.parsePathExpression()
	p.nextToken() // -> first token of body (or straight to "[end]")

	var items []ast.ContentItem
	for !(p.curTokenIs(token.LBRACKET) && p.peekTokenIs(token.END)) && !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.RAWSTRING:
			items = append(items, ast.ContentItem{Literal: p.curToken.Literal})
		case token.STRING:
			switch e := p.stringExpression(p.curToken).(type) {
			case *ast.StringLiteral:
				items = append(items, ast.ContentItem{Literal: e.Value})
			default:
				items = append(items, ast.ContentItem{Expr: e})
			}
		case token.IDENT:
			items = append(items, ast.ContentItem{Name: p.curToken.Literal, IsVar: true})
		default:
			// punctuation between items (commas, etc.) is ignored.
		}
		p.nextToken()
	}
	if !p.curTokenIs(token.LBRACKET) {
		p.unclosedError(tok, "this block", "[end]")
		return nil
	}
	if !p.expectPeek(token.END) {
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	p.nextToken()
	return &ast.FileWriteStatement{Token: tok, File: path, Content: items, Append: isAppend}
}

func (p *Parser) parseDirectoryStatement() ast.Statement {
	tok := p.curToken // '['
	p.nextToken()     // '[' -> DIRECTORY
	p.nextToken()     // -> ']'
	if !p.curTokenIs(token.RBRACKET) {
		p.errorf("expected ']' after [directory, found %s", describe(p.curToken))
		return nil
	}
	p.nextToken() // -> first token of path
	path := p.parsePathExpression()
	if !p.expectPeek(token.TO) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	p.notPrivate(p.curToken, "a variable")
	varName := p.curToken.Literal
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	if !p.expectPeek(token.END) {
		return nil
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	p.nextToken()
	return &ast.DirectoryStatement{Token: tok, Path: path, Var: varName}
}

// ---- expressions ---------------------------------------------------

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noValueError()
		return nil
	}
	left := prefix()

	for precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return left
		}
		p.nextToken()
		left = infix(left)
	}
	return left
}

// parseIdentifier handles both a plain variable reference and a
// function call ("name[args]"). The two are ambiguous at the token
// level whenever an identifier is immediately followed by '[' — which
// also happens at the end of an if-condition, since if-headers close
// with a bare '[' (reversed-bracket syntax: "if ] cond ["). Real call
// arguments always sit on the same physical line as the call's '[';
// a header's closing '[' is always the last token on its line. That
// line-boundary is what tells the two apart here.
//
// Two identifiers side by side on one line are either a name qualified by
// its module ("time now[]") or a sentence-style call ("nums process f",
// meaning process[nums, f]). The parser can't tell which — that depends
// on whether the first name is an imported module or a variable — so both
// become a node with Module set and the evaluator decides. Neither form
// was otherwise valid: every word that can follow a name (is, at, to,
// of, ...) is a keyword, not an IDENT.
//
// "x give ..." is a one-parameter anonymous function.
func (p *Parser) parseIdentifier() ast.Expression {
	tok := p.curToken
	if p.startsPhrase() {
		return p.parseTheoryCall()
	}
	p.usedBeforeTheory()
	if p.startsRandom() {
		return p.parseRandomExpression()
	}
	if p.startsMatrix() {
		return p.parseMatrixLiteral()
	}
	// The keyword was "gives" before v0.9.150.
	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "gives" && p.peekToken.Line == tok.Line {
		p.errorf("%s gives ...: the word is give now: %s give ...", tok.Literal, tok.Literal)
	}
	if p.peekTokenIs(token.OF) && p.peekToken.Line == tok.Line {
		return p.parseFieldExpression()
	}
	if p.peekTokenIs(token.GIVES) && p.peekToken.Line == tok.Line {
		p.nextToken()
		return p.parseFunctionBody(tok, []string{tok.Literal})
	}
	if tok.Literal == "here" {
		p.hereUses++
	}
	module := ""
	if p.isQualifiedName() {
		module = tok.Literal
		p.nextToken()
	}
	name := p.curToken
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == name.Line && !p.bracketStartsFunction(1) && p.peekN(2).Line == p.peekToken.Line {
		p.nextToken()
		args := p.parseCallArguments()
		return p.maybeSentence(&ast.CallExpression{Token: tok, Module: module, Name: name.Literal, Arguments: args})
	}
	if module != "" && p.peekStartsArgument() {
		var args []ast.Expression
		p.nextToken()
		args = append(args, p.parseExpression(SENTENCEARG))
		for !p.inBrackets && p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			args = append(args, p.parseExpression(SENTENCEARG))
		}
		return &ast.CallExpression{Token: tok, Module: module, Name: name.Literal, Arguments: args}
	}
	return &ast.Identifier{Token: tok, Module: module, Value: name.Literal}
}

// peekStartsArgument reports whether the next token, on the same line,
// can begin a sentence-style call's first argument. A '-' counts only
// when it's spaced from the name but attached to what follows: "s get -1"
// passes -1, while "a b - 1" and "a b-1" stay subtractions.
func (p *Parser) peekStartsArgument() bool {
	return p.argumentStartsAt(1)
}

// argumentStartsAt is peekStartsArgument for the token i places ahead.
func (p *Parser) argumentStartsAt(i int) bool {
	t := p.peekN(i)
	if t.Line != p.peekN(i-1).Line {
		return false
	}
	if t.Type == token.MINUS {
		next := p.peekN(i + 1)
		return t.SpaceBefore && !next.SpaceBefore && next.Line == t.Line
	}
	switch t.Type {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.RAWSTRING, token.TRUE, token.FALSE,
		token.NONE, token.LPAREN, token.LIST, token.SET, token.MAP, token.CHANGE,
		token.MIN, token.MAX, token.LENGTH, token.BANG:
		return true
	case token.LBRACKET:
		return p.bracketStartsFunction(i)
	}
	return false
}

// bracketStartsFunction reports whether the '[' at peekN(i) opens an
// anonymous function's parameter list: "[a, b] give", or "[] give" for
// none.
func (p *Parser) bracketStartsFunction(i int) bool {
	if p.peekN(i).Type != token.LBRACKET {
		return false
	}
	if p.peekN(i+1).Type == token.RBRACKET {
		return p.peekN(i+2).Type == token.GIVES
	}
	for i++; ; i += 2 {
		if p.peekN(i).Type != token.IDENT {
			return false
		}
		switch p.peekN(i + 1).Type {
		case token.COMMA:
			continue
		case token.RBRACKET:
			return p.peekN(i+2).Type == token.GIVES
		default:
			return false
		}
	}
}

// parseBracketFunctionLiteral parses "[a, b] give ...". curToken is '['.
func (p *Parser) parseBracketFunctionLiteral() ast.Expression {
	tok := p.curToken
	if !p.bracketStartsFunction(0) {
		p.errorf("unexpected '[' in expression (a list needs the list keyword: list [...])")
		return nil
	}
	var params []string
	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
	}
	for !p.curTokenIs(token.RBRACKET) {
		p.nextToken()
		p.notPrivate(p.curToken, "a parameter")
		params = append(params, p.curToken.Literal)
		p.nextToken()
	}
	p.nextToken() // ']' -> GIVES
	return p.parseFunctionBody(tok, params)
}

// parseFunctionBody parses what follows "give" (curToken): an expression
// on the same line, or, if "give" ends its line, a block of statements
// closed by "give [end]" — the same shape as a def body.
func (p *Parser) parseFunctionBody(tok token.Token, params []string) ast.Expression {
	if p.peekTokenIs(token.EOF) || p.peekToken.Line != p.curToken.Line {
		body := p.parseBlockUntil(p.isGivesEnd)
		if !p.curTokenIs(token.GIVES) {
			p.unclosedError(tok, "this give", "give [end]")
			return nil
		}
		p.nextToken() // -> '['
		p.nextToken() // -> END
		p.nextToken() // -> ']'
		return &ast.FunctionLiteral{Token: tok, Parameters: params, Body: body}
	}
	p.nextToken()
	gtok := p.curToken
	expr := p.parseExpression(LOWEST)
	body := &ast.BlockStatement{Token: gtok, Statements: []ast.Statement{&ast.ReturnStatement{Token: gtok, Value: expr}}}
	return &ast.FunctionLiteral{Token: tok, Parameters: params, Body: body}
}

func (p *Parser) isGivesEnd() bool {
	return p.curTokenIs(token.GIVES) &&
		p.peekN(1).Type == token.LBRACKET &&
		p.peekN(2).Type == token.END &&
		p.peekN(3).Type == token.RBRACKET
}

// parseMethodCallExpression parses "x at method" or "x at method[args]"
// inside any expression (curToken is AT). The statement form's
// unbracketed args ("r is x at slice 0, 3 .") are handled by
// parseIsStatement instead.
func (p *Parser) parseMethodCallExpression(receiver ast.Expression) ast.Expression {
	tok := p.curToken
	p.nextToken()
	method := p.curToken.Literal
	mc := &ast.MethodCallExpression{Token: tok, Receiver: receiver, Method: method}
	// Same rule as function calls: real arguments start on the '[' line;
	// a '[' that ends its line closes an if-header ("if ] x at isempty [").
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line && p.peekN(2).Line == p.peekToken.Line {
		p.nextToken()
		mc.Arguments = p.parseExpressionList(token.RBRACKET)
		mc.Bracketed = true
		return mc
	}
	// Without brackets, one argument: the value right after the method,
	// on its line ("2 at pow 10", "s at contains "a""). It's just that
	// value, so "3 at pow 2 == 9" compares the power and
	// "s at contains "a" && ok" asks both; more than one argument needs
	// brackets (x at get[0, 1]). A test or scroll word isn't an argument.
	if (p.curTokenIs(token.IDENT) || slices.Contains(syntax.Methods, method)) && p.argumentStartsAt(1) && !p.isStop(p.peekToken.Literal) && !p.inVerifyValueStop() {
		p.nextToken()
		mc.Arguments = append(mc.Arguments, p.parseExpression(INDEX))
	}
	return mc
}

// inVerifyValueStop keeps "at least" / "at most" in a verify sentence
// from being read as a method's argument.
func (p *Parser) inVerifyValueStop() bool {
	return p.inVerifyValue > 0 && (p.peekToken.Literal == "least" || p.peekToken.Literal == "most")
}

// sortModuleCall reports whether "sort" starts a call into the sort
// library ("sort min_sort[x]") rather than the sort statement
// ("sort nums ."), and if so makes it a name.
func (p *Parser) sortModuleCall() bool {
	if p.curTokenIs(token.SORT) && p.peekTokenIs(token.IDENT) && p.peekToken.Line == p.curToken.Line &&
		p.peekN(2).Type == token.LBRACKET && p.peekN(2).Line == p.curToken.Line {
		p.curToken.Type = token.IDENT
		return true
	}
	return false
}

// parseSortModuleExpression is "sort min_sort[x]" inside an expression.
func (p *Parser) parseSortModuleExpression() ast.Expression {
	if !p.sortModuleCall() {
		p.errorf("sort here must name the sort library, as in sort min_sort[nums] (to sort a list in place, write sort nums . on its own line)")
		return nil
	}
	return p.parseIdentifier()
}

func (p *Parser) isQualifiedName() bool {
	if p.isStop(p.peekToken.Literal) || p.typeCheckAhead() {
		return false
	}
	return p.curTokenIs(token.IDENT) && p.peekTokenIs(token.IDENT) && p.peekToken.Line == p.curToken.Line
}

// parseNegativeNumber reads "-" and the number right after it as one
// literal. curToken is the "-".
func (p *Parser) parseNegativeNumber() ast.Expression {
	p.nextToken()
	tok := p.curToken
	tok.Literal = "-" + tok.Literal
	if tok.Type == token.INT {
		v, err := strconv.ParseInt(tok.Literal, 10, 64)
		if err != nil {
			p.errorf("integer %s is past the integer limits (-9223372036854775808 to 9223372036854775807); write it as a float, e.g. %s.0", tok.Literal, tok.Literal)
			return nil
		}
		return p.maybeSentence(&ast.IntegerLiteral{Token: tok, Value: v})
	}
	v, err := strconv.ParseFloat(tok.Literal, 64)
	if err != nil {
		p.errorf("%q isn't a number Turtle can read", tok.Literal)
		return nil
	}
	return p.maybeSentence(&ast.FloatLiteral{Token: tok, Value: v})
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	v, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errorf("integer %s is past the integer limits (-9223372036854775808 to 9223372036854775807); write it as a float, e.g. %s.0", p.curToken.Literal, p.curToken.Literal)
		return nil
	}
	return p.maybeSentence(&ast.IntegerLiteral{Token: p.curToken, Value: v})
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	v, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errorf("%q isn't a number Turtle can read", p.curToken.Literal)
		return nil
	}
	return p.maybeSentence(&ast.FloatLiteral{Token: p.curToken, Value: v})
}

func (p *Parser) parseStringLiteral() ast.Expression {
	switch {
	case p.curToken.Unclosed && p.multilineString > 0:
		// The quotes paired up wrong further up: point there.
		line, pos := p.multilineString, p.multilinePos
		p.multilineString = 0
		p.errorAt(line, pos, "this text runs over several lines and the quotes don't pair up: is its closing %c missing?", p.l.Source()[pos])
	case p.curToken.Unclosed:
		p.errorAt(p.curToken.Line, p.curToken.Pos, "this text never closes: add the closing %c", p.l.Source()[p.curToken.Pos])
	case strings.Contains(p.curToken.Literal, "\n") && p.multilineString == 0:
		p.multilineString, p.multilinePos = p.curToken.Line, p.curToken.Pos
		p.multilineEnd = p.curToken.Line + strings.Count(p.l.Source()[p.curToken.Pos:p.curToken.End], "\n")
	}
	return p.maybeSentence(p.stringExpression(p.curToken))
}

// stringExpression turns a STRING token into a StringLiteral, or an
// InterpolatedString when it has {expr} parts: "Total: {qty * price}".
// A '{' is a plain brace when the next non-space character is a quote or
// '}', or nothing follows: JSON ('{"a": 1}', "{}") never needs escaping,
// and an expression never starts with a quote. \{ is always a plain brace.
func (p *Parser) stringExpression(tok token.Token) ast.Expression {
	text := tok.Literal
	literal := func(s string) *ast.StringLiteral {
		return &ast.StringLiteral{Token: tok, Value: strings.ReplaceAll(s, string(token.LiteralBrace), "{")}
	}
	if !strings.Contains(text, "{") {
		return literal(text)
	}
	var parts []ast.Expression
	plain := "" // text so far that's literal, including plain braces
	for {
		open := strings.IndexByte(text, '{')
		if open < 0 {
			break
		}
		if plainBrace(text[open+1:]) {
			plain += text[:open+1]
			text = text[open+1:]
			continue
		}
		if s := plain + text[:open]; s != "" {
			parts = append(parts, literal(s))
		}
		plain = ""
		end := strings.IndexByte(text[open:], '}')
		if end < 0 {
			p.errorAt(tok.Line, tok.Pos, "a '{' in a string needs a closing '}' (write \\{ for a plain brace)")
			return literal(tok.Literal)
		}
		parts = append(parts, p.interpolatedPart(tok, text[open+1:open+end]))
		text = text[open+end+1:]
	}
	if len(parts) == 0 {
		return literal(plain + text)
	}
	if s := plain + text; s != "" {
		parts = append(parts, literal(s))
	}
	return &ast.InterpolatedString{Token: tok, Parts: parts}
}

// plainBrace reports whether a '{' followed by rest is a plain brace: the
// next non-space character is a quote or '}', or there's none.
func plainBrace(rest string) bool {
	rest = strings.TrimLeft(rest, " \t\r\n")
	return rest == "" || rest[0] == '"' || rest[0] == '\'' || rest[0] == '}'
}

// interpolatedPart parses the expression inside one {...} of a string.
func (p *Parser) interpolatedPart(tok token.Token, src string) ast.Expression {
	fail := func(why string) ast.Expression {
		p.errorAt(tok.Line, tok.Pos, "{%s} in a string: %s", src, why)
		return &ast.StringLiteral{Token: tok}
	}
	if strings.TrimSpace(src) == "" {
		return fail("put a name or expression inside the braces")
	}
	sub := New(lexer.New(src))
	if sub.curTokenIs(token.EOF) {
		return fail("put a name or expression inside the braces")
	}
	expr := sub.parseExpression(LOWEST)
	if len(sub.errors) > 0 {
		if strings.Contains(sub.errors[0], "end of this line") || strings.Contains(sub.errors[0], "end of the file") {
			return fail("the expression isn't finished")
		}
		return fail(strings.TrimPrefix(sub.errors[0], "line 1: "))
	}
	if !sub.peekTokenIs(token.EOF) {
		return fail(fmt.Sprintf("unexpected %q", sub.peekToken.Literal))
	}
	return expr
}

// maybeSentence turns a literal or a call result directly followed by a
// name on the same line into a sentence-style call with it as the
// subject: "lo" isinstring line is isinstring["lo", line], and
// copy[nums] process f is process[copy[nums], f]. A value followed by a
// name was never otherwise valid.
func (p *Parser) maybeSentence(subject ast.Expression) ast.Expression {
	if !p.peekTokenIs(token.IDENT) || p.peekToken.Line != p.curToken.Line {
		return subject
	}
	if p.isStop(p.peekToken.Literal) || p.typeCheckAhead() {
		return subject // "to 99.99 rounded to 2", "nums each x give ...", "x type list"
	}
	tok := p.curToken
	p.nextToken()
	name := p.curToken.Literal
	var args []ast.Expression
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line && !p.bracketStartsFunction(1) {
		p.nextToken()
		args = p.parseCallArguments()
	} else if p.peekStartsArgument() {
		p.nextToken()
		args = append(args, p.parseExpression(SENTENCEARG))
		for !p.inBrackets && p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			args = append(args, p.parseExpression(SENTENCEARG))
		}
	}
	return &ast.CallExpression{Token: tok, Subject: subject, Name: name, Arguments: args}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNone() ast.Expression {
	return &ast.NoneLiteral{Token: p.curToken}
}

// attachMethod applies call to the last value of e: the right side of an
// operator, inside a prefix (! or -), but not inside ( ).
func attachMethod(e ast.Expression, call func(ast.Expression) ast.Expression) ast.Expression {
	switch x := e.(type) {
	case *ast.InfixExpression:
		if !x.Grouped {
			c := *x
			c.Right = attachMethod(x.Right, call)
			return &c
		}
	case *ast.PrefixExpression:
		if !x.Grouped {
			c := *x
			c.Right = attachMethod(x.Right, call)
			return &c
		}
	}
	return call(e)
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	tok := p.curToken
	// -7 is one number, so "-7 at abs" is 7 (while "-x at abs" is
	// -(x at abs), as in most languages).
	if tok.Type == token.MINUS && (p.peekTokenIs(token.INT) || p.peekTokenIs(token.FLOAT)) && !p.peekToken.SpaceBefore {
		return p.parseNegativeNumber()
	}
	p.nextToken()
	right := p.parseExpression(PREFIX)
	// "!" applies to a whole method call: "!r at isempty" is
	// !(r at isempty), like Python's "not r.isempty()". (The is-statement
	// form with unbracketed args is handled in parseIsStatement.)
	if tok.Type == token.BANG && !p.inIsReceiver {
		for p.peekTokenIs(token.AT) {
			p.nextToken()
			right = p.parseMethodCallExpression(right)
		}
	}
	return &ast.PrefixExpression{Token: tok, Operator: tok.Literal, Right: right}
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	tok := p.curToken
	prec := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(prec)
	p.checkMethodOnRight(tok, left, right)
	return &ast.InfixExpression{Token: tok, Left: left, Operator: tok.Literal, Right: right}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	saved := p.inIsReceiver
	p.inIsReceiver = false // inside ( ), every "at" is an ordinary method
	expr := p.parseExpression(LOWEST)
	p.inIsReceiver = saved
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	switch e := expr.(type) {
	case *ast.InfixExpression:
		e.Grouped = true
	case *ast.PrefixExpression:
		e.Grouped = true
	}
	return expr
}

func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = true
	var list []ast.Expression
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

// parseCallArguments is a call's [ ... ]: expressions, or key: value
// pairs, which are one map argument: options["--out": "a.csv", "-v": false]
// is options[map ["--out": "a.csv", "-v": false]].
func (p *Parser) parseCallArguments() []ast.Expression {
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = true
	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return nil
	}
	p.nextToken()
	first := p.parseExpression(LOWEST)
	if !p.peekTokenIs(token.COLON) {
		list := []ast.Expression{first}
		for p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			list = append(list, p.parseExpression(LOWEST))
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		return list
	}
	m := &ast.MapLiteral{Token: token.Token{Type: token.MAP, Literal: "map", Line: p.curToken.Line}}
	k := first
	for {
		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		m.Keys = append(m.Keys, k)
		m.Values = append(m.Values, p.parseExpression(LOWEST))
		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
		k = p.parseExpression(LOWEST)
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return []ast.Expression{m}
}

func (p *Parser) parseListLiteral() ast.Expression {
	tok := p.curToken
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	elems := p.parseExpressionList(token.RBRACKET)
	return p.maybeSentence(&ast.ListLiteral{Token: tok, Elements: elems})
}

func (p *Parser) parseSetLiteral() ast.Expression {
	tok := p.curToken
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	elems := p.parseExpressionList(token.RBRACKET)
	return p.maybeSentence(&ast.SetLiteral{Token: tok, Elements: elems})
}

func (p *Parser) parseMapLiteral() ast.Expression {
	tok := p.curToken
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	m := &ast.MapLiteral{Token: tok}
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = true
	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return p.maybeSentence(m)
	}
	p.nextToken()
	for {
		k := p.parseExpression(LOWEST)
		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		v := p.parseExpression(LOWEST)
		m.Keys = append(m.Keys, k)
		m.Values = append(m.Values, v)
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			continue
		}
		break
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return p.maybeSentence(m)
}

func (p *Parser) parseMinMaxLength() ast.Expression {
	tok := p.curToken
	if !p.expectPeek(token.OF) {
		return nil
	}
	p.nextToken()
	// PREFIX, like "field of x": "length of a == 0" is (length of a) == 0,
	// and "length of a + 1" is (length of a) + 1.
	arg := p.parseExpression(PREFIX)
	switch tok.Type {
	case token.MIN:
		return &ast.MinExpression{Token: tok, Arg: arg}
	case token.MAX:
		return &ast.MaxExpression{Token: tok, Arg: arg}
	default:
		return &ast.LengthExpression{Token: tok, Arg: arg}
	}
}

// ---- assembled types ----------------------------------------------------

// parseAssembleStatement parses "assemble Order [item, qty, price]".
func (p *Parser) parseAssembleStatement() ast.Statement {
	tok := p.curToken
	if p.isReservedWord(p.peekToken) {
		p.reservedNameError(p.peekToken, "an assembled type")
		p.nextToken()
	} else if !p.expectPeek(token.IDENT) {
		return nil
	}
	p.notPrivate(p.curToken, "an assembled type")
	name := p.curToken.Literal
	if !p.expectPeek(token.LBRACKET) {
		return nil
	}
	var fields []string
	seen := map[string]bool{}
	for !p.peekTokenIs(token.RBRACKET) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		if p.isReservedWord(p.curToken) {
			p.reservedNameError(p.curToken, "a field")
		} else if !p.curTokenIs(token.IDENT) {
			p.errorf("assemble %s: expected a field name, found %s", name, describe(p.curToken))
			p.skipLine()
			return nil
		}
		p.notPrivate(p.curToken, "a field")
		f := p.curToken.Literal
		if seen[f] {
			p.errorf("assemble %s: field %q listed twice", name, f)
		}
		seen[f] = true
		fields = append(fields, f)
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	p.nextToken() // -> ']'
	p.nextToken()
	if p.curTokenIs(token.PERIOD) {
		p.nextToken()
	}
	return &ast.AssembleStatement{Token: tok, Name: name, Fields: fields}
}

// parseFieldExpression parses "field of <value>" (curToken is the field
// name). The value binds tightly, so "qty of o * price of o" is
// (qty of o) * (price of o), and "x of p of line" is x of (p of line).
//
// Indexing belongs to the collection (see INDEX): in "title of books at
// get[0]" the get picks a book, and title is that book's. Other methods
// apply to the field's value: "name of p at upper" is the name, upper-cased.
func (p *Parser) parseFieldExpression() ast.Expression {
	tok := p.curToken
	p.nextToken() // -> OF
	p.nextToken() // -> first token of the value
	saved := p.inFieldObject
	p.inFieldObject = true
	obj := p.parseExpression(PREFIX)
	p.inFieldObject = saved
	if mc, ok := obj.(*ast.MethodCallExpression); ok && (mc.Method == "get" || mc.Method == "slice") {
		if id, ok := mc.Receiver.(*ast.Identifier); ok {
			mc.Hint = fmt.Sprintf("in %s of %s at %s[...], the %s works on %s first; to take part of %s of %s, name it first: v = %s of %s, then v at %s[...]",
				tok.Literal, id.Value, mc.Method, mc.Method, id.Value, tok.Literal, id.Value, tok.Literal, id.Value, mc.Method)
		}
	}
	return &ast.FieldExpression{Token: tok, Field: tok.Literal, Object: obj}
}

// isIndexMethod reports whether tok names a method that picks out part of
// a collection (get, slice) rather than working on a value.
func isIndexMethod(tok token.Token) bool {
	return tok.Type == token.IDENT && (tok.Literal == "get" || tok.Literal == "slice")
}

// parseFieldStatement parses "qty of o = <expr>" — changing one field.
func (p *Parser) parseFieldStatement() ast.Statement {
	tok := p.curToken
	target := p.parseFieldExpression().(*ast.FieldExpression)
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken()
	val := p.parseExpression(LOWEST)
	p.nextToken()
	if p.curTokenIs(token.PERIOD) {
		p.nextToken()
	}
	return &ast.FieldAssignStatement{Token: tok, Target: target, Value: val}
}

// ---- reserved words used as names --------------------------------------

// isReservedWord reports whether tok is a keyword (show, list, max, ...)
// rather than a name.
func (p *Parser) isReservedWord(tok token.Token) bool {
	return tok.Type != token.IDENT && token.LookupIdent(tok.Literal) != token.IDENT
}

func (p *Parser) reservedNameError(tok token.Token, what string) {
	p.errorAt(tok.Line, tok.Pos, "%q is a reserved word, so it can't be used as %s name — pick another name (e.g. %s_value)",
		tok.Literal, what, tok.Literal)
}

// checkName reports a reserved word, or a private (~) name, where a
// name is expected.
func (p *Parser) checkName(tok token.Token, what string) {
	if p.isReservedWord(tok) {
		p.reservedNameError(tok, what)
	}
	p.notPrivate(tok, what)
}

// notPrivate reports a ~name used for something other than a function:
// ~ marks a function private to its file. A file's variables are private
// to it already, and its assembled types stay public, since the values
// it gives back are its users' to work with.
func (p *Parser) notPrivate(tok token.Token, what string) {
	if len(tok.Literal) > 1 && tok.Literal[0] == '~' {
		p.errorAt(tok.Line, tok.Pos, "%s can't be %s name: ~ marks a private function (def %s[...]), and only a function's name can start with it", tok.Literal, what, tok.Literal)
	}
}

// skipLine moves past the rest of the current line, to resume parsing at
// the next statement after an error.
func (p *Parser) skipLine() {
	line := p.curToken.Line
	for !p.peekTokenIs(token.EOF) && p.peekToken.Line == line {
		p.nextToken()
	}
	p.nextToken()
}

// ParseExpressionOnly parses the whole input as one expression (the REPL
// shows a lone expression's value: "1 + 2", "nums"), allowing a closing
// period. It returns nil, with errors, if the input is anything else.
func (p *Parser) ParseExpressionOnly() ast.Expression {
	if p.curTokenIs(token.EOF) {
		p.errorf("nothing to work out")
		return nil
	}
	e := p.parseExpression(LOWEST)
	if p.peekTokenIs(token.PERIOD) {
		p.nextToken()
	}
	if !p.peekTokenIs(token.EOF) {
		p.errorf("unexpected %q after the expression", p.peekToken.Literal)
		return nil
	}
	return e
}

// typeCheckAhead: the next words are "type <kind>" on this line, as in
// "o type Order" or "x type list". Elsewhere type is an ordinary name.
func (p *Parser) typeCheckAhead() bool {
	if !p.peekTokenIs(token.IDENT) || p.peekToken.Literal != "type" || p.peekToken.Line != p.curToken.Line {
		return false
	}
	k := p.peekN(2)
	if k.Line != p.peekToken.Line {
		return false
	}
	switch k.Type {
	case token.IDENT, token.LIST, token.SET, token.MAP, token.NONE:
		return true
	}
	return false
}

// parseTypeCheck parses "type <kind>" after a value; curToken is type.
func (p *Parser) parseTypeCheck(left ast.Expression) ast.Expression {
	tok := p.curToken
	p.nextToken()
	kind := p.curToken.Literal
	switch p.curToken.Type {
	case token.LIST:
		kind = "list"
	case token.SET:
		kind = "set"
	case token.MAP:
		kind = "map"
	case token.NONE:
		kind = "none"
	}
	return &ast.TypeCheckExpression{Token: tok, Value: left, Kind: kind}
}
