package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/syntax"
	"Turtle/token"
)

// Theories: a new word, the phrases it's written in, and what it means.
//
//	theory tally
//	    abstract
//	        tally says how many times b appears in the list a.
//	    notation tally b in a .
//	    definition
//	        ...
//	    theorem result >= 0
//	    proof
//	        tally 1 in list [1, 1, 0] . is 2
//	theory [end]
//
// A notation's slots are the names the definition uses; its other words
// are fixed. A theory is defined before its word is used: reading top to
// bottom, the parser learns each theory and from then on reads its
// phrases.

// theorySections are the words a theory is made of.
var theorySections = map[string]bool{"abstract": true, "notation": true, "definition": true, "theorem": true, "proof": true}

// theorySpec is what the parser needs to read a theory's phrases.
type theorySpec struct {
	name      string
	line      int
	from      string // the file it's imported from, "" for this file's own
	notations []*ast.Notation
	slots     []string
	fixed     map[string]bool // every fixed word of every notation (not the word itself)
}

var theoryLine = regexp.MustCompile(`(?m)^[ \t]*theory[ \t]+(~?[A-Za-z_][A-Za-z0-9_]*)[ \t]*(//.*)?\r?$`)

// Theories is what this parser has learned of theories, to hand to the
// next one (the REPL reads each entry with a parser of its own).
type Theories map[string]*theorySpec

// Theories gives the theories read so far.
func (p *Parser) Theories() Theories { return Theories(p.theories) }

// UseTheories starts this parser knowing theories another has read.
func (p *Parser) UseTheories(t Theories) {
	if p.theories == nil {
		p.theories = map[string]*theorySpec{}
	}
	for k, v := range t {
		p.theories[k] = v
	}
}

// scanTheories notes where each theory in the source is defined, so a use
// above its theory can say so.
func (p *Parser) scanTheories() {
	src := p.l.Source()
	for _, m := range theoryLine.FindAllStringSubmatchIndex(src, -1) {
		word := src[m[2]:m[3]]
		if p.laterTheories == nil {
			p.laterTheories = map[string]int{}
		}
		if _, seen := p.laterTheories[word]; !seen {
			p.laterTheories[word] = strings.Count(src[:m[0]], "\n") + 1
		}
	}
}

// theoryOnKeyword reports "theory list": a keyword can't be a theory's
// word.
func (p *Parser) theoryOnKeyword() bool {
	if p.curToken.Literal != "theory" || p.peekToken.Line != p.curToken.Line || p.peekN(2).Line == p.curToken.Line || !p.isReservedWord(p.peekToken) {
		return false
	}
	p.errorAt(p.peekToken.Line, p.peekToken.Pos, "%s is already a word in Turtle (a keyword): a theory needs a new word", p.peekToken.Literal)
	return true
}

// startsTheory: curToken is "theory" at the start of a line, with a name
// after it, alone on the line.
func (p *Parser) startsTheory() bool {
	return p.curToken.Literal == "theory" && p.curToken.Type == token.IDENT && p.peekTokenIs(token.IDENT) &&
		p.peekToken.Line == p.curToken.Line && p.peekN(2).Line != p.curToken.Line
}

// column is how far into its line a token starts.
func (p *Parser) column(tok token.Token) int {
	src := p.l.Source()
	start := strings.LastIndexByte(src[:min(tok.Pos, len(src))], '\n') + 1
	return tok.Pos - start
}

// atSection: curToken starts a line with one of a theory's section words,
// at the sections' own indent, or is "theory [end]".
func (p *Parser) atSection(indent int) bool {
	if p.atTheoryEnd() {
		return true
	}
	return p.curToken.Type == token.IDENT && theorySections[p.curToken.Literal] &&
		p.prevToken.Line != p.curToken.Line && p.column(p.curToken) == indent
}

func (p *Parser) atTheoryEnd() bool {
	return p.curToken.Literal == "theory" && p.peekTokenIs(token.LBRACKET) && p.peekN(2).Type == token.END
}

// reseek restarts reading at line n.
func (p *Parser) reseek(n int) {
	p.l.SeekLine(n)
	p.buf = nil
	p.peekToken = p.l.NextToken()
	p.nextToken()
}

