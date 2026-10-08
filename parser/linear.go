package parser

import (
	"Turtle/ast"
	"Turtle/token"
)

// After "import linear", matrix is a word of the language: matrix [...]
// makes a matrix, its rows ending at ; or at the end of a line:
//
//	a = matrix [1, 2; 3, 4]
//	b = matrix [
//	    1, 2, 3
//	    4, 5, 6
//	]
//
// A comma at the end of a line carries the row on to the next one, so a
// long row can wrap. Elsewhere matrix is an ordinary name.

// startsMatrix reports whether curToken begins a matrix literal.
func (p *Parser) startsMatrix() bool {
	return p.linearImported && p.curTokenIs(token.IDENT) && p.curToken.Literal == "matrix"
}

// matrixNameError reports matrix used as a name in a file that imports
// linear.
func (p *Parser) matrixNameError(tok token.Token, what string) {
	p.errorAt(tok.Line, tok.Pos, "matrix is a word of the linear library in a file that imports linear, so it can't name %s here; rename it (a matrix is made with matrix [1, 2; 3, 4])", what)
}

func (p *Parser) parseMatrixLiteral() ast.Expression {
	tok := p.curToken
	if !p.peekTokenIs(token.LBRACKET) {
		p.matrixNameError(tok, "a value")
		return nil
	}
	p.nextToken() // -> [
	ml := &ast.MatrixLiteral{Token: tok}
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = true
	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return p.maybeSentence(ml)
	}
	var row []ast.Expression
	for {
		p.nextToken()
		if p.curTokenIs(token.SEMI) || p.curTokenIs(token.RBRACKET) || p.curTokenIs(token.EOF) {
			p.errorf("a number is missing in this matrix row")
			return nil
		}
		e := p.parseExpression(LOWEST)
		if e == nil {
			return nil
		}
		row = append(row, e)
		switch {
		case p.peekTokenIs(token.COMMA):
			p.nextToken() // the row goes on, even past the end of the line
			continue
		case p.peekTokenIs(token.SEMI):
			p.nextToken()
			ml.Rows = append(ml.Rows, row)
			row = nil
			if p.peekTokenIs(token.RBRACKET) { // matrix [1, 2; 3, 4;]
				p.nextToken()
				return p.finishMatrix(ml)
			}
			continue
		case p.peekTokenIs(token.RBRACKET):
			p.nextToken()
			ml.Rows = append(ml.Rows, row)
			return p.finishMatrix(ml)
		case p.peekToken.Line > p.endLine(p.curToken) && !p.peekTokenIs(token.EOF):
			ml.Rows = append(ml.Rows, row)
			row = nil
			continue
		}
		p.expectError(token.RBRACKET)
		return nil
	}
}

// finishMatrix checks that every row is as long as the first.
func (p *Parser) finishMatrix(ml *ast.MatrixLiteral) ast.Expression {
	for i, row := range ml.Rows {
		if len(row) != len(ml.Rows[0]) {
			p.errorAt(ml.Token.Line, ml.Token.Pos, "every row of a matrix needs the same count of numbers: row 1 has %d, row %d has %d", len(ml.Rows[0]), i+1, len(row))
			return nil
		}
	}
	return p.maybeSentence(ml)
}
