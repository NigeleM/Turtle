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
	"Turtle/token"
)

const (
	LOWEST int = iota
	METHOD     // "x at m": binds loosest, so "a + b at upper" is (a + b) at upper
	OR
	AND
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	INDEX // "x at get[i]" / "x at slice[...]": picks part of the value right before it
)

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
	token.PERCENT:  PRODUCT,
}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

type Parser struct {
	l *lexer.Lexer

	curToken  token.Token
	peekToken token.Token
	buf       []token.Token

	errors []string

	// inBrackets is true while parsing a comma-separated list inside
	// [...] (call arguments, list/set/map literals). There a sentence-style
	// call takes only one argument, so the list's own commas aren't
	// swallowed: check["x", t find "W", 7] is check["x", find[t, "W"], 7].
	inBrackets bool

	// inIsReceiver is true while parsing the receiver of
	// "r is x at m args ." at its top level, where the statement itself
	// handles "at" and its unbracketed args.
	inIsReceiver bool

	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	p.prefixParseFns = map[token.Type]prefixParseFn{
		token.IDENT:    p.parseIdentifier,
		token.INT:      p.parseIntegerLiteral,
		token.FLOAT:    p.parseFloatLiteral,
		token.STRING:   p.parseStringLiteral,
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
	}
	p.infixParseFns = map[token.Type]infixParseFn{
		token.PLUS:     p.parseInfixExpression,
		token.MINUS:    p.parseInfixExpression,
		token.ASTERISK: p.parseInfixExpression,
		token.SLASH:    p.parseInfixExpression,
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
	}

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf("line %d: %s", p.curToken.Line, fmt.Sprintf(format, args...)))
}

func (p *Parser) nextToken() {
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
	p.errorf("expected next token to be %s, got %s (%q) instead", t, p.peekToken.Type, p.peekToken.Literal)
	return false
}

func (p *Parser) peekPrecedence() int {
	if p.peekToken.Line != p.curToken.Line {
		return LOWEST
	}
	// get and slice pick part of the value right before them, so they
	// bind tightest: "10 + row at get["q"] * 2" is 10 + (row at get["q"]) * 2,
	// and "title of books at get[0]" is the title of books at get[0].
	// Other methods bind loosest: "a + b at upper" is (a + b) at upper.
	// Only the bracketed form: "r is nums at get 0 ." is the statement form.
	if p.peekTokenIs(token.AT) && isIndexMethod(p.peekN(2)) &&
		p.peekN(3).Type == token.LBRACKET && p.peekN(3).Line == p.peekToken.Line {
		return INDEX
	}
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
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
	return program
}

// parseBlockUntil parses statements starting just after curToken (the
// block's own opening token) until stop() reports true. It never
// consumes whatever satisfies stop(); the caller inspects/consumes it.
func (p *Parser) parseBlockUntil(stop func() bool) *ast.BlockStatement {
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = false // a block inside brackets ([x] gives ...) is ordinary code
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
	default:
		p.errorf("unexpected token %s (%q) at start of statement", p.curToken.Type, p.curToken.Literal)
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
		p.errorf("unexpected '[' at start of statement (followed by %s)", p.peekToken.Type)
		p.nextToken()
		return nil
	}
}

// ---- assignment / input / is / bare call --------------------------------

