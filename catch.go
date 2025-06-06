package catch

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// Catch is a global variable that can be used to catch and handle errors
var Catch = ErrorCatcher{}

// ErrorConfig holds configuration for error handling
type ErrorConfig struct {
	ShowStackTrace      bool
	ShowSourceCode      bool
	ShowSuggestions     bool
	ExitOnError         bool
	LogToFile           string
	MaxStackDepth       int
	ContextLines        int
	UseColors           bool
	EnableSmartAnalysis bool // New: Toggle for source code analysis
	EnableStackAnalysis bool // New: Toggle for stack trace analysis
}

// DefaultConfig provides sensible defaults with Rust-like formatting
var DefaultConfig = ErrorConfig{
	ShowStackTrace:      true,
	ShowSourceCode:      true,
	ShowSuggestions:     true,
	ExitOnError:         true,
	MaxStackDepth:       10,
	ContextLines:        2,
	UseColors:           true,
	EnableSmartAnalysis: true,
	EnableStackAnalysis: true,
}

// ErrorCatcher is a type that can be used to catch and handle errors
type ErrorCatcher struct {
	Config ErrorConfig
}

// Enhanced error information
type ErrorInfo struct {
	Error       error
	File        string
	Line        int
	Column      int
	Function    string
	Context     map[string]interface{}
	Stack       []StackFrame
	SourceLines []SourceLine
	ErrorCode   string
	Suggestion  string
}

type StackFrame struct {
	File     string
	Line     int
	Function string
}

type SourceLine struct {
	Number  int
	Content string
	IsError bool
}

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	BrightRed = "\033[91m"
	Gray      = "\033[90m"
)

// Configure sets the error handling configuration
func (e *ErrorCatcher) Configure(config ErrorConfig) *ErrorCatcher {
	e.Config = config
	return e
}

// WithContext adds contextual information to error handling
func (e ErrorCatcher) WithContext(key string, value interface{}) *ContextualCatcher {
	return &ContextualCatcher{
		catcher: e,
		context: map[string]interface{}{key: value},
	}
}

// ContextualCatcher allows chaining context information
type ContextualCatcher struct {
	catcher ErrorCatcher
	context map[string]interface{}
}

// WithContext adds more context to the chain
func (c *ContextualCatcher) WithContext(key string, value interface{}) *ContextualCatcher {
	c.context[key] = value
	return c
}

// Set handles error with accumulated context
func (c *ContextualCatcher) Set(err error) error {
	if err != nil {
		info := c.catcher.buildErrorInfo(err, 1)
		info.Context = c.context
		c.catcher.handleError(info)
	}
	return err
}

// X is the main auto-detecting error handler
// Usage: except.X(err)
// Usage: except.X(err, filename)
// Usage: except.X(err, "processing", filename)
// Usage: except.X(err, map[string]interface{}{"file": filename, "op": "read"})
func Err(err error, context ...interface{}) error {
	if err == nil {
		return nil
	}

	// Build smart context
	info := buildSmartErrorInfo(err, context...)
	Catch.handleError(info)
	return err
}

// buildSmartErrorInfo creates comprehensive error info with auto-detection
func buildSmartErrorInfo(err error, context ...interface{}) ErrorInfo {
	config := Catch.getConfig()

	// Get caller information
	pc, file, line, ok := runtime.Caller(2) // Skip X() and buildSmartErrorInfo()
	if !ok {
		file = "unknown"
		line = 0
	}

	var funcName string
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = fn.Name()
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
			funcName = funcName[lastSlash+1:]
		}
	}

	info := ErrorInfo{
		Error:      err,
		File:       file,
		Line:       line,
		Function:   funcName,
		Context:    make(map[string]interface{}),
		ErrorCode:  generateSmartErrorCode(err),
		Suggestion: generateSmartSuggestion(err),
	}

	// Auto-detect and build context
	info.Context = buildSmartContext(file, line, context...)

	// Load source code context if enabled
	if config.ShowSourceCode {
		info.SourceLines = Catch.loadSourceContext(file, line, config.ContextLines)
	}

	// Build stack trace if enabled
	if config.ShowStackTrace {
		info.Stack = Catch.buildStackTrace(1)
	}

	return info
}

