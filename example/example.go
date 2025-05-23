package main

import (
	"except"
	"fmt"
	"io"
	"os"
)

// var e = except.E

func main() {
	// Change this to the path of the file you want to read
	filePath := "example.txt"

	// Example 1: Using the E function (most concise)
	fmt.Println("=== Example 1: Using E function ===")
	example1(filePath)

	// Example 2: Using the global Catch variable
	fmt.Println("\n=== Example 2: Using global Catch variable ===")
	example2(filePath)

	// Example 3: Using a short alias for even more concise syntax
	fmt.Println("\n=== Example 3: Using short alias ===")
	example3(filePath)
}

// Example 1: Using the E function (most concise)
func example1(filePath string) {
	file, err := os.Open(filePath)
	except.E(err) // Single line error handling
	defer file.Close()

	_, err = io.Copy(os.Stdout, file)
	except.E(err) // Single line error handling
}

// Example 2: Using the global Catch variable
func example2(filePath string) {
	file, err := os.Open(filePath)
	except.Catch.Set(err)
	defer file.Close()

	_, err = io.Copy(os.Stdout, file)
	except.Catch.Set(err)
}

// Example 3: Using a short alias for even more concise syntax
func example3(filePath string) {
	// Create a short alias for the error handling function
	e := except.E

	file, err := os.Open(filePath)
	e(err) // Ultra concise error handling
	defer file.Close()

	_, err = io.Copy(os.Stdout, file)
	e(err) // Ultra concise error handling
}
