package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

// nolint: funlen, dupl
func TestMatchJSON(t *testing.T) {
	tt := map[string]struct {
		a           string
		b           string
		expected    bool
		expectedLog string
	}{
		"a matches b": {
			a:        `{ "dogs": [ "fido", "spot", "snoopy" ], "cats": [ "mittens", "sylvester" ] }`,
			b:        `{ "cats": [ "sylvester", "mittens" ], "dogs": [ "snoopy", "fido", "spot" ]}`,
			expected: true,
		},
		"a does not match b": {
			a:        `{ "dogs": [ "fido", "spot", "snoopy" ], "cats": [ "mittens", "sylvester" ] }`,
			b:        `{ "cats": [ "sylvester", "mittens", "garfield" ], "dogs": [ "snoopy", "fido", "spot" ]}`,
			expected: false,
		},
		"invalid a": {
			a:           `{ "dogs": "fido", "spot", "snoopy" ], "cats": [ "mittens", "sylvester" ] }`,
			b:           `{ "cats": [ "sylvester", "mittens", "garfield" ], "dogs": [ "snoopy", "fido", "spot" ]}`,
			expected:    false,
			expectedLog: "error unmarshaling json: invalid character ',' after object key",
		},
		"invalid b": {
			a:           `{ "dogs": [ "fido", "spot", "snoopy" ], "cats": [ "mittens", "sylvester" ] }`,
			b:           ` "cats": [ "sylvester", "mittens", "garfield" ], "dogs": [ "snoopy", "fido", "spot" ]}`,
			expected:    false,
			expectedLog: "error unmarshaling json: invalid character ':' after top-level value",
		},
		"issue_17 out of order": {
			a: `{ "a": [
			      { "n": "in y", "i": "y-1", "t": "y" },
			      { "n": "to x", "i": "x-1", "t": "x" },
			      { "i": "z-1", "n": "in z", "t": "z" }
			    ]}`,
			b: `{ "a": [
		        { "n": "to x", "i": "x-1", "t": "x" },
		        { "n": "in y", "i": "y-1", "t": "y" },
		        { "n": "in z", "i": "z-1", "t": "z" }
		      ]}`,
			expected: true,
		},
		"issue_17 in order": {
			a: `{ "a": [
						{ "i": "x-1", "n": "to x", "t": "x" },
						{ "i": "y-1", "n": "in y", "t": "y" },
						{ "i": "z-1", "n": "in z", "t": "z" }
					]}`,
			b: `{ "a": [
						{ "i": "x-1", "n": "to x", "t": "x" },
						{ "i": "y-1", "n": "in y", "t": "y" },
						{ "i": "z-1", "n": "in z", "t": "z" }
					]}`,
			expected: true,
		},
	}

	for name, tc := range tt {
		tc := tc

		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer func() {
				log.SetOutput(os.Stderr)
			}()
			got := matchJSON(tc.a, tc.b)
			if tc.expectedLog == "" && buf.String() != "" {
				t.Errorf("matchJSON() failed. Expecting empty log. Got: %s\n", buf.String())
			} else if !strings.Contains(buf.String(), tc.expectedLog) {
				t.Errorf("matchJSON() failed. Expecting log to contain: %s\nGot: %s\n", tc.expectedLog, buf.String())
			}
			if got != tc.expected {
				t.Errorf("matchJSON() failed. Expecting:\n%v\nGot:\n%v\n", tc.expected, got)
			}
		})
	}
}

// nolint: funlen, dupl
func TestSliceSorter(t *testing.T) {
	tt := map[string]struct {
		a           interface{}
		b           interface{}
		expected    bool
		expectedLog string
	}{
		"a < b": {
			a:        "x",
			b:        "y",
			expected: true,
		},
		"a = b": {
			a:        "x",
			b:        "x",
			expected: false,
		},
		"a > b": {
			a:        "y",
			b:        "x",
			expected: false,
		},
		"a is chan type": {
			a:           make(chan int),
			b:           "b",
			expected:    false,
			expectedLog: "error marshaling to json: json: unsupported type: chan int",
		},
		"b is chan type": {
			a:           "a",
			b:           make(chan int),
			expected:    false,
			expectedLog: "error marshaling to json: json: unsupported type: chan int",
		},
	}

	for name, tc := range tt {
		tc := tc

		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer func() {
				log.SetOutput(os.Stderr)
			}()
			got := sliceSorter(tc.a, tc.b)
			if tc.expectedLog == "" && buf.String() != "" {
				t.Errorf("sliceSorter() failed. Expecting empty log. Got: %s\n", buf.String())
			} else if !strings.Contains(buf.String(), tc.expectedLog) {
				t.Errorf("sliceSorter() failed. Expecting log to contain: %s\nGot: %s\n", tc.expectedLog, buf.String())
			}
			if got != tc.expected {
				t.Errorf("Expecting:\n%v\nGot:\n%v\n", tc.expected, got)
			}
		})
	}
}
