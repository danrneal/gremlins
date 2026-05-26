/*
 * Copyright 2022 The Gremlins Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package engine_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-gremlins/gremlins/internal/engine"
	"github.com/go-gremlins/gremlins/internal/mutator"
)

func TestLiteralMutator_ApplyAndRollback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		origCode   string
		targetNode string
		wantCode   string
	}{
		{
			name: "true to false",
			origCode: `package main
func test() bool {
	return true
}
`,
			targetNode: "true",
			wantCode:   "package main\n\nfunc test() bool {\n\treturn false\n}\n",
		},
		{
			name: "false to true",
			origCode: `package main
func test() bool {
	return false
}
`,
			targetNode: "false",
			wantCode:   "package main\n\nfunc test() bool {\n\treturn true\n}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			fileName := "test_literal.go"
			filePath := filepath.Join(tmpDir, fileName)

			err := os.WriteFile(filePath, []byte(tc.origCode), 0o600)
			if err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, fileName, tc.origCode, 0)
			if err != nil {
				t.Fatalf("failed to parse file: %v", err)
			}

			var targetNode ast.Node
			ast.Inspect(f, func(node ast.Node) bool {
				if ident, ok := node.(*ast.Ident); ok && ident.Name == tc.targetNode {
					targetNode = ident

					return false
				}

				return true
			})

			if targetNode == nil {
				t.Fatalf("failed to find %q ident in AST", tc.targetNode)
			}

			m := engine.NewLiteralMutant("main", fset, f, targetNode)
			m.SetWorkdir(tmpDir)

			err = m.Apply()
			if err != nil {
				t.Fatalf("failed to apply mutation: %v", err)
			}

			mutatedContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read mutated file: %v", err)
			}

			if string(mutatedContent) != tc.wantCode {
				t.Errorf("unexpected mutated content:\nwant:\n%q\ngot:\n%q", tc.wantCode, string(mutatedContent))
			}

			err = m.Rollback()
			if err != nil {
				t.Fatalf("failed to rollback mutation: %v", err)
			}

			rolledBackContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read rolled back file: %v", err)
			}

			if string(rolledBackContent) != tc.origCode {
				t.Errorf("unexpected rolled back content: %q", string(rolledBackContent))
			}
		})
	}
}

func TestLiteralMutator_Getters(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fileName := "test_literal.go"
	filePath := filepath.Join(tmpDir, fileName)

	origCode := `package main
func test() bool {
	return true
}
`
	err := os.WriteFile(filePath, []byte(origCode), 0o600)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, origCode, 0)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	var targetNode ast.Node
	ast.Inspect(f, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name == "true" {
			targetNode = ident

			return false
		}

		return true
	})

	m := engine.NewLiteralMutant("main", fset, f, targetNode)
	m.SetWorkdir(tmpDir)
	m.SetType(mutator.InvertBooleanLiterals)
	m.SetStatus(mutator.Runnable)

	if m.Type() != mutator.InvertBooleanLiterals {
		t.Errorf("expected type %v, got %v", mutator.InvertBooleanLiterals, m.Type())
	}

	if m.Status() != mutator.Runnable {
		t.Errorf("expected status %v, got %v", mutator.Runnable, m.Status())
	}

	if m.Pos() != targetNode.Pos() {
		t.Errorf("expected pos %v, got %v", targetNode.Pos(), m.Pos())
	}

	if m.Pkg() != "main" {
		t.Errorf("expected pkg main, got %v", m.Pkg())
	}

	if m.Workdir() != tmpDir {
		t.Errorf("expected workdir %v, got %v", tmpDir, m.Workdir())
	}

	err = m.Apply()
	if err != nil {
		t.Fatalf("failed to apply: %v", err)
	}

	if len(m.OrigSnippet()) == 0 {
		t.Error("expected non-empty orig snippet")
	}

	if len(m.MutatedSnippet()) == 0 {
		t.Error("expected non-empty mutated snippet")
	}
}
