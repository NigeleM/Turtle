// Package ast defines the Turtle abstract syntax tree.
package ast

import "Turtle/token"

type Node interface {
	TokenLiteral() string
}

type Statement interface {
	Node
	statementNode()
	// Line returns the source line this statement starts on, so the
	// evaluator can attach a real location to runtime errors instead of
	// reporting none at all.
	Line() int
}

type Expression interface {
	Node
	expressionNode()
}

// ---- Program -------------------------------------------------------

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// ---- Statements ------------------------------------------------------

type BlockStatement struct {
	Token      token.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) Line() int            { return bs.Token.Line }

type AssignStatement struct {
	Token token.Token
	Name  string
	Value Expression
}

func (as *AssignStatement) statementNode()       {}
func (as *AssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignStatement) Line() int            { return as.Token.Line }

type InputStatement struct {
	Token  token.Token
	Name   string
	Prompt string
}

func (is *InputStatement) statementNode()       {}
func (is *InputStatement) TokenLiteral() string { return is.Token.Literal }
func (is *InputStatement) Line() int            { return is.Token.Line }

type ShowStatement struct {
	Token       token.Token
	Expressions []Expression
}

func (ss *ShowStatement) statementNode()       {}
func (ss *ShowStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *ShowStatement) Line() int            { return ss.Token.Line }

// ReturnStatement's Value is nil for a bare "return", which yields none.
type ReturnStatement struct {
	Token token.Token
	Value Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) Line() int            { return rs.Token.Line }

type BreakStatement struct{ Token token.Token }

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) Line() int            { return bs.Token.Line }

type ContinueStatement struct{ Token token.Token }

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) Line() int            { return cs.Token.Line }

type FunctionDefStatement struct {
	Token      token.Token
	Name       string
	Parameters []string
	Body       *BlockStatement
}

func (fd *FunctionDefStatement) statementNode()       {}
func (fd *FunctionDefStatement) TokenLiteral() string { return fd.Token.Literal }
func (fd *FunctionDefStatement) Line() int            { return fd.Token.Line }

type CallStatement struct {
	Token token.Token
	Call  *CallExpression
}

func (cs *CallStatement) statementNode()       {}
func (cs *CallStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *CallStatement) Line() int            { return cs.Token.Line }

// ExpressionStatement is a bare expression used as a statement. Print is
// true for the "min of x ." / "max of x ." / "length of x ." forms,
// which print their result as a side effect rather than discarding it.
type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
	Print      bool
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) Line() int            { return es.Token.Line }

type IfClause struct {
	Condition Expression // nil for the trailing "else" clause
	Body      *BlockStatement
}

type IfStatement struct {
	Token   token.Token
	Clauses []*IfClause
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) Line() int            { return is.Token.Line }

type LoopKind int

const (
	LoopCStyle LoopKind = iota
	LoopWhile
	LoopEach
)

type LoopStatement struct {
	Token     token.Token
	Kind      LoopKind
	Init      Statement  // LoopCStyle only, may be nil
	Condition Expression // LoopCStyle and LoopWhile
	Post      Statement  // LoopCStyle only, may be nil
	Vars      []string   // LoopEach only: "x" or "k, v" in "[loop][k, v in m]"
	Iterable  Expression // LoopEach only
	Body      *BlockStatement
}

func (ls *LoopStatement) statementNode()       {}
func (ls *LoopStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LoopStatement) Line() int            { return ls.Token.Line }

type OpKind int

const (
	OpAdd OpKind = iota
	OpRemove
	OpDelete
	OpSort
	OpReverse
	OpInsert
)

// DataOpStatement covers the data-structure "sentence" operations:
// add/remove/delete/sort/reverse/insert.
type DataOpStatement struct {
	Token  token.Token
	Kind   OpKind
	Target string
	Value  Expression // nil for Sort/Reverse
	Index  Expression // OpInsert only
}

func (dop *DataOpStatement) statementNode()       {}
func (dop *DataOpStatement) TokenLiteral() string { return dop.Token.Literal }
func (dop *DataOpStatement) Line() int            { return dop.Token.Line }

// ImportStatement is "import m" (Names nil: the whole module) or
// "import m [a, b]" (only those names).
type ImportStatement struct {
	Token token.Token
	Path  string
	Names []string
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) Line() int            { return is.Token.Line }

type SysStatement struct {
	Token   token.Token
	Command string
}

func (ss *SysStatement) statementNode()       {}
func (ss *SysStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SysStatement) Line() int            { return ss.Token.Line }

type FileReadStatement struct {
	Token token.Token
	File  Expression
	Var   string
}

func (fr *FileReadStatement) statementNode()       {}
func (fr *FileReadStatement) TokenLiteral() string { return fr.Token.Literal }
func (fr *FileReadStatement) Line() int            { return fr.Token.Line }

