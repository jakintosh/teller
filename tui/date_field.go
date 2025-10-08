package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// setTime initializes the date field with a given time value
func (d *dateField) setTime(t time.Time) {
	d.year = t.Year()
	d.month = int(t.Month())
	d.day = t.Day()
	d.segment = dateSegmentYear
	d.buffer = ""
}

// time converts the date field to a time.Time value
func (d dateField) time() time.Time {
	if d.year == 0 || d.month == 0 || d.day == 0 {
		return time.Time{}
	}
	return time.Date(d.year, time.Month(d.month), d.day, 0, 0, 0, 0, time.Local)
}

// display returns a formatted date string with the focused segment highlighted
func (d dateField) display(focused bool) string {
	parts := []string{
		fmt.Sprintf("%04d", d.year),
		fmt.Sprintf("%02d", d.month),
		fmt.Sprintf("%02d", d.day),
	}
	if focused {
		parts[d.segment] = "[" + parts[d.segment] + "]"
	}
	return strings.Join(parts, "-")
}

// segmentLeft moves focus to the previous date segment
func (d *dateField) segmentLeft() {
	d.buffer = ""
	if d.segment > dateSegmentYear {
		d.segment--
	}
}

// segmentRight moves focus to the next date segment
func (d *dateField) segmentRight() {
	d.buffer = ""
	if d.segment < dateSegmentDay {
		d.segment++
	}
}

// increment adjusts the focused date segment by the given delta
func (d *dateField) increment(delta int) {
	switch d.segment {
	case dateSegmentYear:
		d.year += delta
	case dateSegmentMonth:
		d.month += delta
		if d.month < 1 {
			d.month = 12
			d.year--
		} else if d.month > 12 {
			d.month = 1
			d.year++
		}
	case dateSegmentDay:
		t := d.time()
		if t.IsZero() {
			t = time.Now()
		}
		t = t.AddDate(0, 0, delta)
		d.year = t.Year()
		d.month = int(t.Month())
		d.day = t.Day()
	}
	d.ensureDayInMonth()
}

// handleDigit processes a numeric input for the focused date segment
func (d *dateField) handleDigit(r rune) {
	if r < '0' || r > '9' {
		return
	}
	d.buffer += string(r)
	switch d.segment {
	case dateSegmentYear:
		if len(d.buffer) > 4 {
			d.buffer = d.buffer[len(d.buffer)-4:]
		}
		if val, err := strconv.Atoi(d.buffer); err == nil {
			d.year = val
		}
	case dateSegmentMonth:
		if len(d.buffer) > 2 {
			d.buffer = d.buffer[len(d.buffer)-2:]
		}
		if val, err := strconv.Atoi(d.buffer); err == nil {
			if val < 1 {
				val = 1
			}
			if val > 12 {
				val = 12
			}
			d.month = val
		}
		if len(d.buffer) >= 2 {
			d.segmentRight()
		}
	case dateSegmentDay:
		if len(d.buffer) > 2 {
			d.buffer = d.buffer[len(d.buffer)-2:]
		}
		if val, err := strconv.Atoi(d.buffer); err == nil {
			maxDay := daysInMonth(d.year, d.month)
			if val < 1 {
				val = 1
			}
			if val > maxDay {
				val = maxDay
			}
			d.day = val
		}
		if len(d.buffer) >= 2 {
			d.segmentRight()
		}
	}
	d.ensureDayInMonth()
}

// ensureDayInMonth validates and clamps the day value to the valid range for the current month
func (d *dateField) ensureDayInMonth() {
	maxDay := daysInMonth(d.year, d.month)
	if d.day > maxDay {
		d.day = maxDay
	}
	if d.day < 1 {
		d.day = 1
	}
}

// daysInMonth returns the number of days in the given month and year
func daysInMonth(year, month int) int {
	if month < 1 || month > 12 {
		return 31
	}
	t := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return t.AddDate(0, 1, -1).Day()
}

// handleDateKey processes keyboard input for the date field
// Returns true if the key was handled, false otherwise
func (m *Model) handleDateKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "left":
		m.form.date.segmentLeft()
		return true
	case "right":
		m.form.date.segmentRight()
		return true
	case "up":
		m.form.date.increment(1)
		return true
	case "down":
		m.form.date.increment(-1)
		return true
	}
	if len(msg.Runes) == 1 {
		r := msg.Runes[0]
		if r >= '0' && r <= '9' {
			m.form.date.handleDigit(r)
			return true
		}
	}
	return false
}
