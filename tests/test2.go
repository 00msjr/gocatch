package main

import (
	"except"
	"fmt"
	"os"
)

// var Err = except.X

func main() {
	testFile1()
	testFile2()
	// testFile3()
}

func testFile1() {
	file, err := os.Open("file.txt")
	if err != nil {
		fmt.Println("file operation failed")
		// log.Fatal(err)
	} else if err == nil {
		fmt.Println("file success")
	}
	defer file.Close()
	fmt.Println("\n\n\n")
}

func testFile2() {
	fmt.Println("• Testing file operations...")

	// Try to open non-existent file
	filename := "nonexistent_file.txt"
	file, err := os.Open(filename)
	except.Err(err, filename)

	if file != nil {
		file.Close()
	}

	fmt.Println("    ✓ File operation error handled")
}
