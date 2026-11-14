package parse

import "time"

// Package is the representation of a single package being tested. The
// summary field is an event that contains all relevant information about the
// package, namely Package (name), Elapsed and Action (big pass or fail).
type Package struct {
	Summary *Event
	Tests   []*Test

	// StartTime is the time the package started running. This is only available
	// in go1.20 and above.
	StartTime time.Time

	// NoTestFiles indicates whether the package contains tests: [no test files]
	// This only occurs at the package level
	NoTestFiles bool

	// NoTests indicates a package contains one or more files with no tests. This doesn't
	// necessarily mean the file is empty or that the package doesn't have any tests.
	// Unfortunately go test marks the package summary with [no tests to run].
	NoTests bool
	// NoTestSlice holds events that contain "testing: warning: no tests to run" and
	// a non-empty test name.
	NoTestSlice []*Event

	// Cached indicates whether the test result was obtained from the cache.
	Cached bool

	// Cover reports whether the package contains coverage (go test run with -cover)
	Cover    bool
	Coverage float64

	// HasPanic marks the entire package as panicked. Game over.
	HasPanic bool
	// Once a package has been marked HasPanic all subsequent events are added to PanicEvents.
	PanicEvents []*Event

	// HasDataRace marks the entire package as having a data race.
	HasDataRace bool
	// DataRaceTests captures an individual test names as having a data race.
	DataRaceTests []string

	// HasFailedBuildOrSetup marks the package as having a failed build or setup.
	// Example: [build failed] or [setup failed]
	HasFailedBuildOrSetup bool

	// index memoizes test lookups by name. Long streams carry many events per
	// test, and scanning Tests for every event dominates the parse time.
	index map[string]*Test
}

// newPackage initializes and returns a Package.
func newPackage() *Package {
	return &Package{
		Summary: &Event{},
		Tests:   []*Test{},
	}
}

// AddEvent adds the event to a test based on test name.
func (p *Package) AddEvent(event *Event) {
	var t *Test
	if t = p.GetTest(event.Test); t == nil {
		// Test does not exist, add it to pkg.
		t = &Test{
			Name:    event.Test,
			Package: event.Package,
		}
		p.Tests = append(p.Tests, t)
		p.index[event.Test] = t
	}

	t.Events = append(t.Events, event)
}

// GetTest returns a test based on given name, if no test is found
// return nil
//
// Lookups are served from an index built once from the Tests slice, so
// repeated lookups for the same name do not rescan the slice.
func (p *Package) GetTest(name string) *Test {
	if p.index == nil {
		p.index = make(map[string]*Test, len(p.Tests))
		for _, t := range p.Tests {
			if _, ok := p.index[t.Name]; !ok {
				p.index[t.Name] = t
			}
		}
	}
	return p.index[name]
}

// TestsByAction returns all tests that identify as one of the following
// actions: pass, skip or fail.
//
// An empty slice if returned if there are no tests.
func (p *Package) TestsByAction(action Action) []*Test {
	var tests []*Test
	for _, t := range p.Tests {
		if t.Status() == action {
			tests = append(tests, t)
		}
	}
	return tests
}

// noteSpecialOutput parses the raw output to add additional metadata to the
// package: data races, cached results, missing tests and coverage.
func (p *Package) noteSpecialOutput(e *Event) {
	switch {
	case e.IsRace():
		p.HasDataRace = true
		if e.Test != "" {
			p.DataRaceTests = append(p.DataRaceTests, e.Test)
		}
	case e.IsCached():
		p.Cached = true
	case e.NoTestFiles():
		p.NoTestFiles = true
		// Manually mark [no test files] as "pass", because the go test tool reports the
		// package Summary action as "skip".
		// TODO(mf): revisit this behavior?
		p.Summary.Package = e.Package
		p.Summary.Action = ActionPass
	case e.NoTestsWarn():
		// One or more tests within the package contains no tests.
		p.NoTestSlice = append(p.NoTestSlice, e)
	case e.NoTestsToRun():
		// Only packages marked as "pass" will contain a summary line appended with [no tests to run].
		// This indicates one or more tests is marked as having no tests to run.
		p.NoTests = true
		p.Summary.Package = e.Package
		p.Summary.Action = ActionPass
	default:
		if cover, ok := e.Cover(); ok {
			p.Cover = true
			p.Coverage = cover
		}
	}
}
