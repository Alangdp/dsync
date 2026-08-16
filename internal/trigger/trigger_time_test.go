package trigger

import (
	"testing"
)

func TestTriggerTime(t *testing.T) {
	assertCorrectTime := func(t *testing.T, timeString string, hours, minutes, seconds, centiSeconds int8) {
		t.Helper()

		result, err := ConvertTime(timeString)
		if err != nil {
			t.Fatal(err)
		}

		// Se o horário esperado não é igual ao recebido
		if result.Hours != hours ||
			result.Minutes != minutes ||
			result.Seconds != seconds ||
			result.CentiSeconds != centiSeconds {
			t.Errorf("Error in time formatting routine. Expected: [%d, %d, %d, %d], Got: [%d, %d, %d, %d]", hours, minutes, seconds, centiSeconds, result.Hours, result.Minutes, result.Seconds, result.CentiSeconds)
		}

	}

	assertError := func(t *testing.T, input string) {
		t.Helper()

		_, err := ConvertTime(input)
		if err == nil {
			t.Errorf("expected error for %q", input)
		}
	}

	validCases := []struct {
		input                                 string
		hours, minutes, seconds, centiSeconds int8
	}{
		{"12:00:00", 12, 0, 0, 0},
		{"15:00:00", 15, 0, 0, 0},
		{"12:30:00", 12, 30, 0, 0},
		{"12:30:15", 12, 30, 15, 0},
		{"12:30:15:50", 12, 30, 15, 50},
	}
	for _, c := range validCases {
		t.Run("valid_"+c.input, func(t *testing.T) {
			assertCorrectTime(t, c.input, c.hours, c.minutes, c.seconds, c.centiSeconds)
		})
	}

	invalidCases := []string{
		"12::15", ":30:15", "12:30:", "12:30:15:00:00:00",
		"12:70:15", "12:30", "", "12:60:15", "25:00:15",
		"12:30:200", "-1:30:00",
	}
	for _, input := range invalidCases {
		t.Run("invalid_"+input, func(t *testing.T) {
			assertError(t, input)
		})
	}
}
