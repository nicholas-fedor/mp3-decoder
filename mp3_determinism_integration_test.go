// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package mp3_test

import (
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/mp3-decoder/internal/conformance"
)

// goldenPath is the file that records the hash of each fixture's decoded samples.
const goldenPath = "testdata/golden.txt"

// update rewrites the recorded hashes instead of checking them.
var update = flag.Bool("update", false, "rewrite testdata/golden.txt from the decoded fixtures")

// TestProductsRounded checks that every floating-point product in the decoder's production code is
// converted to a floating-point type where it is made.
//
// Go lets a compiler fuse a product with a later sum into one multiply-add, which rounds once
// instead of twice. Only an explicit conversion to a floating-point type forbids it. arm64 fuses
// such pairs by default, and amd64 fuses some at GOAMD64=v3. An unconverted product can therefore
// decode different bits on different machines. A compound *= assignment is reported too, since
// it cannot convert.
func TestProductsRounded(t *testing.T) {
	t.Parallel()

	files := token.NewFileSet()
	sources, ok := importer.ForCompiler(files, "source", nil).(types.ImporterFrom)
	require.True(t, ok, "the source importer imports from a directory")

	for dir, parsed := range parsePackages(t, files) {
		info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
		config := types.Config{Importer: sources}

		_, err := config.Check(dir, files, parsed, info)
		require.NoError(t, err, dir)

		for _, file := range parsed {
			for _, at := range unrounded(info, file) {
				t.Errorf("%s: the floating-point product is not converted where it is made", files.Position(at))
			}
		}
	}
}

// TestGolden checks the hash of the samples each fixture decodes to against those golden.txt
// records, or with -update records them there.
//
// The hashes are the same on every architecture and at every GOAMD64 level, so any change to a
// single bit of the output fails the test.
func TestGolden(t *testing.T) {
	t.Parallel()

	var got strings.Builder

	for _, name := range []string{"mpeg1-joint.mp3", "mpeg1-mono.mp3", "mpeg2.mp3", "mpeg25.mp3", "protected.mp3"} {
		fmt.Fprintf(&got, "%s %x\n", name, conformance.Digest(decodeFile(t, name)))
	}

	if *update {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got.String()), 0o600))

		return
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err)

	assert.Equal(t, string(want), got.String(), "the decoded samples changed: rerun the test with -update")
}

// parsePackages parses the production files of every package in the decoder's tree, by directory.
func parsePackages(t *testing.T, files *token.FileSet) map[string][]*ast.File {
	t.Helper()

	packages := make(map[string][]*ast.File)

	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir() && entry.Name() == "testdata":
			return filepath.SkipDir
		case entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go"):
			return nil
		default:
			file, err := parser.ParseFile(files, path, nil, parser.SkipObjectResolution)
			packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], file)

			return err
		}
	})
	require.NoError(t, err)

	return packages
}

// unrounded returns where a file makes a floating-point product that no conversion to a
// floating-point type takes straight away, or a floating-point *= assignment.
func unrounded(info *types.Info, file *ast.File) []token.Pos {
	var found []token.Pos

	ast.PreorderStack(file, nil, func(node ast.Node, stack []ast.Node) bool {
		switch node := node.(type) {
		case *ast.BinaryExpr:
			if node.Op == token.MUL && floating(info, node) && !converted(info, stack) {
				found = append(found, node.OpPos)
			}
		case *ast.AssignStmt:
			if node.Tok == token.MUL_ASSIGN && floating(info, node.Lhs[0]) {
				found = append(found, node.TokPos)
			}
		default:
		}

		return true
	})

	return found
}

// floating reports whether an expression is a floating-point value computed at run time.
func floating(info *types.Info, expr ast.Expr) bool {
	value := info.Types[expr]
	basic, ok := value.Type.Underlying().(*types.Basic)

	return ok && value.Value == nil && basic.Info()&types.IsFloat != 0
}

// converted reports whether the innermost of an expression's ancestors, past any parentheses, is a
// conversion to a floating-point type.
func converted(info *types.Info, stack []ast.Node) bool {
	for _, s := range slices.Backward(stack) {
		switch parent := s.(type) {
		case *ast.ParenExpr:
			continue
		case *ast.CallExpr:
			return info.Types[parent.Fun].IsType() && floating(info, parent)
		default:
			return false
		}
	}

	return false
}
