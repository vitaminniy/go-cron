package cron

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Expression represents a standard crontab expression and a command to
// execute.
//
// Misc:
// See https://en.wikipedia.org/wiki/Cron#CRON_expression for valid fields
// format.
type Expression struct {
	Minutes   []uint8
	Hours     []uint8
	MonthDays []uint8
	Months    []uint8
	WeekDays  []uint8
	Command   string
}

// DumpFormatted pretty-prints expression to w.
func (e *Expression) DumpFormatted(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', tabwriter.TabIndent)
	defer tw.Flush() // nolint:errcheck

	rows := []struct {
		name  string
		value string
	}{
		{
			name:  "minute",
			value: join(e.Minutes),
		},
		{
			name:  "hour",
			value: join(e.Hours),
		},
		{
			name:  "day of month",
			value: join(e.MonthDays),
		},
		{
			name:  "month",
			value: join(e.Months),
		},
		{
			name:  "day of week",
			value: join(e.WeekDays),
		},
		{
			name:  "command",
			value: e.Command,
		},
	}

	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row.name, row.value); err != nil {
			return fmt.Errorf("could not write %s: %w", row.name, err)
		}
	}

	return nil
}

func join(ss []uint8) string {
	sb := strings.Builder{}
	for i, s := range ss {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(strconv.Itoa(int(s)))
	}

	return sb.String()
}

// number of args in expression
//
// minutes hours monthdays months weekdays `command [args]`.
const numExressionArgs = 6
const (
	minutesMin = 0
	minutesMax = 59

	hoursMin = 0
	hoursMax = 23

	daysInMonthMin = 1
	daysInMonthMax = 31

	monthsMin = 1
	monthsMax = 12

	weekdaysMin = 0
	weekdaysMax = 6
)

// ParseExpression parses line in a valid cron expression.
func ParseExpression(line string) (e Expression, err error) {
	args := strings.SplitN(line, " ", numExressionArgs)

	if e.Minutes, err = parseTime(args[0], minutesMin, minutesMax); err != nil {
		return e, fmt.Errorf("invalid minutes arg: %w", err)
	}

	if e.Hours, err = parseTime(args[1], hoursMin, hoursMax); err != nil {
		return e, fmt.Errorf("invalid hours arg: %w", err)
	}

	if e.MonthDays, err = parseTime(args[2], daysInMonthMin, daysInMonthMax); err != nil {
		return e, fmt.Errorf("invalid monthdays arg: %w", err)
	}

	if e.Months, err = parseTime(args[3], monthsMin, monthsMax); err != nil {
		return e, fmt.Errorf("invalid month arg: %w", err)
	}

	if e.WeekDays, err = parseTime(args[4], weekdaysMin, weekdaysMax); err != nil {
		return e, fmt.Errorf("invalid weekdays arg: %w", err)
	}

	if args[5] == "" {
		return e, errors.New("expected command but got an empty string")
	}

	e.Command = args[5]

	return e, nil
}

func parseTime(arg string, min, max uint8) ([]uint8, error) {
	if arg == "" {
		return nil, errors.New("expected arg but got empty string")
	}

	if arg == "*" {
		result := make([]uint8, 0, max-min+1)
		for i := min; i <= max; i++ {
			result = append(result, i)
		}

		return result, nil
	}

	rnge := strings.Split(arg, "-")
	if len(rnge) > 1 {
		return parseRange(rnge, min, max)
	}

	steps := strings.Split(arg, ",")
	if len(steps) > 1 {
		return parseSteps(steps, min, max)
	}

	intervals := strings.Split(arg, "/")
	if len(intervals) > 1 {
		return parseIntervals(intervals, min, max)
	}

	exact, err := parseIntegral(arg, min, max)
	if err != nil {
		return nil, fmt.Errorf("could not parse exact value: %w", err)
	}

	return []uint8{exact}, nil
}

func parseIntegral(s string, min, max uint8) (uint8, error) {
	v, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("could not parse value: %w", err)
	}

	if uint8(v) < min || uint8(v) > max {
		return 0, fmt.Errorf("value must be in range [%d; %d]", min, max)
	}

	return uint8(v), nil
}

func parseRange(rnge []string, min, max uint8) ([]uint8, error) {
	if len(rnge) != 2 {
		return nil, fmt.Errorf("expected range but got %+v", rnge)
	}

	first, err := parseIntegral(rnge[0], min, max)
	if err != nil {
		return nil, fmt.Errorf("invalid begin of range %q: %w", rnge[0], err)
	}

	end, err := parseIntegral(rnge[1], min, max)
	if err != nil {
		return nil, fmt.Errorf("invalid end of range %q: %w", rnge[1], err)
	}

	if first >= end {
		return nil, fmt.Errorf("invalid range [%d; %d]", first, end)
	}

	result := make([]uint8, 0, end-first)
	for i := first; i <= end; i++ {
		result = append(result, i)
	}

	return result, nil
}

func parseSteps(steps []string, min, max uint8) ([]uint8, error) {
	result := make([]uint8, 0, len(steps))

	unique := make(map[uint8]struct{}, len(steps))
	for _, s := range steps {
		if s == "" {
			return nil, errors.New("expected step but got an empty string")
		}

		step, err := parseIntegral(s, min, max)
		if err != nil {
			return nil, fmt.Errorf("invalid step %q: %w", s, err)
		}

		if _, ok := unique[step]; !ok {
			unique[step] = struct{}{}
			result = append(result, step)
		}
	}

	return result, nil
}

func parseIntervals(intervals []string, min, max uint8) ([]uint8, error) {
	if len(intervals) != 2 {
		return nil, errors.New("malformed intervals arg")
	}

	start := uint8(0)
	if intervals[0] != "*" {
		s, err := parseIntegral(intervals[0], min, max)
		if err != nil {
			return nil, fmt.Errorf("could not parse starting point: %w", err)
		}
		start = s
	}

	every, err := parseIntegral(intervals[1], min, max)
	if err != nil {
		return nil, fmt.Errorf("could not parse repeated interval: %w", err)
	}

	if every == 0 {
		return nil, errors.New("repeated interval must be greater than 0")
	}

	if (max+1)%every != 0 {
		return nil, fmt.Errorf("invalid repeated interval: %d for %d", every, max+1)
	}

	result := make([]uint8, 0, (max+1)/every)
	for i := start + min; i < max; i += every {
		result = append(result, i)
	}

	return result, nil
}
