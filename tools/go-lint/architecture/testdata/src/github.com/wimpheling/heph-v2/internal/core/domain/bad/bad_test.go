package bad

import "testing"

func TestDomainRestrictionsExcludeTests(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("test logic may use computations")
	}
}
