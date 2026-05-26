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

package engine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"

	"github.com/go-gremlins/gremlins/internal/mutator"
)

const (
	trueStr      = "true"
	falseStr     = "false"
	contextLines = 3
)

// LiteralMutator is a mutator.Mutator for [ast.Ident] (booleans).
type LiteralMutator struct {
	node           ast.Node
	fs             *token.FileSet
	file           *ast.File
	pkg            string
	workDir        string
	origFile       []byte
	origSnippet    []byte
	mutatedSnippet []byte
	status         mutator.Status
	mutantType     mutator.Type
}

// NewLiteralMutant initializes a LiteralMutator.
func NewLiteralMutant(pkg string, set *token.FileSet, file *ast.File, node ast.Node) *LiteralMutator {
	return &LiteralMutator{
		pkg:  pkg,
		fs:   set,
		file: file,
		node: node,
	}
}

// Type returns the mutator.Type of the mutant.Mutator.
func (m *LiteralMutator) Type() mutator.Type {
	return m.mutantType
}

// SetType sets the mutator.Type of the mutant.Mutator.
func (m *LiteralMutator) SetType(mt mutator.Type) {
	m.mutantType = mt
}

// Status returns the mutator.Status of the mutant.Mutator.
func (m *LiteralMutator) Status() mutator.Status {
	return m.status
}

// SetStatus sets the mutator.Status of the mutant.Mutator.
func (m *LiteralMutator) SetStatus(s mutator.Status) {
	m.status = s
}

// Position returns the [token.Position] where the LiteralMutator resides.
func (m *LiteralMutator) Position() token.Position {
	return m.fs.Position(m.node.Pos())
}

// Pos returns the [token.Pos] where the LiteralMutator resides.
func (m *LiteralMutator) Pos() token.Pos {
	return m.node.Pos()
}

// Pkg returns the package name to which the mutant belongs.
func (m *LiteralMutator) Pkg() string {
	return m.pkg
}

// Apply mutates the literal node and writes to disk.
func (m *LiteralMutator) Apply() error {
	fileLock(m.Position().Filename).Lock()
	defer fileLock(m.Position().Filename).Unlock()

	filename := filepath.Join(m.workDir, m.Position().Filename)

	var err error

	m.origFile, err = os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return fmt.Errorf("read original file: %w", err)
	}

	m.origSnippet = extractSnippet(m.origFile, m.Position().Line, contextLines)

	var revert func()

	if n, ok := m.node.(*ast.Ident); ok {
		old := n.Name
		if n.Name == trueStr {
			n.Name = falseStr
		} else {
			n.Name = trueStr
		}

		revert = func() { n.Name = old }
	}

	if err := m.writeMutatedFile(filename); err != nil {
		return fmt.Errorf("write mutated file: %w", err)
	}

	// Rollback here to facilitate the atomicity of the operation.
	if revert != nil {
		revert()
	}

	return nil
}

func (m *LiteralMutator) writeMutatedFile(filename string) error {
	w := &bytes.Buffer{}

	err := printer.Fprint(w, m.fs, m.file)
	if err != nil {
		return fmt.Errorf("print mutated file: %w", err)
	}

	payload := w.Bytes()

	err = os.WriteFile(filename, payload, 0o600)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	m.mutatedSnippet = extractSnippet(payload, m.Position().Line, contextLines)

	return nil
}

// Rollback puts back the original file after the test and cleans up the LiteralMutator.
func (m *LiteralMutator) Rollback() error {
	defer m.resetOrigFile()

	filename := filepath.Join(m.workDir, m.Position().Filename)

	err := os.WriteFile(filename, m.origFile, 0o600)
	if err != nil {
		return fmt.Errorf("rollback file: %w", err)
	}

	return nil
}

// SetWorkdir sets the base path on which to Apply and Rollback operations.
func (m *LiteralMutator) SetWorkdir(path string) {
	m.workDir = path
}

// Workdir returns the current working dir in which the Mutator will apply its mutations.
func (m *LiteralMutator) Workdir() string {
	return m.workDir
}

func (m *LiteralMutator) resetOrigFile() {
	var zeroByte []byte

	m.origFile = zeroByte
}

// OrigSnippet returns the original code snippet around the mutation point.
func (m *LiteralMutator) OrigSnippet() []byte {
	return m.origSnippet
}

// MutatedSnippet returns the mutated code snippet around the mutation point.
func (m *LiteralMutator) MutatedSnippet() []byte {
	return m.mutatedSnippet
}
