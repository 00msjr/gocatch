package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"except" // Adjust import path as needed
)

func main() {
	fmt.Println("🧪 Error Handling Module Test Suite")
	fmt.Println("====================================")

	// Configure error handling for non-fatal testing
	except.Catch.Configure(except.ErrorConfig{
		ShowStackTrace: true,
		ExitOnError:    false, // Set to true to see fatal behavior
		LogToFile:      "test_errors.log",
		MaxStackDepth:  5,
	})

	fmt.Println("\n1. Testing basic error handling...")
	testBasicErrorHandling()

	fmt.Println("\n2. Testing contextual error handling...")
	testContextualErrors()

	fmt.Println("\n3. Testing custom formatting...")
	testCustomFormatting()

	fmt.Println("\n4. Testing Must function...")
	testMustFunction()

	fmt.Println("\n5. Testing assertions...")
	testAssertions()

	fmt.Println("\n6. Testing error wrapping...")
	testErrorWrapping()

	fmt.Println("\n7. Testing panic recovery...")
	testPanicRecovery()

	fmt.Println("\n8. Testing Try with defer...")
	testTryDefer()

	fmt.Println("\n9. Testing Check function...")
	testCheckFunction()

	fmt.Println("\n10. Testing file operations...")
	testFileOperations()

	fmt.Println("\n✅ Test suite completed! Check 'test_errors.log' for logged errors.")
}

// testBasicErrorHandling demonstrates E() and Set() functions
func testBasicErrorHandling() {
	fmt.Println("  • Testing successful operation...")
	err := workingFunction()
	except.E(err) // Should pass silently
	fmt.Println("    ✓ Working function completed")

	fmt.Println("  • Testing failing operation...")
	err = failingFunction()
	except.E(err) // Will log error but not exit (ExitOnError: false)
	fmt.Println("    ✓ Error handled and logged")

	// Test Catch.Set()
	fmt.Println("  • Testing Catch.Set()...")
	_, err = brokenFileOpen()
	except.Catch.Set(err) // Alternative syntax
	fmt.Println("    ✓ Catch.Set() handled error")
}

// testContextualErrors demonstrates contextual error handling
func testContextualErrors() {
	fmt.Println("  • Testing error with context...")

	userID := 12345
	operation := "user_profile_update"

	err := simulateUserError(userID)
	except.Catch.WithContext("user_id", userID).
		WithContext("operation", operation).
		WithContext("timestamp", time.Now().Unix()).
		WithContext("client_ip", "192.168.1.100").
		Set(err)

	fmt.Println("    ✓ Contextual error handled")
}

// testCustomFormatting demonstrates F() function
func testCustomFormatting() {
	fmt.Println("  • Testing custom error formatting...")

	filename := "config.json"
	userID := 999

	err := simulateConfigError()
	except.F(err, "failed to load configuration file '%s' for user %d", filename, userID)

	fmt.Println("    ✓ Custom formatted error handled")
}

// testMustFunction demonstrates Must() with both success and failure
func testMustFunction() {
	fmt.Println("  • Testing Must() with working function...")

	// This will work
	result := except.Must(workingIntFunction())
	fmt.Printf("    ✓ Must() succeeded with result: %d\n", result)

	fmt.Println("  • Testing Must() with failing function...")

	// Catch panic from Must()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("    ✓ Must() panic caught: %v\n", r)
		}
	}()

	// This will panic
	_ = except.Must(failingIntFunction())
	fmt.Println("    ❌ This line should not be reached")
}

// testAssertions demonstrates Assert() function
func testAssertions() {
	fmt.Println("  • Testing successful assertion...")
	slice := []int{1, 2, 3}
	except.Assert(len(slice) > 0, "slice should not be empty")
	fmt.Println("    ✓ Assertion passed")

	fmt.Println("  • Testing failing assertion...")
	emptySlice := []int{}
	except.Assert(len(emptySlice) > 0, "slice should not be empty, got length %d", len(emptySlice))
	fmt.Println("    ✓ Assertion failure handled")
}

// testErrorWrapping demonstrates Wrap() function
func testErrorWrapping() {
	fmt.Println("  • Testing error wrapping...")

	err := processUserData(999)
	if err != nil {
		fmt.Printf("    ✓ Wrapped error: %v\n", err)
	}
}

// testPanicRecovery demonstrates Recover() function
func testPanicRecovery() {
	fmt.Println("  • Testing panic recovery...")

	var err error
	defer except.Recover()(&err)

	// This will panic
	panicFunction()

	if err != nil {
		fmt.Printf("    ✓ Panic recovered as error: %v\n", err)
	}
}

// testTryDefer demonstrates Try() with defer
func testTryDefer() {
	fmt.Println("  • Testing Try() with defer...")

	err := operationWithDefer()
	if err != nil {
		fmt.Printf("    ✓ Deferred error handling completed\n")
	}
}