// buildSmartContext auto-detects context from various sources
func buildSmartContext(file string, line int, context ...interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})

	// 1. Parse provided context
	ctx = parseProvidedContext(ctx, context...)

	// 2. Auto-detect from source code
	if Catch.getConfig().EnableSmartAnalysis {
		sourceCtx := detectContextFromSource(file, line)
		for k, v := range sourceCtx {
			if _, exists := ctx[k]; !exists { // Don't override explicit context
				ctx[k] = v
			}
		}
	}

	// 3. Auto-detect from stack trace
	if Catch.getConfig().EnableStackAnalysis {
		stackCtx := detectContextFromStack()
		for k, v := range stackCtx {
			if _, exists := ctx[k]; !exists {
				ctx[k] = v
			}
		}
	}

	return ctx
}

// parseProvidedContext handles various context input formats
func parseProvidedContext(ctx map[string]interface{}, context ...interface{}) map[string]interface{} {
	if len(context) == 0 {
		return ctx
	}

	// Handle map[string]interface{} directly
	if len(context) == 1 {
		if m, ok := context[0].(map[string]interface{}); ok {
			for k, v := range m {
				ctx[k] = v
			}
			return ctx
		}
	}

	// Handle key-value pairs or auto-name values
	for i, item := range context {
		switch v := item.(type) {
		case string:
			if i+1 < len(context) && i%2 == 0 {
				// String followed by value = key-value pair
				ctx[v] = context[i+1]
				continue
			}
			// Standalone string = operation or description
			if strings.Contains(v, "/") || strings.Contains(v, ".") {
				ctx["path"] = v
			} else {
				ctx["operation"] = v
			}
		case int, int64, float64:
			ctx[fmt.Sprintf("value_%d", i)] = v
		default:
			// Try to infer meaning from type
			ctx[inferContextKey(v)] = v
		}
	}

	return ctx
}

// inferContextKey tries to guess appropriate key names
func inferContextKey(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "/") || strings.Contains(v, ".") {
			return "path"
		}
		if len(v) < 20 {
			return "name"
		}
		return "description"
	case *os.File:
		return "file"
	default:
		t := reflect.TypeOf(value)
		if t != nil {
			return strings.ToLower(t.Name())
		}
		return "context"
	}
}

// detectContextFromSource analyzes source code around error line
func detectContextFromSource(filename string, errorLine int) map[string]interface{} {
	ctx := make(map[string]interface{})

	// Wrap in defer to handle any panics from AST parsing
	defer func() {
		if r := recover(); r != nil {
			// Silently fall back to basic context if AST parsing fails
		}
	}()

	// Parse the source file
	fset := token.NewFileSet()
	src, err := os.ReadFile(filename)
	if err != nil {
		return ctx
	}

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return ctx
	}

	// Find variables and function calls near the error line
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pos := fset.Position(n.Pos())
		if abs(pos.Line-errorLine) <= 2 { // Within 2 lines of error
			switch node := n.(type) {
			case *ast.CallExpr:
				// Detect function calls like os.Open(filename)
				if ident, ok := node.Fun.(*ast.SelectorExpr); ok {
					if x, ok := ident.X.(*ast.Ident); ok {
						funcName := fmt.Sprintf("%s.%s", x.Name, ident.Sel.Name)
						ctx["function_call"] = funcName

						// Extract arguments
						for i, arg := range node.Args {
							if ident, ok := arg.(*ast.Ident); ok {
								ctx[fmt.Sprintf("arg_%d_%s", i, ident.Name)] = ident.Name

								// Common patterns
								if strings.Contains(ident.Name, "file") || strings.Contains(ident.Name, "path") {
									ctx["target_file"] = ident.Name
								}
							}
						}
					}
				}
			case *ast.AssignStmt:
				// Detect assignments like file, err := os.Open(...)
				for _, expr := range node.Lhs {
					if ident, ok := expr.(*ast.Ident); ok {
						if ident.Name != "err" {
							ctx["assigned_var"] = ident.Name
						}
					}
				}
			}
		}
		return true
	})

	return ctx
}

