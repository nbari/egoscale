package egoscale

import (
	"testing"
	"time"
)

func TestFibonacciRetryStrategy(t *testing.T) {
	var i int64

	if FibonacciRetryStrategy(i) != time.Duration(0) {
		t.Error("fib(0) = 0")
	}

	i++
	if FibonacciRetryStrategy(i) != time.Second {
		t.Error("fib(1) = 1")
	}

	i++
	if FibonacciRetryStrategy(i) != time.Second {
		t.Error("fib(2) = 1")
	}

	i++
	to := FibonacciRetryStrategy(i)
	if to != time.Duration(2)*time.Second {
		t.Errorf("fib(3) = 2 != %v", to)
	}

	i += 20
	to = FibonacciRetryStrategy(i)
	if to != time.Duration(28657)*time.Second {
		t.Errorf("fib(23) = 7h57m37s != %v", to)
	}
}
