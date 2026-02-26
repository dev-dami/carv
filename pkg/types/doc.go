// Package types performs semantic analysis and static type checking.
//
// Design decisions:
//   - Type checking returns structured issues so tools can consume machine-readable diagnostics.
//   - Ownership, borrowing, interfaces, and async validation are checked in dedicated units to
//     keep rules isolated and maintainable.
//
// Usage pattern:
//
//	checker := types.NewChecker()
//	result := checker.Check(program)
//	if len(result.Errors) > 0 {
//	    // compilation should fail
//	}
package types
