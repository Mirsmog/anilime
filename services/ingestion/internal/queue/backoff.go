package queue

import "time"

func backoffDelay(numDelivered uint64) time.Duration {
	// 1st failure -> 1s, 2nd -> 2s, 3rd -> 4s ... capped
	attempt := int(numDelivered)
	if attempt < 1 {
		attempt = 1
	}
	sec := 1 << (attempt - 1)
	if sec > 60 {
		sec = 60
	}
	return time.Duration(sec) * time.Second
}
