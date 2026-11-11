package config

import (
	"reflect"
	"testing"
	"time"
)

func TestPositiveIntEnv(t *testing.T) {
	cases := []struct {
		value string
		want  int
	}{{"", 6}, {"9", 9}, {" 12 ", 12}, {"0", 6}, {"-3", 6}, {"abc", 6}}
	for _, c := range cases {
		t.Setenv("TEST_POSITIVE_INT", c.value)
		if got := positiveIntEnv("TEST_POSITIVE_INT", 6); got != c.want {
			t.Errorf("positiveIntEnv(%q) = %d, want %d", c.value, got, c.want)
		}
	}
}

func TestDurationEnv(t *testing.T) {
	cases := []struct {
		value string
		want  time.Duration
	}{{"", 15 * time.Minute}, {"5m", 5 * time.Minute}, {"90s", 90 * time.Second}, {"0s", 0}, {"-1m", 15 * time.Minute}, {"soon", 15 * time.Minute}}
	for _, c := range cases {
		t.Setenv("TEST_DURATION", c.value)
		if got := durationEnv("TEST_DURATION", 15*time.Minute); got != c.want {
			t.Errorf("durationEnv(%q) = %v, want %v", c.value, got, c.want)
		}
	}
}

func TestParseList(t *testing.T) {
	if got := parseList(" Done, ,Fixed ", nil); !reflect.DeepEqual(got, []string{"Done", "Fixed"}) {
		t.Errorf("parseList = %v, want [Done Fixed]", got)
	}
	if got := parseList("", []string{"x"}); !reflect.DeepEqual(got, []string{"x"}) {
		t.Errorf("parseList(empty) = %v, want the default", got)
	}
	if got := parseList(" , ", nil); got != nil {
		t.Errorf("parseList(blanks) = %v, want nil", got)
	}
}