func (p *Parser) parseTheoryStatement() ast.Statement {
	tok := p.curToken
	p.nextToken() // -> the word
	name := p.curToken.Literal
	if len(name) > 1 && name[0] == '~' {
		p.errorAt(p.curToken.Line, p.curToken.Pos, "theory %s: private theories aren't part of Turtle yet", name)
	}
	if other, ok := p.theories[name]; ok {
		if other.from != "" {
			p.errorAt(p.curToken.Line, p.curToken.Pos, "theory %s: %s is already a theory, imported from %s", name, name, other.from)
		} else {
			p.errorAt(p.curToken.Line, p.curToken.Pos, "theory %s: %s is already a theory (line %d)", name, name, other.line)
		}
	}
	ts := &ast.TheoryStatement{Token: tok, Name: name}
	p.nextToken() // -> the first section
	if p.curTokenIs(token.EOF) {
		p.errorAt(tok.Line, tok.Pos, "theory %s is never closed: add theory [end] after its last line", name)
		return nil
	}
	indent := p.column(p.curToken)
	seen := map[string]bool{}
	var spec *theorySpec
	for !p.atTheoryEnd() {
		if p.curTokenIs(token.EOF) {
			p.errorAt(tok.Line, tok.Pos, "theory %s is never closed: add theory [end] after its last line", name)
			return nil
		}
		sec := p.curToken
		if !p.atSection(indent) {
			p.errorAt(sec.Line, sec.Pos, "theory %s: expected abstract, notation, definition, theorem or proof, found %s", name, describe(sec))
			p.skipLine()
			continue
		}
		if seen[sec.Literal] && sec.Literal != "notation" && sec.Literal != "theorem" {
			p.errorAt(sec.Line, sec.Pos, "theory %s has two %ss: write it once", name, sec.Literal)
		}
		seen[sec.Literal] = true
		switch sec.Literal {
		case "abstract":
			ts.Abstract = p.parseAbstract(sec, indent)
		case "notation":
			if n := p.parseNotation(ts); n != nil {
				ts.Notations = append(ts.Notations, n)
			}
		case "definition":
			// It ends at the next line as far in as the sections.
			ts.Definition = p.parseBlockUntil(func() bool {
				return p.atSection(indent) || p.curTokenIs(token.EOF) || p.prevToken.Line != p.curToken.Line && p.column(p.curToken) <= indent
			})
			spec = p.learnTheory(ts)
		case "theorem":
			p.nextToken()
			start := p.curToken.Pos
			expr := p.parseExpression(LOWEST)
			ts.Theorems = append(ts.Theorems, &ast.Theorem{Line: sec.Line, Expr: expr, Text: strings.TrimSpace(p.l.Source()[start:p.curToken.End])})
			p.nextToken()
		case "proof":
			if spec == nil {
				p.errorAt(sec.Line, sec.Pos, "theory %s: its notation and definition come before its proof", name)
				p.skipToSection(indent)
				continue
			}
			p.nextToken()
			for !p.atSection(indent) && !p.curTokenIs(token.EOF) {
				if c := p.parseProofCase(spec); c != nil {
					ts.Proofs = append(ts.Proofs, c)
				}
			}
		}
	}
	for _, need := range []string{"abstract", "notation", "definition"} {
		if !seen[need] {
			p.errorAt(tok.Line, tok.Pos, "theory %s needs %s %s", name, article(need), need)
		}
	}
	p.nextToken() // theory -> [
	p.nextToken() // [ -> end
	p.nextToken() // end -> ]
	p.nextToken()
	return ts
}

func article(w string) string {
	if strings.ContainsRune("aeiou", rune(w[0])) {
		return "an"
	}
	return "a"
}

// skipToTheoryEnd moves past a theory that can't be read, to its
// theory [end].
func (p *Parser) skipToTheoryEnd() {
	for !p.atTheoryEnd() && !p.curTokenIs(token.EOF) {
		p.nextToken()
	}
	for i := 0; i < 4 && !p.curTokenIs(token.EOF); i++ {
		p.nextToken()
	}
}