// detectContextFromStack analyzes stack frames for patterns
func detectContextFromStack() map[string]interface{} {
	ctx := make(map[string]interface{})

	// Look at calling functions
	for i := 3; i < 8; i++ { // Skip our internal calls
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName := fn.Name()

			// Clean function name
			if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
				funcName = funcName[lastSlash+1:]
			}

			// Detect patterns in function names
			lower := strings.ToLower(funcName)
			switch {
			case strings.Contains(lower, "read"):
				ctx["operation_type"] = "read"
			case strings.Contains(lower, "write"):
				ctx["operation_type"] = "write"
			case strings.Contains(lower, "open"):
				ctx["operation_type"] = "open"
			case strings.Contains(lower, "process"):
				ctx["operation_type"] = "process"
			case strings.Contains(lower, "handle"):
				ctx["operation_type"] = "handle"
			}

			// Add calling context
			ctx["caller_function"] = funcName
			ctx["caller_location"] = fmt.Sprintf("%s:%d", filepath.Base(file), line)
			break
		}
	}

	return ctx
}

// generateSmartErrorCode creates context-aware error codes
func generateSmartErrorCode(err error) string {
	errStr := strings.ToLower(err.Error())

	// File system errors
	switch {
	case strings.Contains(errStr, "no such file"):
		return "FS001"
	case strings.Contains(errStr, "permission denied"):
		return "FS002"
	case strings.Contains(errStr, "file exists"):
		return "FS003"
	case strings.Contains(errStr, "is a directory"):
		return "FS004"
	case strings.Contains(errStr, "not a directory"):
		return "FS005"

	// Network errors
	case strings.Contains(errStr, "connection refused"):
		return "NET001"
	case strings.Contains(errStr, "timeout"):
		return "NET002"
	case strings.Contains(errStr, "host not found"):
		return "NET003"
	case strings.Contains(errStr, "network unreachable"):
		return "NET004"

	// Data errors
	case strings.Contains(errStr, "parse"):
		return "DATA001"
	case strings.Contains(errStr, "invalid format"):
		return "DATA002"
	case strings.Contains(errStr, "decode"):
		return "DATA003"
	case strings.Contains(errStr, "encode"):
		return "DATA004"

	// Logic errors
	case strings.Contains(errStr, "index out of range"):
		return "LOGIC001"
	case strings.Contains(errStr, "nil pointer"):
		return "LOGIC002"
	case strings.Contains(errStr, "assertion failed"):
		return "LOGIC003"

	// Generic
	default:
		return "GEN000"
	}
}

// generateSmartSuggestion creates context-aware suggestions
func generateSmartSuggestion(err error) string {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "no such file"):
		return "verify the file path exists, check for typos, or create the file first"
	case strings.Contains(errStr, "permission denied"):
		return "run with appropriate permissions, check file ownership, or modify file permissions"
	case strings.Contains(errStr, "connection refused"):
		return "ensure the target service is running, check firewall settings, or verify the address and port"
	case strings.Contains(errStr, "timeout"):
		return "increase timeout duration, check network connectivity, or optimize the operation"
	case strings.Contains(errStr, "parse"):
		return "validate input format, check for encoding issues, or review the data structure"
	case strings.Contains(errStr, "index out of range"):
		return "add bounds checking, validate array/slice length, or review loop conditions"
	case strings.Contains(errStr, "nil pointer"):
		return "add nil checks, initialize variables properly, or review pointer assignments"
	default:
		return "check the error context, consult documentation, or add debug logging"
	}
}

