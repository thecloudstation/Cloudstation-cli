# Rust Application Validation Test Suite

Execute comprehensive validation tests for the Rust CLI application, returning results in a standardized JSON format for automated processing.

## Purpose

Proactively identify and fix issues in the application before they impact users or developers. By running this comprehensive test suite, you can:
- Detect compilation errors, type mismatches, and borrowing issues
- Identify broken tests or security vulnerabilities
- Verify code formatting, linting, and build processes
- Ensure the application is in a healthy state

## Variables

TEST_COMMAND_TIMEOUT: 5 minutes

## Instructions

- Execute each test in the sequence provided below
- Capture the result (passed/failed) and any error messages
- IMPORTANT: Return ONLY the JSON array with test results
  - IMPORTANT: Do not include any additional text, explanations, or markdown formatting
  - We'll immediately run JSON.parse() on the output, so make sure it's valid JSON
- If a test passes, omit the error field
- If a test fails, include the error message in the error field
- Execute all tests even if some fail
- Error Handling:
  - If a command returns non-zero exit code, mark as failed and immediately stop processing tests
  - Capture stderr output for error field
  - Timeout commands after `TEST_COMMAND_TIMEOUT`
  - IMPORTANT: If a test fails, stop processing tests and return the results thus far
- Test execution order is important - formatting and linting should be validated before running tests
- All file paths are relative to the claude-wrapper directory
- Working directory: `/Users/oumnyabenhassou/Code/tac/tac-8/claude-wrapper`

## Test Execution Sequence

### Rust Tests

1. **Rust Format Check**
   - Preparation Command: None
   - Command: `cargo fmt -- --check`
   - test_name: "rust_format_check"
   - test_purpose: "Validates Rust code formatting standards using rustfmt, ensuring consistent code style across the project"

2. **Rust Linting (Clippy)**
   - Preparation Command: None
   - Command: `cargo clippy -- -D warnings`
   - test_name: "rust_clippy"
   - test_purpose: "Identifies potential bugs, performance issues, code smells, and non-idiomatic Rust patterns using Clippy linter"

3. **Rust Compilation Check**
   - Preparation Command: None
   - Command: `cargo check`
   - test_name: "rust_check"
   - test_purpose: "Fast compilation check without producing binaries, validates syntax, types, borrowing rules, and lifetimes"

4. **All Rust Tests**
   - Preparation Command: None
   - Command: `cargo test --verbose`
   - test_name: "all_rust_tests"
   - test_purpose: "Runs all unit and integration tests, validating application functionality including API endpoints, state management, workflows, and data processing"

5. **Release Build**
   - Preparation Command: None
   - Command: `cargo build --release`
   - test_name: "rust_release_build"
   - test_purpose: "Validates production build compilation with optimizations enabled, ensuring the application can be built for deployment"

## Report

- IMPORTANT: Return results exclusively as a JSON array based on the `Output Structure` section below.
- Sort the JSON array with failed tests (passed: false) at the top
- Include all tests in the output, both passed and failed
- The execution_command field should contain the exact command that can be run to reproduce the test
- This allows subsequent agents to quickly identify and resolve errors

### Output Structure

```json
[
  {
    "test_name": "string",
    "passed": boolean,
    "execution_command": "string",
    "test_purpose": "string",
    "error": "optional string"
  },
  ...
]
```

### Example Output

```json
[
  {
    "test_name": "rust_clippy",
    "passed": false,
    "execution_command": "cargo clippy -- -D warnings",
    "test_purpose": "Identifies potential bugs, performance issues, code smells, and non-idiomatic Rust patterns using Clippy linter",
    "error": "warning: unused variable: `result`\n  --> src/main.rs:42:9\n   |\n42 |     let result = process_data();\n   |         ^^^^^^ help: if this is intentional, prefix it with an underscore: `_result`"
  },
  {
    "test_name": "rust_format_check",
    "passed": true,
    "execution_command": "cargo fmt -- --check",
    "test_purpose": "Validates Rust code formatting standards using rustfmt, ensuring consistent code style across the project"
  },
  {
    "test_name": "all_rust_tests",
    "passed": true,
    "execution_command": "cargo test --verbose",
    "test_purpose": "Runs all unit and integration tests, validating application functionality including API endpoints, state management, workflows, and data processing"
  }
]
```