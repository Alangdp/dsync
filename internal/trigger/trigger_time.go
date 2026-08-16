package trigger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Alangdp/dsync.git/internal/clock"
)

func ConvertTime(timeString string) (*clock.Time, error) {
	timeSliceString := strings.Split(timeString, ":")
	timeSliceLength := len(timeSliceString)

	// Se recebeu em um formato inválido
	if timeSliceLength != 3 && timeSliceLength != 4 {
		return nil, fmt.Errorf("invalid time format: %q (expected HH:MM:SS:CC)", timeString)
	}

	// Verifica se recebeu números e são positivos
	timeSlice := make([]int64, 4)
	for i, slice := range timeSliceString {
		sliceInt, err := strconv.ParseInt(slice, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("time value is not numeric: %q", timeString)
		}
		if sliceInt < 0 {
			return nil, fmt.Errorf("time value contains negative numbers: %q", timeString)
		}
		timeSlice[i] = sliceInt
	}

	// Verifica se nenhum valor ultrapassa o valor máximo
	if timeSlice[0] > 23 || timeSlice[1] > 59 || timeSlice[2] > 59 || timeSlice[3] > 99 {
		return nil, fmt.Errorf("time value is out of the valid range: %q", timeString)
	}

	t := clock.Time{
		Hours:        int8(timeSlice[0]),
		Minutes:      int8(timeSlice[1]),
		Seconds:      int8(timeSlice[2]),
		CentiSeconds: int8(timeSlice[3]),
	}
	return &t, nil
}