func (p *Parser) skipToSection(indent int) {
	for !p.atSection(indent) && !p.curTokenIs(token.EOF) {
		p.nextToken()
	}
}

// parseAbstract takes the lines indented under "abstract" (or the rest of
// its own line) as they are written: free text, not code.
func (p *Parser) parseAbstract(sec token.Token, indent int) string {
	lines := strings.Split(p.l.Source(), "\n")
	var text []string
	if rest := strings.TrimSpace(lines[sec.Line-1][p.column(sec)+len("abstract"):]); rest != "" {
		text = append(text, rest)
	}
	n := sec.Line + 1 // the next line, from 1
	for ; n <= len(lines); n++ {
		l := strings.TrimRight(lines[n-1], "\r")
		t := strings.TrimSpace(l)
		if t != "" && len(l)-len(strings.TrimLeft(l, " \t")) <= indent {
			break
		}
		text = append(text, t)
	}
	for len(text) > 0 && text[len(text)-1] == "" {
		text = text[:len(text)-1]
	}
	if len(text) == 0 {
		p.errorAt(sec.Line, sec.Pos, "the abstract is empty: say in a line or two what the word means")
	}
	p.reseek(n)
	return strings.Join(text, "\n")
}

// parseNotation reads "notation tally b in a ." Which words are slots is
// settled once the definition is read (learnTheory).
func (p *Parser) parseNotation(ts *ast.TheoryStatement) *ast.Notation {
	sec := p.curToken
	n := &ast.Notation{Line: sec.Line}
	p.nextToken()
	for p.curToken.Line == sec.Line && !p.curTokenIs(token.PERIOD) && !p.curTokenIs(token.EOF) {
		t := p.curToken
		switch {
		case t.Type == token.COMMA:
			n.Parts = append(n.Parts, ast.NotationPart{Word: ","})
		case t.Type == token.IDENT || token.LookupIdent(t.Literal) != token.IDENT && isWord(t.Literal):
			switch t.Literal {
			case "at", "of", "is", "give":
				p.errorAt(t.Line, t.Pos, "a notation can't use %q: it already joins values in Turtle (x %s ...)", t.Literal, t.Literal)
			}
			n.Parts = append(n.Parts, ast.NotationPart{Word: t.Literal})
		default:
			p.errorAt(t.Line, t.Pos, "a notation is made of words, and commas between values: found %s", describe(t))
		}
		p.nextToken()
	}
	if !p.curTokenIs(token.PERIOD) || p.curToken.Line != sec.Line {
		p.errorAt(sec.Line, sec.Pos, "a notation ends with a period: notation %s b in a .", ts.Name)
		return nil
	}
	p.nextToken()
	if len(n.Parts) == 0 || n.Parts[0].Word != ts.Name {
		p.errorAt(sec.Line, sec.Pos, "%s's notation starts with its word: notation %s ... .", ts.Name, ts.Name)
		return nil
	}
	return n
}

func isWord(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_') {
			return false
		}
	}
	return s != ""
}

