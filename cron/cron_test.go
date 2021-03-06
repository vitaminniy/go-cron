package cron

import (
	"strings"
	"testing"
)

func CompareSlices(t *testing.T, expected, actual []uint8) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("len mismatch: want %+v; got %+v", expected, actual)
	}

	for i := range expected {
		want := expected[i]
		got := actual[i]

		if want != got {
			t.Errorf("mismatch at %d: want %d; got %d", i, want, got)
		}
	}
}

func TestParseIntegral(t *testing.T) {
	cases := []struct {
		name               string
		value              string
		min, max, expected uint8
		shouldFail         bool
	}{
		{
			name:       "success",
			value:      "10",
			min:        0,
			max:        11,
			expected:   10,
			shouldFail: false,
		},
		{
			name:       "invalid value",
			value:      "-1",
			shouldFail: true,
		},
		{
			name:       "below min",
			value:      "1",
			min:        2,
			shouldFail: true,
		},
		{
			name:       "above max",
			value:      "10",
			min:        2,
			max:        3,
			shouldFail: true,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			v, err := parseIntegral(c.value, c.min, c.max)
			if err != nil {
				if c.shouldFail {
					return
				}
				t.Fatalf("parse of %q failed: %v", c.value, err)
			}

			if c.expected != v {
				t.Fatalf("parse mismatch: want %d; got %d", c.expected, v)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	cases := []struct {
		name       string
		value      []string
		rnge       []uint8
		min, max   uint8
		shouldFail bool
	}{
		{
			name:       "empty range",
			value:      nil,
			shouldFail: true,
		},
		{
			name:       "invalid range (too small)",
			value:      []string{"1"},
			shouldFail: true,
		},
		{
			name:       "invalid range (too big)",
			value:      []string{"1", "2", "3"},
			shouldFail: true,
		},
		{
			name:       "begin less than min",
			value:      []string{"1", "10"},
			min:        5,
			max:        15,
			shouldFail: true,
		},
		{
			name:       "begin more than max",
			value:      []string{"10", "100"},
			min:        1,
			max:        10,
			shouldFail: true,
		},
		{
			name:       "end less than min",
			value:      []string{"1", "0"},
			min:        0,
			max:        15,
			shouldFail: true,
		},
		{
			name:       "end more than max",
			value:      []string{"10", "100"},
			min:        1,
			max:        90,
			shouldFail: true,
		},
		{
			name:       "begin is more than end",
			value:      []string{"100", "10"},
			min:        0,
			max:        100,
			shouldFail: true,
		},
		{
			name:       "happy path",
			value:      []string{"1", "10"},
			min:        0,
			max:        59,
			rnge:       []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			shouldFail: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rnge, err := parseRange(c.value, c.min, c.max)
			if err != nil {
				if c.shouldFail {
					return
				}
				t.Fatalf("could not parse range: %v", err)
			}

			CompareSlices(t, c.rnge, rnge)
		})
	}
}

func TestParseSteps(t *testing.T) {
	cases := []struct {
		name       string
		steps      []string
		min, max   uint8
		expected   []uint8
		shouldFail bool
	}{
		{
			name:       "has empty step",
			steps:      []string{"", "1", "2"},
			shouldFail: true,
		},
		{
			name:       "has invalid step",
			steps:      []string{"3", "1", "4", "5"},
			min:        2,
			max:        10,
			shouldFail: true,
		},
		{
			name:       "has invalid step (non integer)",
			steps:      []string{"3", "hello", "4", "5"},
			min:        2,
			max:        10,
			shouldFail: true,
		},
		{
			name:       "non unique steps",
			steps:      []string{"1", "1", "1", "1"},
			min:        0,
			max:        10,
			expected:   []uint8{1},
			shouldFail: false,
		},
		{
			name:       "valid steps",
			steps:      []string{"1", "2", "3", "4"},
			min:        0,
			max:        10,
			expected:   []uint8{1, 2, 3, 4},
			shouldFail: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			actual, err := parseSteps(c.steps, c.min, c.max)
			if err != nil {
				if c.shouldFail {
					return
				}
				t.Fatalf("could not parse steps: %v", err)
			}

			CompareSlices(t, c.expected, actual)
		})
	}
}

func TestParseIntervals(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		min, max   uint8
		intervals  []uint8
		shouldFail bool
	}{
		{
			name:       "invalid intervals",
			input:      "/",
			shouldFail: true,
		},
		{
			name:       "zero repeat step",
			input:      "*/0",
			shouldFail: true,
		},
		{
			name:       "out of bound step",
			input:      "*/17",
			min:        minutesMin,
			max:        minutesMax,
			shouldFail: true,
		},
		{
			name:  "every 5 min",
			input: "*/5",
			min:   minutesMin,
			max:   minutesMax,
			intervals: []uint8{
				0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55,
			},
			shouldFail: false,
		},
		{
			name:  "every 5 min after 0",
			input: "0/5",
			min:   minutesMin,
			max:   minutesMax,
			intervals: []uint8{
				0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55,
			},
			shouldFail: false,
		},
		{
			name:  "every 5 min after 1",
			input: "1/5",
			min:   minutesMin,
			max:   minutesMax,
			intervals: []uint8{
				1, 6, 11, 16, 21, 26, 31, 36, 41, 46, 51, 56,
			},
			shouldFail: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			intervals, err := parseIntervals(
				strings.Split(c.input, "/"), c.min, c.max)
			if err != nil {
				if c.shouldFail {
					return
				}
				t.Fatalf("could not parse intervals: %v", err)
			}

			CompareSlices(t, c.intervals, intervals)
		})
	}
}

func TestParseTime(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		min, max uint8
		expected []uint8
	}{
		{
			name:     "parse range minutes",
			input:    "1-10",
			min:      0,
			max:      59,
			expected: []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:     "parse exact minutes",
			input:    "1",
			min:      0,
			max:      59,
			expected: []uint8{1},
		},
		{
			name:  "parse every hour",
			input: "*",
			min:   0,
			max:   23,
			expected: []uint8{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
				13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
			},
		},
		{
			name:  "parse step minutes",
			input: "15,30,45,00",
			min:   0,
			max:   59,
			expected: []uint8{
				15, 30, 45, 00,
			},
		},
		{
			name:  "parse every 15 minutes",
			input: "*/15",
			min:   0,
			max:   59,
			expected: []uint8{
				0, 15, 30, 45,
			},
		},
		{
			name:  "parse every 15 minutes from 1",
			input: "1/15",
			min:   0,
			max:   59,
			expected: []uint8{
				1, 16, 31, 46,
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			actual, err := parseTime(c.input, c.min, c.max)
			if err != nil {
				t.Fatalf("could not parse expression: %v", err)
			}

			CompareSlices(t, c.expected, actual)
		})
	}
}
