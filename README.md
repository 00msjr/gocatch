# except - Go Error Handling Simplified

A lightweight Go module that makes error handling more concise.

## Description

The `except` module is a simple utility that transforms the standard Go error handling pattern into a more concise syntax. It allows you to replace verbose error checking blocks with a single function call, making your code cleaner and more readable while still maintaining proper error handling.

## Installation

```bash
go get github.com/soup-ms/except
```

## Usage

The `except` module provides several ways to simplify error handling in Go:

### 1. The E Function (Most Concise)

```go
// Traditional Go error handling:
file, err := os.Open(filePath)
if err != nil {
    fmt.Printf("Error opening file: %v\n", err)
    return
}

// With except module:
file, err := os.Open(filePath)
except.E(err) // Single line error handling
```

### 2. Using the Global Catch Variable

```go
file, err := os.Open(filePath)
except.Catch.Set(err) // Handles the error if it's not nil
```

### 3. Using a Short Alias for Ultra-Concise Syntax

```go
// Create a short alias at the beginning of your function or file
e := except.E

// Then use it for ultra-concise error handling
file, err := os.Open(filePath)
e(err) // Ultra concise error handling
```

### 4. The F Function (With Custom Messages)

```go
file, err := os.Open(filePath)
except.F(err, "failed to open %s", filePath) // With custom message
```

### 5. The Must Function (For One-liners)

```go
file := except.Must(os.Open(filePath)) // Returns the value if no error
```

### 6. The Try Function (With Defer)

```go
func readFile(filePath string) {
    var err error
    defer except.Try()(&err) // Will check err at the end of function

    file, err := os.Open(filePath)
    if err != nil {
        return
    }
    defer file.Close()

    _, err = io.Copy(os.Stdout, file)
}
```

## Error Handling Behavior

All error handling functions in the module will:

1. Check if the error is not nil
2. If an error exists, print the error with file and line information
3. Exit the program with status code 1 (except for `Must` which panics)

This approach is particularly useful for scripts, tools, and applications where you want to fail fast and provide clear error messages.

## Authors

[@soup-ms](https://github.com/soup-ms)

## Version History

* v0.1.0
  * Initial Release

## License

This project is licensed under the MIT License - see the LICENSE.md file for details
