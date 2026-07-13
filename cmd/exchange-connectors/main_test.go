package main

import (
	"testing"
)

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"BTCUSDT,ETHUSDT", []string{"BTCUSDT", "ETHUSDT"}},
		{"BTCUSDT", []string{"BTCUSDT"}},
		{"", []string{}},
		{"A,,B,", []string{"A", "B"}},
		{",leading", []string{"leading"}},
	}
	for _, c := range cases {
		got := splitCSV(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("splitCSV(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}