func (p *Parser) parseIdentifierLeadStatement() ast.Statement {
	if p.peekTokenIs(token.OF) && p.peekToken.Line == p.curToken.Line {
		return p.parseFieldStatement()
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
	p.errorf("unexpected identifier %q at start of statement", p.curToken.Literal)
	p.nextToken()
	return nil
}

// parseAssignOrInputStatement parses "name = expr" and
// "name = ? \"prompt\"". curToken is the name; peek is '='.
func (p *Parser) parseAssignOrInputStatement() ast.Statement {
	tok := p.curToken
	name := p.curToken.Literal
	p.nextToken() // name -> '='

	if p.peekTokenIs(token.QUESTION) {
		p.nextToken() // '=' -> '?'
		if !p.expectPeek(token.STRING) {
			return nil
		}
		prompt := p.stringExpression(p.curToken)
		p.nextToken()
		return &ast.InputStatement{Token: tok, Name: name, Prompt: prompt}
	}

	p.nextToken() // '=' -> first token of value
	val := p.parseExpression(LOWEST)
	p.nextToken()
	return &ast.AssignStatement{Token: tok, Name: name, Value: val}
}

// parseIsStatement parses "name is expr ." and
// "name is receiver at method arg, arg ." curToken is name; peek is IS.
func (p *Parser) parseIsStatement() ast.Statement {
	tok := p.curToken
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
		if !p.peekTokenIs(token.PERIOD) && !p.peekTokenIs(token.EOF) {
			p.nextToken()
			args = append(args, p.parseExpression(LOWEST))
			for p.peekTokenIs(token.COMMA) {
				p.nextToken()
				p.nextToken()
				args = append(args, p.parseExpression(LOWEST))
			}
		}
		if !p.requirePeriod() {
			return nil
		}
		// "r is !s at contains "a" ." negates the method call, not s.
		if pre, ok := receiver.(*ast.PrefixExpression); ok && pre.Operator == "!" {
			call := &ast.MethodCallExpression{Token: tok, Receiver: pre.Right, Method: method, Arguments: args}
			return &ast.AssignStatement{Token: tok, Name: name, Value: &ast.PrefixExpression{Token: pre.Token, Operator: "!", Right: call}}
		}
		return &ast.AssignStatement{
			Token: tok, Name: name,
			Value: &ast.MethodCallExpression{Token: tok, Receiver: receiver, Method: method, Arguments: args},
		}
	}

	if !p.requirePeriod() {
		return nil
	}
	return &ast.AssignStatement{Token: tok, Name: name, Value: receiver}
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
	return &ast.ReturnStatement{Token: tok, Value: val}
}

// ---- safe / handle / fail ---------------------------------------------

