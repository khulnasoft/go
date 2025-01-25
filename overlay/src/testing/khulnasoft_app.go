//go:build khulnasoft
// +build khulnasoft

package testing

import _ "unsafe"

// khulnasoftStartTest is called when a test starts running. This allows Khulnasoft's testing framework to
// isolate behavior between different tests on global state. It is linked to the Khulnasoft runtime via go:linkname.
//go:linkname khulnasoftStartTest khulnasoft.runtime.khulnasoftStartTest
func khulnasoftStartTest(t *T, fn func(t *T))

// khulnasoftEndTest is called when a test ends. This allows Khulnasoft's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Khulnasoft runtime via go:linkname.
//go:linkname khulnasoftEndTest khulnasoft.runtime.khulnasoftEndTest
func khulnasoftEndTest(t *T)

// khulnasoftPauseTest is called when a test is paused. This allows Khulnasoft's testing framework to
// isolate behavior between different tests on global state. It is linked to the Khulnasoft runtime via go:linkname.
//go:linkname khulnasoftPauseTest khulnasoft.runtime.khulnasoftPauseTest
func khulnasoftPauseTest(t *T)

// khulnasoftResumeTest is called when a test is resumed after being paused. This allows Khulnasoft's testing framework to clear down any state from the test
// and to perform any assertions on that state that it needs to. It is linked to the Khulnasoft runtime via go:linkname.
//go:linkname khulnasoftResumeTest khulnasoft.runtime.khulnasoftResumeTest
func khulnasoftResumeTest(t *T)

// khulnasoftTestLog is called when a test logs a line. This allows Khulnasoft's testing framework to capture the log output
// and emit that log output to the test trace.
func khulnasoftTestLog(line string, frameSkip int)
