//go:build !khulnsoft
// +build !khulnsoft

package testing

// khulnsoftStartTest is called when a test starts running. This allows Khulnsoft's testing framework to
// isolate behavior between different tests on global state.
//
// This implementation is simply a no-op as it's used when the tests are not being run against an Khulnsoft application
func khulnsoftStartTest(t *T, fn func(t *T)) {}

// khulnsoftEndTest is called when a test ends. This allows Khulnsoft's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Khulnsoft runtime via go:linkname.
//
// This implementation is simply a no-op as it's used when the tests are not being run against an Khulnsoft application
func khulnsoftEndTest(t *T) {}

// khulnsoftPauseTest is called when a test is paused. This allows Khulnsoft's testing framework to
// isolate behavior between different tests on global state. It is linked to the Khulnsoft runtime via go:linkname.
func khulnsoftPauseTest(t *T) {}

// khulnsoftResumeTest is called when a test is resumed after being paused. This allows Khulnsoft's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Khulnsoft runtime via go:linkname.
func khulnsoftResumeTest(t *T) {}

// khulnsoftTestLog is called when a test logs a line. This allows Khulnsoft's testing framework to capture the log output
// and emit that log output to the test trace.
func khulnsoftTestLog(line string, frameSkip int) {}
