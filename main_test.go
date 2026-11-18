package main

import "testing"

func TestParseArgsListAndSet(t *testing.T) {
	opts, code, done := parseArgs([]string{"--set", "team=core", "--set", "tags=a,b", "--list", "o/r"})
	if done || code != 0 {
		t.Fatalf("done=%v code=%d", done, code)
	}
	if !opts.list || opts.repoArg != "o/r" || len(opts.set) != 2 {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if opts.set[1].Name != "tags" || opts.set[1].Value != "a,b" {
		t.Errorf("set[1] = %+v", opts.set[1])
	}
}

func TestParseArgsBadSet(t *testing.T) {
	if _, code, done := parseArgs([]string{"--set", "novalue", "o/r"}); !done || code != 2 {
		t.Errorf("done=%v code=%d, want done with code 2", done, code)
	}
}