// errorKinds are the names a handle statement can list; they match the
// kinds the evaluator gives its runtime errors.
var errorKinds = []string{"file", "number", "math", "index", "key", "name", "type", "json", "date", "http", "sql", "csv", "custom"}

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
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpAdd, Target: target, Value: val}
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
	if !p.requirePeriod() {
		return nil
	}
	return &ast.DataOpStatement{Token: tok, Kind: ast.OpInsert, Target: target, Value: val, Index: idx}
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
		p.errorf("expected 'def [end]' to close function %q, got %s (%q)", name, p.curToken.Type, p.curToken.Literal)
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
			p.errorf("expected 'if [end]' to close if-statement, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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
	} else {
		kind, init, cond, post = p.parseLoopHeader()
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	body := p.parseBlockUntil(p.isLoopEnd)
	if !p.curTokenIs(token.LBRACKET) {
		p.errorf("expected '[loop][end]' to close loop, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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
			p.errorf("expected ';' after loop init, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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

// parsePathExpression parses a file path starting at curToken: either a
// quoted string, or a bareword like "file.txt" / "data/in.csv"
// reconstructed from the IDENT/PERIOD/SLASH/MINUS/INT tokens the lexer
// split it into (or just "." on its own, for [directory]).
func (p *Parser) parsePathExpression() ast.Expression {
	tok := p.curToken
	if p.curTokenIs(token.STRING) {
		return p.stringExpression(tok)
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

func (p *Parser) parseFileReadStatement() ast.Statement {
	tok := p.curToken // '['
	p.nextToken()     // '[' -> READ
	p.nextToken()     // READ -> ']'
	if !p.curTokenIs(token.RBRACKET) {
		p.errorf("expected ']' after [read, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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
		p.errorf("expected ']' after [write/[append, got %s (%q)", p.curToken.Type, p.curToken.Literal)
		return nil
	}
	p.nextToken() // -> first token of path
	path := p.parsePathExpression()
	p.nextToken() // -> first token of body (or straight to "[end]")

	var items []ast.ContentItem
	for !(p.curTokenIs(token.LBRACKET) && p.peekTokenIs(token.END)) && !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
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
		p.errorf("expected '[end]' to close file block, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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
		p.errorf("expected ']' after [directory, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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
		p.errorf("no prefix parse function for %s (%q)", p.curToken.Type, p.curToken.Literal)
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
// "x gives ..." is a one-parameter anonymous function.
func (p *Parser) parseIdentifier() ast.Expression {
	tok := p.curToken
	if p.peekTokenIs(token.OF) && p.peekToken.Line == tok.Line {
		return p.parseFieldExpression()
	}
	if p.peekTokenIs(token.GIVES) && p.peekToken.Line == tok.Line {
		p.nextToken()
		return p.parseFunctionBody(tok, []string{tok.Literal})
	}
	module := ""
	if p.isQualifiedName() {
		module = tok.Literal
		p.nextToken()
	}
	name := p.curToken
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == name.Line && !p.bracketStartsFunction(1) && p.peekN(2).Line == p.peekToken.Line {
		p.nextToken()
		args := p.parseExpressionList(token.RBRACKET)
		return p.maybeSentence(&ast.CallExpression{Token: tok, Module: module, Name: name.Literal, Arguments: args})
	}
	if module != "" && p.peekStartsArgument() {
		var args []ast.Expression
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
		for !p.inBrackets && p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			args = append(args, p.parseExpression(LOWEST))
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
	if p.peekToken.Line != p.curToken.Line {
		return false
	}
	if p.peekToken.Type == token.MINUS {
		next := p.peekN(2)
		return p.peekToken.SpaceBefore && !next.SpaceBefore && next.Line == p.peekToken.Line
	}
	switch p.peekToken.Type {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.TRUE, token.FALSE,
		token.NONE, token.LPAREN, token.LIST, token.SET, token.MAP, token.CHANGE,
		token.MIN, token.MAX, token.LENGTH, token.BANG:
		return true
	case token.LBRACKET:
		return p.bracketStartsFunction(1)
	}
	return false
}

// bracketStartsFunction reports whether the '[' at peekN(i) opens an
// anonymous function's parameter list: "[a, b] gives", or "[] gives" for
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

// parseBracketFunctionLiteral parses "[a, b] gives ...". curToken is '['.
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
		params = append(params, p.curToken.Literal)
		p.nextToken()
	}
	p.nextToken() // ']' -> GIVES
	return p.parseFunctionBody(tok, params)
}

// parseFunctionBody parses what follows "gives" (curToken): an expression
// on the same line, or, if "gives" ends its line, a block of statements
// closed by "gives [end]" — the same shape as a def body.
func (p *Parser) parseFunctionBody(tok token.Token, params []string) ast.Expression {
	if p.peekTokenIs(token.EOF) || p.peekToken.Line != p.curToken.Line {
		body := p.parseBlockUntil(p.isGivesEnd)
		if !p.curTokenIs(token.GIVES) {
			p.errorf("expected 'gives [end]' to close the function, got %s (%q)", p.curToken.Type, p.curToken.Literal)
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
	// a '[' that ends its line closes an if-header ("if ] x at isEmpty [").
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line && p.peekN(2).Line == p.peekToken.Line {
		p.nextToken()
		mc.Arguments = p.parseExpressionList(token.RBRACKET)
		mc.Bracketed = true
	}
	return mc
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
	return p.curTokenIs(token.IDENT) && p.peekTokenIs(token.IDENT) && p.peekToken.Line == p.curToken.Line
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
		p.errorf("could not parse %q as float", p.curToken.Literal)
		return nil
	}
	return p.maybeSentence(&ast.FloatLiteral{Token: p.curToken, Value: v})
}

func (p *Parser) parseStringLiteral() ast.Expression {
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
			p.errors = append(p.errors, fmt.Sprintf("line %d: a '{' in a string needs a closing '}' (write \\{ for a plain brace)", tok.Line))
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
		p.errors = append(p.errors, fmt.Sprintf("line %d: {%s} in a string: %s", tok.Line, src, why))
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
		if strings.Contains(sub.errors[0], "EOF") {
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
	tok := p.curToken
	p.nextToken()
	name := p.curToken.Literal
	var args []ast.Expression
	if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line && !p.bracketStartsFunction(1) {
		p.nextToken()
		args = p.parseExpressionList(token.RBRACKET)
	} else if p.peekStartsArgument() {
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
		for !p.inBrackets && p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			args = append(args, p.parseExpression(LOWEST))
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

func (p *Parser) parsePrefixExpression() ast.Expression {
	tok := p.curToken
	p.nextToken()
	right := p.parseExpression(PREFIX)
	// "!" applies to a whole method call: "!r at isEmpty" is
	// !(r at isEmpty), like Python's "not r.isEmpty()". (The is-statement
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
	return &ast.InfixExpression{Token: tok, Left: left, Operator: tok.Literal, Right: right}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	expr := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
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
			p.errorf("expected a field name in assemble %s, got %s (%q)", name, p.curToken.Type, p.curToken.Literal)
			p.skipLine()
			return nil
		}
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
	return &ast.FieldExpression{Token: tok, Field: tok.Literal, Object: p.parseExpression(PREFIX)}
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
	p.errors = append(p.errors, fmt.Sprintf("line %d: %q is a reserved word, so it can't be used as %s name — pick another name (e.g. %s_value)",
		tok.Line, tok.Literal, what, tok.Literal))
}

// checkName reports a reserved word where a name is expected.
func (p *Parser) checkName(tok token.Token, what string) {
	if p.isReservedWord(tok) {
		p.reservedNameError(tok, what)
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
