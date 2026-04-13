package main

import "testing"

func TestMain_Compiles(t *testing.T) {
	t.Helper()
	if 2+2 != 4 {
		t.Fatalf("smoke test failed: expected 2+2==4")
	}
}
