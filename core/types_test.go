package core

import "testing"

func TestGetMethodNormalizesCasing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		configured string
		want       string
	}{
		{"get", METHOD_GET},
		{"GET", METHOD_GET},
		{"Post", METHOD_POST},
		{"  put  ", METHOD_PUT},
		{"patch", METHOD_PATCH},
		{"delete", METHOD_DELETE},
		{"head", METHOD_HEAD},
		{"options", METHOD_OPTIONS},
	}

	for _, tc := range tests {
		task := HttpTask{Method: tc.configured}
		if got := task.GetMethod(); got != tc.want {
			t.Errorf("HttpTask{Method: %q}.GetMethod() = %q, want %q", tc.configured, got, tc.want)
		}
	}
}

func TestGetMethodRejectsUnusableValues(t *testing.T) {
	t.Parallel()

	for _, configured := range []string{"", "   ", "NOTAMETHOD", "TRACE"} {
		task := HttpTask{Method: configured}
		if got := task.GetMethod(); got != METHOD_UNAVAILABLE {
			t.Errorf("HttpTask{Method: %q}.GetMethod() = %q, want %q", configured, got, METHOD_UNAVAILABLE)
		}
	}
}

func TestEnforceMethodsAcceptsOptions(t *testing.T) {
	t.Parallel()

	if !enforceMethods(METHOD_OPTIONS) {
		t.Error("OPTIONS is a defined method constant and should be accepted")
	}
}
