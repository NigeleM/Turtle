// Package docs carries the reference into the turtle program, so help can
// answer from the same tables the reference shows.
package docs

import _ "embed"

// Reference is reference.md.
//
//go:embed reference.md
var Reference string
