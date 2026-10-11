// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

// Package docs carries the reference into the turtle program, so help can
// answer from the same tables the reference shows.
package docs

import _ "embed"

// Reference is reference.md.
//
//go:embed reference.md
var Reference string