// learnTheory settles each notation's slots (the names the definition
// uses), checks them, and from now on reads the theory's phrases.
func (p *Parser) learnTheory(ts *ast.TheoryStatement) *theorySpec {
	if len(ts.Notations) == 0 || ts.Definition == nil {
		return nil
	}
	used := namesUsed(ts.Definition)
	spec := &theorySpec{name: ts.Name, line: ts.Token.Line, notations: ts.Notations, fixed: map[string]bool{}}
	for _, n := range ts.Notations {
		prevSlot := ""
		for i := range n.Parts {
			part := &n.Parts[i]
			part.Slot = i > 0 && part.Word != "," && used[part.Word]
			if !part.Slot {
				if i > 0 {
					spec.fixed[part.Word] = true
				}
				prevSlot = ""
				continue
			}
			if prevSlot != "" {
				p.errorAt(n.Line, 0, "%s: two values (%s, %s) need a word or a comma between them", n.Text(), prevSlot, part.Word)
			}
			prevSlot = part.Word
			if !containsString(spec.slots, part.Word) {
				spec.slots = append(spec.slots, part.Word)
			}
		}
	}
	ts.Slots = spec.slots
	if p.theories == nil {
		p.theories = map[string]*theorySpec{}
	}
	p.theories[ts.Name] = spec
	return spec
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// namesUsed is every name the code reads.
func namesUsed(node any) map[string]bool {
	out := map[string]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			if v.IsNil() {
				return
			}
			if id, ok := v.Interface().(*ast.Identifier); ok {
				out[id.Value] = true
				if id.Module != "" {
					out[id.Module] = true
				}
				return
			}
			walk(v.Elem())
		case reflect.Struct:
			if v.Type() == reflect.TypeOf(token.Token{}) {
				return
			}
			for i := range v.NumField() {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice, reflect.Array:
			for i := range v.Len() {
				walk(v.Index(i))
			}
		case reflect.String:
			// a sentence call's subject and names written as text in the
			// tree (a for-each's list name, ...) aren't values read here
		}
	}
	walk(reflect.ValueOf(node))
	// Names a call reads: a call's own name isn't a value, its subject is.
	return out
}

// startsPhrase: curToken is a theory's word, defined above, with its
// phrase after it on the line. The word alone is the theory itself, a
// value (typeof[tally], help[tally], diagnose[tally]).
func (p *Parser) startsPhrase() bool {
	if p.curToken.Type != token.IDENT {
		return false
	}
	spec, ok := p.theories[p.curToken.Literal]
	if !ok || p.peekToken.Line != p.curToken.Line || p.typeCheckAhead() {
		return false
	}
	return spec.fixed[p.peekToken.Literal] || p.argumentStartsAt(1)
}

// usedBeforeTheory reports a theory's word used above its theory, or one
// its file's import list leaves out.
func (p *Parser) usedBeforeTheory() bool {
	if path, ok := p.unlisted[p.curToken.Literal]; ok && p.theories[p.curToken.Literal] == nil && p.peekToken.Line == p.curToken.Line && p.argumentStartsAt(1) {
		p.errorAt(p.curToken.Line, p.curToken.Pos, "%q isn't imported: add it to \"import %s [...]\"", p.curToken.Literal, path)
		return true
	}
	line, later := p.laterTheories[p.curToken.Literal]
	if !later || line <= p.curToken.Line || p.theories[p.curToken.Literal] != nil {
		return false
	}
	p.errorAt(p.curToken.Line, p.curToken.Pos, "%s is defined by a theory on line %d, below this line: a theory comes before its word is used", p.curToken.Literal, line)
	return true
}

// parseTheoryCall reads a phrase: the word, then its fixed words and the
// values between them, matched against the theory's notations.
func (p *Parser) parseTheoryCall() *ast.TheoryCall {
	tok := p.curToken
	spec := p.theories[tok.Literal]
	var stops []string
	for w := range spec.fixed {
		stops = append(stops, w)
	}
	p.pushStops(stops...)
	defer p.popStops(stops...)
	defer func(was bool) { p.inBrackets = was }(p.inBrackets)
	p.inBrackets = false

	// The phrase as read: each fixed word, and each value in its place.
	var shape []string
	var values []ast.Expression
	for p.peekToken.Line == tok.Line {
		next := p.peekToken
		if spec.fixed[next.Literal] && (next.Type == token.IDENT || next.Type == token.COMMA || token.LookupIdent(next.Literal) != token.IDENT) && (next.Type != token.COMMA || spec.fixed[","]) {
			shape = append(shape, next.Literal)
			p.nextToken()
			continue
		}
		if len(shape) > 0 && shape[len(shape)-1] == "\x00" {
			// A value right after a value: the phrase is over, unless a
			// word follows, which no notation has there.
			if next.Type == token.IDENT {
				shape = append(shape, next.Literal)
			}
			break
		}
		if !p.argumentStartsAt(1) && !p.peekTokenIs(token.LBRACKET) {
			break
		}
		p.nextToken()
		values = append(values, p.parseExpression(SENTENCEARG))
		shape = append(shape, "\x00")
	}
	for _, n := range spec.notations {
		if !matches(n, shape) {
			continue
		}
		call := &ast.TheoryCall{Token: tok, Word: tok.Literal, Args: make([]ast.Expression, len(spec.slots))}
		k := 0
		for _, part := range n.Parts[1:] {
			if part.Slot {
				for i, s := range spec.slots {
					if s == part.Word {
						call.Args[i] = values[k]
					}
				}
				k++
			}
		}
		return call
	}
	var forms []string
	for _, n := range spec.notations {
		forms = append(forms, n.Text())
	}
	p.errorAt(tok.Line, tok.Pos, "this isn't how %s is written: %s", tok.Literal, strings.Join(forms, " or "))
	return &ast.TheoryCall{Token: tok, Word: tok.Literal, Args: make([]ast.Expression, len(spec.slots))}
}

