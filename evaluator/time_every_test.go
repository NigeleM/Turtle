// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"strings"
	"testing"
	"time"
)

// every runs in milliseconds and nanoseconds as well as the calendar units.
func TestEveryRunsInMilliseconds(t *testing.T) {
	src := "import time\nn = list [0]\ndef job[]\n    m = n at get 0 + 1\n    put m to n at 0 .\n    if ] m >= 20 [\n        return false\n    if [end]\n    return true\ndef [end]\nevery[1, \"ms\", job]\nshow n at get 0 .\n"
	start := time.Now()
	got, err := run(t, src, "")
	if err != nil || strings.TrimSpace(got) != "20" {
		t.Fatalf("got %q, %v", got, err)
	}
	// 20 runs, one millisecond apart: about 19 ms, not seconds.
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("took %v for 20 ms steps", d)
	}
}

func TestEveryUnitsAndErrors(t *testing.T) {
	ok := "import time\ndef job[]\n    return false\ndef [end]\nevery[1, \"ns\", job]\nevery[1, \"ms\", job]\nshow \"ran\" .\n"
	if got, err := run(t, ok, ""); err != nil || strings.TrimSpace(got) != "ran" {
		t.Errorf("ns and ms: got %q, %v", got, err)
	}
	bad := "import time\ndef job[]\n    return false\ndef [end]\nevery[1, \"fortnights\", job]\n"
	if _, err := run(t, bad, ""); err == nil || !strings.Contains(err.Error(), "must be one of ns, ms, seconds") {
		t.Errorf("a bad unit: got %v", err)
	}
}
