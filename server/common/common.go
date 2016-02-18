package common

import (
	"strconv"
)

func Atoi(s string) (uint32, error) {
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(i), nil
}

func Itoa(i uint32) string {
	return strconv.FormatUint(uint64(i), 10)
}
