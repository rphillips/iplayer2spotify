package main

import (
	"testing"
	"io/ioutil"
)

func TestParser(t *testing.T) {
	data, _ := ioutil.ReadFile("fixtures/bbc-1.html")
	segments := parseSegments(string(data))
	if len(segments) != 11 {
		t.Error("invalid number of segments")
	}
}