// buildErrorInfo creates detailed error information with source code context
func (e ErrorCatcher) buildErrorInfo(err error, skip int) ErrorInfo {
	config := e.getConfig()

	// Get caller information
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		file = "unknown"
		line = 0
	}

	var funcName string
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = fn.Name()
		// Clean up function name (remove package path)
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
			funcName = funcName[lastSlash+1:]
		}
	}

	info := ErrorInfo{
		Error:      err,
		File:       file,
		Line:       line,
		Function:   funcName,
		Context:    make(map[string]interface{}),
		ErrorCode:  generateSmartErrorCode(err),
		Suggestion: generateSmartSuggestion(err),
	}

	// Load source code context if enabled
	if config.ShowSourceCode {
		info.SourceLines = e.loadSourceContext(file, line, config.ContextLines)
	}

	// Build stack trace if enabled
	if config.ShowStackTrace {
		info.Stack = e.buildStackTrace(skip + 1)
	}

	return info
}

// loadSourceContext reads source code around the error line
func (e ErrorCatcher) loadSourceContext(filename string, errorLine, contextLines int) []SourceLine {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []SourceLine
	scanner := bufio.NewScanner(file)
	currentLine := 0

	startLine := errorLine - contextLines
	endLine := errorLine + contextLines

	if startLine < 1 {
		startLine = 1
	}

	for scanner.Scan() {
		currentLine++
		if currentLine >= startLine && currentLine <= endLine {
			lines = append(lines, SourceLine{
				Number:  currentLine,
				Content: scanner.Text(),
				IsError: currentLine == errorLine,
			})
		}
		if currentLine > endLine {
			break
		}
	}

	return lines
}

// buildStackTrace creates a stack trace
func (e ErrorCatcher) buildStackTrace(skip int) []StackFrame {
	config := e.getConfig()
	var stack []StackFrame

	for i := skip + 1; i < skip+config.MaxStackDepth; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		var funcName string
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = fn.Name()
			if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
				funcName = funcName[lastSlash+1:]
			}
		}

		stack = append(stack, StackFrame{
			File:     file,
			Line:     line,
			Function: funcName,
		})
	}

	return stack
}

