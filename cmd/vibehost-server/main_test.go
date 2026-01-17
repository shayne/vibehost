package main

import "testing"

func TestParsePortMapping(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		port   int
		found  bool
	}{
		{
			name:  "single mapping",
			input: "0.0.0.0:8080\n",
			port:  8080,
			found: true,
		},
		{
			name:  "multiple mappings",
			input: "0.0.0.0:8081\n[::]:8081\n",
			port:  8081,
			found: true,
		},
		{
			name:  "empty output",
			input: "",
			port:  0,
			found: false,
		},
		{
			name:  "noise output",
			input: "not a port\n",
			port:  0,
			found: false,
		},
	}

	for _, tc := range cases {
		port, found := parsePortMapping(tc.input)
		if found != tc.found || port != tc.port {
			t.Fatalf("%s: expected (%v, %d), got (%v, %d)", tc.name, tc.found, tc.port, found, port)
		}
	}
}
