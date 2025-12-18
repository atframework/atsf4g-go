---
applyTo: "**/*_test.go"
---

# Golang Unit Test Coding Conventions

## Common Specifications

### Basic Specifications

- Test File Naming: Corresponding to the tested file `xxx.go`, the test file must be named `xxx_test.go`.
- Test Function Naming: Follow the format `Test[TestedFunction/MethodName][ScenarioDescription]`.
- Testing Framework: Use Go's native `testing` package; the `github.com/stretchr/testify/assert` assertion library is optional.

### Test Coverage Requirements

- Cover core business logic, all branches (if/else/switch), and loop boundaries.
- Cover boundary conditions (0, empty values, maximum values, minimum values, nil) and abnormal input scenarios.
- Cover error return and panic scenarios (panic must be captured and verified using `recover`).

### Dependency Handling

- Isolate external dependencies (databases, HTTP, file systems, etc.) via interface mocks; prioritize implementing mocks with native anonymous structs.
- Prohibit reliance on real environments to ensure tests can run independently without side effects.

### Code Structure

- Each test case follows the Arrange-Act-Assert (AAA) pattern.
- Add comments to each test case to explain the test scenario; supplement design rationale for key logic.
- Test cases are independent and stateless, with no mutual interference.

### Assertion Requirements

- Use `errors.Is`/`errors.As` to verify error types; avoid directly comparing error strings.
- Each test case verifies only one core logic, with precise and clear assertions.

Please generate complete test code (including necessary imports and mock struct definitions), and list any uncovered scenarios with reasons at the end.

## Classification Specifications

### Unit Tests for Independent Utility Functions

**Testing Framework:**
- Go's native `testing` package + `github.com/stretchr/testify/assert`

**Naming Specifications:**
- Test functions follow the format `Test[FunctionName][InputCharacteristic][ExpectedResult]`

**Test Coverage:**
- Cover all branch logic, and scenarios with valid/invalid parameters
- Cover boundary values (0, empty strings, maximum values, nil)
- Include tests for panic scenarios (if the function may trigger panic)

**Code Requirements:**
- No mocks required (no external dependencies); directly verify input and output
- Each test case follows the Arrange-Act-Assert structure and includes comments explaining the scenario
- Use the `assert` library to simplify assertions; verify errors using `errors.Is`/`errors.As`

### Unit Tests for Struct Methods with External Dependencies

**Testing Framework:**
- Go's native `testing` package + `github.com/stretchr/testify/assert`

**Dependency Handling:**
- Define a mock struct that implements the [DependentInterfaceName] interface to replace real external dependencies
- Mock methods must support verification of call count and parameter passing correctness

**Test Coverage:**
- Core scenarios: Normal execution, dependent call failure, invalid parameters, non-existent resources
- Boundary conditions: Parameters with 0/nil/empty values, dependencies returning critical data

**Code Requirements:**
- Each test case follows the structure: Arrange (initialize mocks + test data) → Act (call the method) → Assert (verify results + mock calls)
- Clearly label the design purpose of mock logic; add comments for test scenarios
- Prohibit reliance on real databases/HTTP services to ensure tests can run independently

### Interface Implementation Classes Testing

**Testing Framework:**
- Go's native `testing` package

**Core Goal:**
- Verify that the implementation class fully complies with all method contracts of the [InterfaceName] interface

**Test Coverage:**
- For each interface method, cover core logic, boundary conditions, and error scenarios
- Call methods via the interface type in test cases (to ensure interface compatibility)

**Code Requirements:**
- Test directly if there are no external dependencies; use native mocks for isolation if dependencies exist
- Split test cases for each method by scenario and add detailed comments
- Verify method input/output, error types, and side effects (if any)

### Unit Tests for Concurrency-Related Code

**Testing Framework:**
- Go's native `testing` package + `github.com/stretchr/testify/assert`

**Testing Focus:**
- Concurrent safety: No data races when multiple goroutines call the code simultaneously
- Functional correctness: Results meet expectations in concurrent scenarios
- Performance benchmarking (optional): Generate Benchmark tests to evaluate concurrent performance

**Code Requirements:**
- Use `sync.WaitGroup` to control concurrent goroutines and ensure complete test execution
- Enable data race detection code (e.g., `testing.M`'s `RaceDetectorMode`)
- For each test case, clearly specify the number of concurrent tasks and execution logic; add comments for test scenarios
- Dependency isolation: Replace external dependencies with mocks to avoid concurrent conflicts

### Benchmark Tests

**Testing Framework:**
- Go's native `testing` package

**Naming Specifications:**
- Benchmark functions follow the format `Benchmark[TestedFunction/MethodName]`

**Testing Requirements:**
- Test performance under different input scales (e.g., small, medium, and large data volumes)
- Avoid redundant operations in test code (e.g., creating objects inside loops)
- Set a reasonable `b.N` loop to ensure stable test results

**Code Requirements:**
- Include necessary Setup (initialize test data) and Teardown logic
- Use mocks to isolate external dependencies (if any) to avoid affecting performance test results
- Generate complete Benchmark code and add comments explaining test scenarios