// matches reports whether a phrase's shape (fixed words, and "\x00" for
// each value) is notation n's.
func matches(n *ast.Notation, shape []string) bool {
	if len(n.Parts)-1 != len(shape) {
		return false
	}
	for i, part := range n.Parts[1:] {
		if part.Slot != (shape[i] == "\x00") || !part.Slot && part.Word != shape[i] {
			return false
		}
	}
	return true
}

// parseProofCase reads one proof line: "tally 1 in list [1, 1, 0] . is 2".
func (p *Parser) parseProofCase(spec *theorySpec) *ast.ProofCase {
	tok := p.curToken
	if tok.Literal != spec.name {
		p.errorAt(tok.Line, tok.Pos, "a proof line is a use of %s and what it gives: %s ... . is ...", spec.name, spec.name)
		p.skipLine()
		return nil
	}
	use := p.parseTheoryCall()
	p.nextToken()
	if p.curTokenIs(token.PERIOD) && p.curToken.Line == tok.Line {
		p.nextToken()
	}
	if !p.curTokenIs(token.IS) || p.curToken.Line != tok.Line {
		p.errorAt(tok.Line, tok.Pos, "a proof line ends with is and what it gives: %s ... . is 2", spec.name)
		p.skipLine()
		return nil
	}
	p.nextToken()
	want := p.parseExpression(LOWEST)
	text := strings.TrimSpace(p.l.Source()[tok.Pos:p.curToken.End])
	p.nextToken()
	return &ast.ProofCase{Line: tok.Line, Text: text, Use: use, Want: want}
}

// importTheories learns the theories of the file "import path" brings in
// (the ones it defines, not its own imports'), so their phrases can be
// read from here on. A file that can't be found or read is the import's
// own error, when the program runs.
func (p *Parser) importTheories(tok token.Token, path string, names []string) {
	if p.ModuleDir == "" || p.importing[path] {
		return
	}
	full := filepath.Join(p.ModuleDir, filepath.FromSlash(path))
	ext, err := syntax.FindModule(full, func(ext string) bool {
		info, err := os.Stat(full + ext)
		return err == nil && !info.IsDir()
	})
	if err != nil || ext == "" {
		return
	}
	data, err := os.ReadFile(full + ext)
	if err != nil {
		return
	}
	sub := New(lexer.New(strings.ReplaceAll(string(data), "\r\n", "\n")))
	sub.ModuleDir = p.ModuleDir
	sub.importing = map[string]bool{path: true}
	for k := range p.importing {
		sub.importing[k] = true
	}
	sub.ParseProgram()
	for word, spec := range sub.theories {
		if spec.from != "" {
			continue
		}
		if names != nil && !containsString(names, word) {
			if p.unlisted == nil {
				p.unlisted = map[string]string{}
			}
			p.unlisted[word] = path
			continue
		}
		if p.theories == nil {
			p.theories = map[string]*theorySpec{}
		}
		if other, ok := p.theories[word]; ok {
			where := "this file"
			if other.from != "" {
				where = other.from
			}
			p.errorAt(tok.Line, tok.Pos, "import %s: %s is a theory in %s too: a word has one meaning", path, word, where)
			continue
		}
		mine := *spec
		mine.from = path
		p.theories[word] = &mine
	}
}
