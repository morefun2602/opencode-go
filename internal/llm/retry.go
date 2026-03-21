package llm

import (
	"errors"
	"math"
	"time"
)

const (
	retryBaseDelay = 1 * time.Second
	retryFactor    = 2.0
	retryMaxDelay  = 30 * time.Second
)

func RetryDelay(attempt int, err error) time.Duration {
	var re *RetryableError
	if errors.As(err, &re) && re.RetryAfter > 0 {
		d := time.Duration(re.RetryAfter) * time.Second
		if d > retryMaxDelay {
			d = retryMaxDelay
		}
		return d
	}

	d := time.Duration(float64(retryBaseDelay) * math.Pow(retryFactor, float64(attempt)))
	if d > retryMaxDelay {
		d = retryMaxDelay
	}
	return d
}