// handleError processes and outputs the error in Rust style
func (e ErrorCatcher) handleError(info ErrorInfo) {
	config := e.getConfig()

	var output strings.Builder

	// Rust-style error header
	if config.UseColors {
		output.WriteString(fmt.Sprintf("%serror[%s%s%s]: %s%s%s\n",
			BrightRed, Bold, info.ErrorCode, Reset+BrightRed, Bold, info.Error.Error(), Reset))
	} else {
		output.WriteString(fmt.Sprintf("error[%s]: %s\n", info.ErrorCode, info.Error.Error()))
	}

	// File location with arrow
	filename := filepath.Base(info.File)
	if config.UseColors {
		output.WriteString(fmt.Sprintf(" %s-->%s %s:%d\n", Blue+Bold, Reset, filename, info.Line))
	} else {
		output.WriteString(fmt.Sprintf(" --> %s:%d\n", filename, info.Line))
	}

	// Source code context
	if config.ShowSourceCode && len(info.SourceLines) > 0 {
		output.WriteString("  |\n")

		// Calculate padding for line numbers
		maxLineNum := info.SourceLines[len(info.SourceLines)-1].Number
		padding := len(fmt.Sprintf("%d", maxLineNum))

		for _, sourceLine := range info.SourceLines {
			lineNumStr := fmt.Sprintf("%*d", padding, sourceLine.Number)

			if sourceLine.IsError {
				if config.UseColors {
					output.WriteString(fmt.Sprintf("%s%s%s |%s %s\n",
						Red+Bold, lineNumStr, Reset, Reset, sourceLine.Content))
				} else {
					output.WriteString(fmt.Sprintf("%s | %s\n", lineNumStr, sourceLine.Content))
				}

				// Add error pointer
				spaces := strings.Repeat(" ", padding)
				if config.UseColors {
					output.WriteString(fmt.Sprintf("%s |%s %s%s^\n",
						spaces, Reset, Red+Bold, Reset))
				} else {
					output.WriteString(fmt.Sprintf("%s | ^\n", spaces))
				}
			} else {
				if config.UseColors {
					output.WriteString(fmt.Sprintf("%s%s%s |%s %s%s%s\n",
						Blue, lineNumStr, Reset, Reset, Gray, sourceLine.Content, Reset))
				} else {
					output.WriteString(fmt.Sprintf("%s | %s\n", lineNumStr, sourceLine.Content))
				}
			}
		}
		output.WriteString("  |\n")
	}

	// Add context if available
	if len(info.Context) > 0 {
		if config.UseColors {
			output.WriteString(fmt.Sprintf("  %s=%s %scontext:%s\n", Blue+Bold, Reset, Yellow+Bold, Reset))
		} else {
			output.WriteString("  = context:\n")
		}

		for k, v := range info.Context {
			if config.UseColors {
				output.WriteString(fmt.Sprintf("    %s%s%s: %v\n", Cyan, k, Reset, v))
			} else {
				output.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
			}
		}
		output.WriteString("\n")
	}

	// Add suggestion
	if config.ShowSuggestions && info.Suggestion != "" {
		if config.UseColors {
			output.WriteString(fmt.Sprintf("  %s=%s %shelp:%s %s\n",
				Blue+Bold, Reset, Green+Bold, Reset, info.Suggestion))
		} else {
			output.WriteString(fmt.Sprintf("  = help: %s\n", info.Suggestion))
		}
		output.WriteString("\n")
	}

	// Add stack trace if enabled
	if config.ShowStackTrace && len(info.Stack) > 0 {
		if config.UseColors {
			output.WriteString(fmt.Sprintf("  %s=%s %sstack backtrace:%s\n",
				Blue+Bold, Reset, Yellow+Bold, Reset))
		} else {
			output.WriteString("  = stack backtrace:\n")
		}

		for i, frame := range info.Stack {
			frameFile := filepath.Base(frame.File)
			if config.UseColors {
				output.WriteString(fmt.Sprintf("   %s%2d:%s %s%s%s\n          at %s%s:%d%s\n",
					Gray, i, Reset, Bold, frame.Function, Reset,
					Gray, frameFile, frame.Line, Reset))
			} else {
				output.WriteString(fmt.Sprintf("   %2d: %s\n          at %s:%d\n",
					i, frame.Function, frameFile, frame.Line))
			}
		}
		output.WriteString("\n")
	}

	message := output.String()

	// Output to stderr
	fmt.Fprint(os.Stderr, message)

	// Log to file if configured
	if config.LogToFile != "" {
		e.logToFile(config.LogToFile, message)
	}

	// Exit if configured
	if config.ExitOnError {
		os.Exit(1)
	}
}

// logToFile writes error to a log file (without colors)
func (e ErrorCatcher) logToFile(filename, message string) {
	// Strip ANSI colors for file logging
	cleanMessage := e.stripANSI(message)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Silently fail to avoid infinite recursion
	}
	defer file.Close()

	fmt.Fprint(file, cleanMessage)
}

// stripANSI removes ANSI color codes from text
func (e ErrorCatcher) stripANSI(text string) string {
	// Simple ANSI escape sequence removal
	result := text
	escapes := []string{Reset, Bold, Red, Green, Yellow, Blue, Magenta, Cyan, White, BrightRed, Gray}
	for _, escape := range escapes {
		result = strings.ReplaceAll(result, escape, "")
	}
	return result
}

// getConfig returns the current configuration or default
func (e ErrorCatcher) getConfig() ErrorConfig {
	if e.Config.MaxStackDepth == 0 {
		return DefaultConfig
	}
	return e.Config
}

// Set assigns an error value and handles it if not nil
// Usage: file, err := os.Open(filePath); except.Catch.Set(err)
func (e ErrorCatcher) Set(err error) error {
	if err != nil {
		info := e.buildErrorInfo(err, 1)
		e.handleError(info)
	}
	return err
}

