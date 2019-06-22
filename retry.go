package main

import (
	"math"
	"time"
)

const (
	maxRetries = 5
	retryDelay = 1.2
)

func retry(f func() error) error {
	var err error

	for i := 1; i <= maxRetries; i++ {
		if err = f(); err == nil {
			return nil
		}

		sleep := time.Duration(math.Pow(retryDelay, float64(i)) * float64(time.Second))
		time.Sleep(sleep)
	}

	return err
}
