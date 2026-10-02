package main

import (
	"net/url"
	"testing"
)

func TestParseParams(t *testing.T) {
	cases := []struct {
		q  string
		ok bool
	}{{"venue=home&season=2025", true}, {"", true}, {"venue=moon", false}, {"season=abc", false}, {"season=1800", false}}
	for _, c := range cases {
		v, _ := url.ParseQuery(c.q)
		if _, _, err := parseParams(v); (err == nil) != c.ok {
			t.Errorf("%q: ok=%v, err=%v", c.q, c.ok, err)
		}
	}
}
