// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// runIntegration runs a testdata program (rel, from testdata/) that checks
// its own results, with stdin as its input, and fails unless it reports no
// failures.
func runIntegration(t *testing.T, rel, stdin string) {
	name := filepath.Base(rel)
	dir, err := filepath.Abs("../testdata")
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, name), src, 0o644); err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(src)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatal(errs)
	}
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	it := NewWithStdin(dir, strings.NewReader(stdin))
	it.WorkDir = work
	runErr := it.Run(program)
	w.Close()
	os.Stdout = orig
	var out strings.Builder
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		out.WriteString(sc.Text() + "\n")
	}
	if runErr != nil || !strings.Contains(out.String(), "failures: 0") {
		t.Fatalf("%s failed (err %v):\n%s", name, runErr, out.String())
	}
}

// testdata/theories_and_imports.turtle: theories in every form, import
// aliases, and what they touch (see the file).
func TestTheoriesAndImportsScript(t *testing.T) {
	runIntegration(t, "theories_and_imports.turtle", "")
}

// testdata/programs/: themed programs, each checking its own results (see
// the file's header for what it covers).
func TestProgramLanguageCore(t *testing.T) {
	runIntegration(t, "programs/language_core.turtle", "7\n")
}

func TestProgramControlAndFunctions(t *testing.T) {
	runIntegration(t, "programs/control_and_functions.turtle", "")
}

func TestProgramJobQueue(t *testing.T) {
	runIntegration(t, "programs/job_queue.turtle", "")
}

func TestProgramMultiline(t *testing.T) {
	runIntegration(t, "programs/multiline.turtle", "")
}

func TestProgramBankSimulation(t *testing.T) {
	runIntegration(t, "programs/bank_simulation.turtle", "")
}

func TestProgramCSVTransform(t *testing.T) {
	runIntegration(t, "programs/csv_transform.turtle", "")
}

func TestProgramTextReport(t *testing.T) {
	runIntegration(t, "programs/text_report.turtle", "")
}

func TestProgramScheduleTime(t *testing.T) {
	runIntegration(t, "programs/schedule_time.turtle", "")
}

func TestProgramPipelineRecovery(t *testing.T) {
	runIntegration(t, "programs/pipeline_recovery.turtle", "")
}

func TestProgramTheoryLibrary(t *testing.T) {
	runIntegration(t, "programs/theory_library.turtle", "")
}

func TestProgramCollections(t *testing.T) {
	runIntegration(t, "programs/collections.turtle", "")
}

func TestProgramAssembledTypes(t *testing.T) {
	runIntegration(t, "programs/assembled_types.turtle", "")
}

func TestProgramDataFormats(t *testing.T) {
	runIntegration(t, "programs/data_formats.turtle", "")
}

func TestProgramSQLInventory(t *testing.T) {
	runIntegration(t, "programs/sql_inventory.turtle", "")
}

func TestProgramMathRandomMatrix(t *testing.T) {
	runIntegration(t, "programs/math_random_matrix.turtle", "")
}

func TestProgramFilesAndFolders(t *testing.T) {
	runIntegration(t, "programs/files_and_folders.turtle", "")
}

func TestProgramImportSettings(t *testing.T) {
	runIntegration(t, "programs/import_settings.turtle", "")
}

func TestProgramShowArgumentTerminator(t *testing.T) {
	runIntegration(t, "programs/show_argument_terminator.turtle", "")
}
