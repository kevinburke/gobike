package main

import "testing"

func TestPrefixAfter(t *testing.T) {
	prefix := "2018-09-01"
	if "2018-09-02" <= prefix {
		t.Errorf("2018-09-02 should be greater than 09-01, wasn't")
	}
	if "2018-10-01" <= prefix {
		t.Errorf("2018-10-01 should be greater than 09-01, wasn't")
	}
	if "2019-09-01" <= prefix {
		t.Errorf("2019-09-01 should be greater than 09-01, wasn't")
	}
	if "2019-01-01" <= prefix {
		t.Errorf("2019-01-01 should be greater than 09-01, wasn't")
	}
}
