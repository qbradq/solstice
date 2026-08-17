package solstice

import (
	"fmt"
	"image/color"
	"strings"

	"solstice/data"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
	"github.com/mitchellh/go-wordwrap"
)

const (
	replMaxCommandHistory = 100
	replMaxOutputHistory  = 1000
	replWrapWidth         = 80
)

var defaultTengoREPL *TengoREPL

// TengoREPLLine represents a single line in the REPL output log with color metadata.
type TengoREPLLine struct {
	Text  string
	Color color.Color
}

// GetTengoREPL returns the global TengoREPL instance.
func GetTengoREPL() *TengoREPL {
	return defaultTengoREPL
}

// SetTengoREPL sets the global TengoREPL instance.
func SetTengoREPL(r *TengoREPL) {
	defaultTengoREPL = r
}

// ResetTengoREPL resets the global TengoREPL instance if it exists.
func ResetTengoREPL() {
	if r := GetTengoREPL(); r != nil {
		r.Reset()
	}
}

// TengoREPL manages the Tengo REPL execution environment, persisting symbols and globals between statements,
// maintaining command history and output history.
type TengoREPL struct {
	symbolTable    *tengo.SymbolTable
	globals        []tengo.Object
	constants      []tengo.Object
	commandHistory []string
	outputHistory  []TengoREPLLine
}

// NewTengoREPL creates and initializes a new TengoREPL instance, running autoexec.tengo.
func NewTengoREPL() *TengoREPL {
	r := &TengoREPL{
		commandHistory: make([]string, 0, replMaxCommandHistory),
		outputHistory:  make([]TengoREPLLine, 0, replMaxOutputHistory),
	}
	r.Reset()
	SetTengoREPL(r)
	return r
}

// Reset clears the REPL globals and output history, resets the symbol table, and executes data/scripts/repl/autoexec.tengo.
// Command history is preserved across resets.
func (r *TengoREPL) Reset() {
	// Clear output history
	r.ClearOutputHistory()

	// Reset symbol table with Tengo builtins
	r.symbolTable = tengo.NewSymbolTable()
	for idx, fn := range tengo.GetAllBuiltinFunctions() {
		r.symbolTable.DefineBuiltin(idx, fn.Name)
	}

	// Reset globals and constants
	r.globals = make([]tengo.Object, tengo.GlobalsSize)
	r.constants = nil

	// Execute autoexec.tengo
	r.runAutoexec()
}

func (r *TengoREPL) runAutoexec() {
	autoexecPaths := []string{
		"data/scripts/repl/autoexec.tengo",
		"scripts/repl/autoexec.tengo",
		"repl/autoexec.tengo",
	}

	var src []byte
	var err error
	for _, p := range autoexecPaths {
		src, err = data.FS.ReadFile(p)
		if err == nil && len(src) > 0 {
			break
		}
	}

	if err != nil || len(src) == 0 {
		return
	}

	_ = r.executeSource("autoexec.tengo", src)
}

func (r *TengoREPL) executeSource(filename string, src []byte) error {
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile(filename, -1, len(src))

	p := parser.NewParser(srcFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	c := tengo.NewCompiler(srcFile, r.symbolTable, r.constants, GetScriptModuleMap(), nil)
	if err := c.Compile(file); err != nil {
		return fmt.Errorf("compile error: %w", err)
	}

	bytecode := c.Bytecode()
	vm := tengo.NewVM(bytecode, r.globals, -1)
	if err := vm.Run(); err != nil {
		return fmt.Errorf("runtime error: %w", err)
	}

	r.constants = bytecode.Constants
	return nil
}

// isIdentifier checks if a string is a valid single Tengo identifier.
func isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

// Execute executes a single statement or multi-statement Tengo code string within the persistent REPL environment.
// Special case: if the entire input line is just the name of a global variable, it outputs that global's string representation.
func (r *TengoREPL) Execute(statement string) error {
	trimmed := strings.TrimSpace(statement)
	if isIdentifier(trimmed) {
		if sym, _, ok := r.symbolTable.Resolve(trimmed, false); ok && sym != nil && sym.Scope == tengo.ScopeGlobal {
			if sym.Index >= 0 && sym.Index < len(r.globals) && r.globals[sym.Index] != nil {
				obj := r.globals[sym.Index]
				r.AddOutputColored(obj.String(), VGAPalette16[15])
				return nil
			}
		}
	}

	return r.executeSource("repl", []byte(statement))
}

// GetGlobal returns the value of a global variable by name, or nil if not found.
func (r *TengoREPL) GetGlobal(name string) tengo.Object {
	if sym, _, ok := r.symbolTable.Resolve(name, false); ok && sym != nil && sym.Scope == tengo.ScopeGlobal {
		if sym.Index >= 0 && sym.Index < len(r.globals) {
			return r.globals[sym.Index]
		}
	}
	return nil
}

// AddCommand adds a command line to the command history (up to 100 entries, FIFO).
func (r *TengoREPL) AddCommand(cmd string) {
	r.commandHistory = append(r.commandHistory, cmd)
	if len(r.commandHistory) > replMaxCommandHistory {
		r.commandHistory = r.commandHistory[len(r.commandHistory)-replMaxCommandHistory:]
	}
}

// GetCommandHistory returns a copy of the command history.
func (r *TengoREPL) GetCommandHistory() []string {
	res := make([]string, len(r.commandHistory))
	copy(res, r.commandHistory)
	return res
}

// AddOutput word-wraps text to 80 columns and appends each line in default white color (up to 1000 entries, FIFO).
func (r *TengoREPL) AddOutput(text string) {
	r.AddOutputColored(text, VGAPalette16[15])
}

// AddOutputColored word-wraps text to 80 columns and appends each line with the specified color (up to 1000 entries, FIFO).
func (r *TengoREPL) AddOutputColored(text string, c color.Color) {
	wrapped := wordwrap.WrapString(text, replWrapWidth)
	lines := strings.Split(wrapped, "\n")
	for _, l := range lines {
		r.outputHistory = append(r.outputHistory, TengoREPLLine{
			Text:  l,
			Color: c,
		})
	}

	if len(r.outputHistory) > replMaxOutputHistory {
		r.outputHistory = r.outputHistory[len(r.outputHistory)-replMaxOutputHistory:]
	}
}

// AddRawOutputColored appends a line without word-wrapping with the specified color (up to 1000 entries, FIFO).
func (r *TengoREPL) AddRawOutputColored(line string, c color.Color) {
	r.outputHistory = append(r.outputHistory, TengoREPLLine{
		Text:  line,
		Color: c,
	})

	if len(r.outputHistory) > replMaxOutputHistory {
		r.outputHistory = r.outputHistory[len(r.outputHistory)-replMaxOutputHistory:]
	}
}

// GetOutputHistory returns a copy of all lines currently in output history with color metadata.
func (r *TengoREPL) GetOutputHistory() []TengoREPLLine {
	res := make([]TengoREPLLine, len(r.outputHistory))
	copy(res, r.outputHistory)
	return res
}

// GetOutputTexts returns a copy of the raw text strings of all output history lines.
func (r *TengoREPL) GetOutputTexts() []string {
	res := make([]string, len(r.outputHistory))
	for i, l := range r.outputHistory {
		res[i] = l.Text
	}
	return res
}

// ClearOutputHistory empties the output line history.
func (r *TengoREPL) ClearOutputHistory() {
	r.outputHistory = r.outputHistory[:0]
}
