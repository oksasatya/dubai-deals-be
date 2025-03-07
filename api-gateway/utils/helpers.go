package utils

import (
	"strconv"
)

func StringToInt(value string) int {
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return i
}
