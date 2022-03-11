package utils

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestTimedTask(t *testing.T) {
	date := time.Date(2021, 12, 30, 14, 58, 0, 0, time.Local)
	TimedTask(date, func() {
		fmt.Println("aaa")
	})
	time.Sleep(math.MaxInt64)
}