// testCheckFunction demonstrates Check() function
func testCheckFunction() {
	fmt.Println("  • Testing Check() function...")

	err := workingFunction()
	if except.Check(err) {
		fmt.Println("    ✓ Check() returned true for nil error")
	}

	err = failingFunction()
	if !except.Check(err) {
		fmt.Println("    ✓ Check() returned false and handled error")
	}
}

// testFileOperations demonstrates real-world file operation errors
func testFileOperations() {
	fmt.Println("  • Testing file operations...")

	// Try to open non-existent file
	filename := "nonexistent_file.txt"
	file, err := os.Open(filename)
	except.Catch.WithContext("filename", filename).
		WithContext("operation", "file_open").
		Set(err)

	if file != nil {
		file.Close()
	}

	fmt.Println("    ✓ File operation error handled")
}

// ==================== HELPER FUNCTIONS ====================

// Working functions (return nil errors)
func workingFunction() error {
	// Simulate successful operation
	return nil
}

func workingIntFunction() (int, error) {
	return 42, nil
}

// Failing functions (return errors)
func failingFunction() error {
	return errors.New("something went wrong in failingFunction")
}

func failingIntFunction() (int, error) {
	return 0, errors.New("failed to get integer value")
}

func brokenFileOpen() (*os.File, error) {
	// Try to open a file that doesn't exist
	return os.Open("/nonexistent/path/file.txt")
}

func simulateUserError(userID int) error {
	return fmt.Errorf("user %d not found in database", userID)
}

func simulateConfigError() error {
	return errors.New("invalid JSON format")
}

func processUserData(userID int) error {
	err := fetchUserFromDB(userID)
	if err != nil {
		return except.Wrap(err, "failed to process user data for user %d", userID)
	}
	return nil
}

func fetchUserFromDB(userID int) error {
	// Simulate database error
	return errors.New("database connection timeout")
}

func panicFunction() {
	// This will cause a panic
	var slice []int
	_ = slice[10] // Index out of range panic
}

func operationWithDefer() error {
	var err error
	defer except.Try()(&err)

	// Simulate an operation that might fail
	err = errors.New("operation failed in deferred function")
	return err
}

// ==================== INSTRUCTIONS TO BREAK FUNCTIONS ====================

/*
HOW TO TEST DIFFERENT ERROR SCENARIOS:

1. MAKE ERRORS FATAL:
   - Change ExitOnError to true in main() to see program termination
   - Comment out the defer recover() in testMustFunction() to see panic

2. BREAK FILE OPERATIONS:
   - Try to open files without read permissions: chmod 000 somefile.txt
   - Try to write to read-only directories
   - Use invalid file paths with special characters

3. BREAK NETWORK OPERATIONS (add these to test):
   ```go
   import "net/http"

   func testNetworkError() {
       resp, err := http.Get("http://invalid-domain-12345.com")
       except.F(err, "failed to fetch data from %s", "invalid-domain-12345.com")
       if resp != nil {
           resp.Body.Close()
       }
   }
   ```

4. BREAK JSON OPERATIONS (add these to test):
   ```go
   import "encoding/json"

   func testJSONError() {
       invalidJSON := `{"name": "test", "age":}`
       var data map[string]interface{}
       err := json.Unmarshal([]byte(invalidJSON), &data)
       except.F(err, "failed to parse JSON data")
   }
   ```

5. BREAK STRING CONVERSIONS:
   ```go
   func testConversionError() {
       _, err := strconv.Atoi("not-a-number")
       except.F(err, "failed to convert string to integer")
   }
   ```

6. CREATE CUSTOM ERRORS:
   ```go
   func testCustomError() {
       err := fmt.Errorf("custom error: %w", errors.New("underlying cause"))
       except.Catch.WithContext("error_type", "custom").Set(err)
   }
   ```

7. TEST WITH DIFFERENT ERROR TYPES:
   - os.PathError (file system errors)
   - net.Error (network errors)
   - json.SyntaxError (JSON parsing errors)
   - strconv.NumError (number conversion errors)

8. STRESS TEST WITH GOROUTINES:
   ```go
   func testConcurrentErrors() {
       for i := 0; i < 10; i++ {
           go func(id int) {
               err := fmt.Errorf("goroutine %d error", id)
               except.Catch.WithContext("goroutine_id", id).Set(err)
           }(i)
       }
       time.Sleep(100 * time.Millisecond)
   }
   ```

REMEMBER:
- Set ExitOnError: true to see real fatal behavior
- Check the test_errors.log file for all logged errors
- Uncomment different test sections to focus on specific error types
- Add your own broken functions to test edge cases
*/
