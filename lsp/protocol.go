package lsp

import "encoding/json"

// The parts of the Language Server Protocol this server uses: JSON-RPC 2.0
// messages, each sent with a Content-Length header, over stdin and stdout.
// https://microsoft.github.io/language-server-protocol/

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errNotInitialized = -32002
	errInvalidRequest = -32600
)

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type rangeLSP struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type location struct {
	URI   string   `json:"uri"`
	Range rangeLSP `json:"range"`
}

type textDocumentID struct {
	URI string `json:"uri"`
}

type positionParams struct {
	TextDocument textDocumentID `json:"textDocument"`
	Position     position       `json:"position"`
}

type diagnostic struct {
	Range    rangeLSP `json:"range"`
	Severity int      `json:"severity"` // 1 error, 2 warning
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

const (
	severityError   = 1
	severityWarning = 2
)

type completionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	SortText      string `json:"sortText,omitempty"`
}

// Completion item and symbol kinds.
const (
	completionMethod   = 2
	completionFunction = 3
	completionVariable = 6
	completionClass    = 7
	completionModule   = 9
	completionKeyword  = 14
	completionFile     = 17

	symbolFunction = 12
	symbolVariable = 13
	symbolStruct   = 23
)

type documentSymbol struct {
	Name           string   `json:"name"`
	Detail         string   `json:"detail,omitempty"`
	Kind           int      `json:"kind"`
	Range          rangeLSP `json:"range"`
	SelectionRange rangeLSP `json:"selectionRange"`
}

type hover struct {
	Contents markup `json:"contents"`
}

type markup struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// The colors (semantic tokens) the server sends, by index.
var tokenTypes = []string{"keyword", "string", "number", "comment", "function", "type"}
var tokenModifiers = []string{"declaration", "defaultLibrary"}
