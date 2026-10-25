package main

import (
	"bytes"
	"testing"
)

func TestExplicitPathsRequired(t *testing.T) {
	for _, args := range [][]string{{}, {"backup"}, {"export"}, {"verify"}, {"restore"}, {"restore", "--archive", "/missing.zip"}, {"backup", "--force"}} {
		if e := run(args, &bytes.Buffer{}); e == nil {
			t.Fatal("missing or unsafe flags accepted", args)
		}
	}
}
