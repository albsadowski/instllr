package main

import "testing"

func TestCmpVersion(t *testing.T) {
	cases := []struct {
		v1, v2   string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.10.0", "1.9.0", 1},
		{"1.9.0", "1.10.0", -1},
		{"10.0.0", "9.0.0", 1},
		{"9.0.0", "10.0.0", -1},
	}

	for _, c := range cases {
		actual := cmpVersion(c.v1, c.v2)
		if actual != c.expected {
			t.Errorf("cmpVersion(%s, %s) == %d, expected %d", c.v1, c.v2, actual, c.expected)
		}
	}
}
