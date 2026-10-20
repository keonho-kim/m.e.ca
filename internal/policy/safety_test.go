// Purpose: Verify explicit mutation rules are shared across shell and file adapters.
// Pattern: Pure policy tests.
// Usage: go test ./internal/policy.
// Related: internal/policy/safety.go.
package policy

import "testing"

func TestMutationSafety(t *testing.T) {
	for _, test := range []struct {
		path                       string
		removal, blocked, approval bool
	}{
		{"/work/project/src/file", false, false, false},
		{"/work/project/src/file", true, false, true},
		{"/work/project", true, true, false},
		{"/work", true, true, false},
		{"/", true, true, false},
		{"/home/user", true, true, false},
		{"/home/user/bin/tool", true, true, false},
		{"/home/user/.cache/tool", false, false, true},
		{"/etc/passwd", false, true, false},
		{"/private/etc/passwd", true, true, false},
		{"/usr/lib/libc.so", true, true, false},
		{"/dev/null", false, false, false},
		{"/dev/null", true, true, false},
		{"/tmp/specific-file", true, false, true},
		{"/work/project-other/file", false, false, true},
	} {
		result, err := Mutation("/work/project", "/home/user", test.path, test.removal)
		if (err != nil) != test.blocked || (len(result.Reasons) > 0) != test.approval {
			t.Errorf("%+v: %+v %v", test, result, err)
		}
	}
}

func TestMutationWithinDirectory(t *testing.T) {
	for _, test := range []struct {
		path              string
		blocked, approval bool
	}{
		{"/work/project", false, true},
		{"/work/project/src", false, true},
		{"/tmp/specific-directory", false, true},
		{"/work", true, false},
		{"/", true, false},
		{"/home/user", true, false},
		{"/etc", true, false},
	} {
		result, err := MutationWithin("/work/project", "/home/user", test.path, true)
		if (err != nil) != test.blocked || (len(result.Reasons) > 0) != test.approval {
			t.Errorf("%+v: %+v %v", test, result, err)
		}
		if err == nil && (len(result.Targets) != 1 || result.Targets[0] != test.path) {
			t.Errorf("directory scope was replaced: %+v", result)
		}
	}
}
