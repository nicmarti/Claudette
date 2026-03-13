package parser

import "regexp"

var testPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^test_`),
	regexp.MustCompile(`^Test`),
	regexp.MustCompile(`_test$`),
	regexp.MustCompile(`\.test\.`),
	regexp.MustCompile(`\.spec\.`),
	regexp.MustCompile(`_spec$`),
}

var testFilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`test_.*\.py$`),
	regexp.MustCompile(`.*_test\.py$`),
	regexp.MustCompile(`.*\.test\.[jt]sx?$`),
	regexp.MustCompile(`.*\.spec\.[jt]sx?$`),
	regexp.MustCompile(`.*_test\.go$`),
	regexp.MustCompile(`.*Tests?\.java$`),
	regexp.MustCompile(`tests?/`),
}

// IsTestFile returns true if the file path matches test file patterns.
func IsTestFile(path string) bool {
	for _, p := range testFilePatterns {
		if p.MatchString(path) {
			return true
		}
	}
	return false
}

// IsTestFunction returns true if the function name matches test patterns.
func IsTestFunction(name string) bool {
	for _, p := range testPatterns {
		if p.MatchString(name) {
			return true
		}
	}
	return false
}