// ContentItem is one line of a [write]/[append] body: either a literal
// string (Literal set, IsVar false) or a bare variable reference
// (Name set, IsVar true) resolved at run time.
type ContentItem struct {
	Literal string
	Name    string
	IsVar   bool
}

type FileWriteStatement struct {
	Token   token.Token
	File    Expression
	Content []ContentItem
	Append  bool
}

func (fw *FileWriteStatement) statementNode()       {}
func (fw *FileWriteStatement) TokenLiteral() string { return fw.Token.Literal }
func (fw *FileWriteStatement) Line() int            { return fw.Token.Line }

type DirectoryStatement struct {
	Token token.Token
	Path  Expression
	Var   string
}

func (ds *DirectoryStatement) statementNode()       {}
func (ds *DirectoryStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DirectoryStatement) Line() int            { return ds.Token.Line }

// ---- Expressions -----------------------------------------------------

// Identifier is a name, optionally qualified by the module it comes from
// ("time now", Module "time") to resolve a clash between imports.
type Identifier struct {
	Token  token.Token
	Module string
	Value  string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }

type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }

type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }

// FunctionLiteral is an anonymous function: "x gives x + 1",
// "[a, b] gives a + b", or the block form "[x] gives" ... "gives [end]".
// An expression body is stored as a one-statement block returning it.
type FunctionLiteral struct {
	Token      token.Token
	Parameters []string
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }

type NoneLiteral struct{ Token token.Token }

func (nl *NoneLiteral) expressionNode()      {}
func (nl *NoneLiteral) TokenLiteral() string { return nl.Token.Literal }

type PrefixExpression struct {
	Token    token.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }

type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }

// CallExpression is "name[args]", or, when Module is set, either
// "module name[args]" (a function from an imported module) or the
// sentence-style "subject verb args" (verb[subject, args...]). Which one
// is decided at run time: Module names an imported module, or a variable.
type CallExpression struct {
	Token     token.Token
	Module    string
	Name      string
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

type ListLiteral struct {
	Token    token.Token
	Elements []Expression
}

func (ll *ListLiteral) expressionNode()      {}
func (ll *ListLiteral) TokenLiteral() string { return ll.Token.Literal }

type SetLiteral struct {
	Token    token.Token
	Elements []Expression
}

func (sl *SetLiteral) expressionNode()      {}
func (sl *SetLiteral) TokenLiteral() string { return sl.Token.Literal }

type MapLiteral struct {
	Token  token.Token
	Keys   []Expression
	Values []Expression
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }

type MethodCallExpression struct {
	Token     token.Token
	Receiver  Expression
	Method    string
	Arguments []Expression
	Bracketed bool // args were given as "x at m[args]" (expression form)
}

func (mc *MethodCallExpression) expressionNode()      {}
func (mc *MethodCallExpression) TokenLiteral() string { return mc.Token.Literal }

type MinExpression struct {
	Token token.Token
	Arg   Expression
}

func (m *MinExpression) expressionNode()      {}
func (m *MinExpression) TokenLiteral() string { return m.Token.Literal }

type MaxExpression struct {
	Token token.Token
	Arg   Expression
}

func (m *MaxExpression) expressionNode()      {}
func (m *MaxExpression) TokenLiteral() string { return m.Token.Literal }

type LengthExpression struct {
	Token token.Token
	Arg   Expression
}

func (l *LengthExpression) expressionNode()      {}
func (l *LengthExpression) TokenLiteral() string { return l.Token.Literal }

// ChangeExpression converts Source's value to TypeName ("integer", "float",
// "string", "ascii", "char", or "hex"). As a standalone statement
// ("change a to integer .") the parser lowers it into an AssignStatement
// that writes the result back into Source's own identifier, mutating it in
// place; used anywhere else (assigned to a new variable, nested in a larger
// expression) it just produces the converted value.
type ChangeExpression struct {
	Token    token.Token
	Source   Expression
	TypeName string
}

func (c *ChangeExpression) expressionNode()      {}
func (c *ChangeExpression) TokenLiteral() string { return c.Token.Literal }

// AssembleStatement is "assemble Order [item, qty, price]".
type AssembleStatement struct {
	Token  token.Token
	Name   string
	Fields []string
}

func (as *AssembleStatement) statementNode()       {}
func (as *AssembleStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssembleStatement) Line() int            { return as.Token.Line }

// FieldExpression is "qty of o": one field of an assembled value.
type FieldExpression struct {
	Token  token.Token
	Field  string
	Object Expression
}

func (fe *FieldExpression) expressionNode()      {}
func (fe *FieldExpression) TokenLiteral() string { return fe.Token.Literal }

// FieldAssignStatement is "qty of o = 5".
type FieldAssignStatement struct {
	Token  token.Token
	Target *FieldExpression
	Value  Expression
}

func (fa *FieldAssignStatement) statementNode()       {}
func (fa *FieldAssignStatement) TokenLiteral() string { return fa.Token.Literal }
func (fa *FieldAssignStatement) Line() int            { return fa.Token.Line }