// E is the shortest possible function name for error handling
// Usage: E(err) will check if err is not nil and handle it
func E(err error) {
	if err != nil {
		info := Catch.buildErrorInfo(err, 1)
		Catch.handleError(info)
	}
}

// F is like E but with a custom format string and context
// Usage: F(err, "failed to open %s", filename)
func F(err error, format string, args ...interface{}) {
	if err != nil {
		// Create a wrapped error with the formatted message
		wrappedErr := fmt.Errorf(format+": %w", append(args, err)...)
		info := Catch.buildErrorInfo(wrappedErr, 1)
		Catch.handleError(info)
	}
}

// Wrap creates a new error with additional context without handling it
// Usage: return except.Wrap(err, "failed to process file %s", filename)
func Wrap(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// Must panics if err is not nil with enhanced error info
// Usage: file := Must(os.Open(filename))
func Must[T any](val T, err error) T {
	if err != nil {
		info := Catch.buildErrorInfo(err, 1)
		var msg strings.Builder
		msg.WriteString(fmt.Sprintf("Must failed in %s:%d", filepath.Base(info.File), info.Line))
		if info.Function != "" {
			msg.WriteString(fmt.Sprintf(" (%s)", info.Function))
		}
		msg.WriteString(fmt.Sprintf(": %v", err))
		panic(msg.String())
	}
	return val
}

// Try returns a function that will check the error and handle it
// Usage: defer Try()(&err)
func Try() func(*error) {
	return func(errp *error) {
		if errp != nil && *errp != nil {
			info := Catch.buildErrorInfo(*errp, 2) // Skip 2 levels for defer
			Catch.handleError(info)
		}
	}
}

// Assert checks a condition and creates an error if false
// Usage: except.Assert(len(items) > 0, "items slice cannot be empty")
func Assert(condition bool, message string, args ...interface{}) {
	if !condition {
		err := fmt.Errorf("assertion failed: "+message, args...)
		info := Catch.buildErrorInfo(err, 1)
		Catch.handleError(info)
	}
}

// Check is a convenient function that returns true if error is nil
// Usage: if !except.Check(err) { return }
func Check(err error) bool {
	if err != nil {
		info := Catch.buildErrorInfo(err, 1)
		Catch.handleError(info)
		return false
	}
	return true
}

// Recover handles panics and converts them to errors
// Usage: defer except.Recover()(&err)
func Recover() func(*error) {
	return func(errp *error) {
		if r := recover(); r != nil {
			var err error
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = fmt.Errorf("panic: %s", v)
			default:
				err = fmt.Errorf("panic: %v", v)
			}

			if errp != nil {
				*errp = err
			} else {
				info := Catch.buildErrorInfo(err, 2)
				Catch.handleError(info)
			}
		}
	}
}

// Additional convenience functions that work with X()

// Xf formats and handles error with context
// Usage: except.Xf(err, "failed to process %s", filename)
func Errf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	wrappedErr := fmt.Errorf(format+": %w", append(args, err)...)
	return Err(wrappedErr, args...)
}

// XMust panics with smart error info if err is not nil
// Usage: file := except.XMust(os.Open(filename))
func ErrMust[T any](val T, err error) T {
	if err != nil {
		Err(err) // This will exit due to default config
	}
	return val
}

// XCheck returns false and handles error if not nil
// Usage: if !except.XCheck(err) { return }
func ErrCheck(err error) bool {
	if err != nil {
		Err(err)
		return false
	}
	return true
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Examples of usage:
/*
// Ultra simple - auto-detects everything
except.X(err)

// With explicit context
except.X(err, filename)
except.X(err, "file_processing", filename)
except.X(err, map[string]interface{}{"file": filename, "operation": "read"})

// Auto-formatting
except.Xf(err, "failed to process %s", filename)

// Must pattern
file := except.XMust(os.Open(filename))

// Check pattern
if !except.XCheck(err) { return }

// All original functions still work:
except.E(err)
except.F(err, "message")
except.Catch.Set(err)
except.Catch.WithContext("key", value).Set(err)
*/
