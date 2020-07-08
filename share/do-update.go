package share

import (
	"sync/atomic"
	"time"
)

func DoUpdate(period time.Duration, f func()) {
	go func() {
		var lock int32
		var ticker *time.Ticker

		for {
			go func() {
				if atomic.SwapInt32(&lock, 1) != 0 {
					return
				}
				defer atomic.StoreInt32(&lock, 0)

				f()
			}()

			if ticker == nil {
				<-time.After(time.Until(time.Now().Truncate(period).Add(period)))
				ticker = time.NewTicker(period)
			} else {
				<-ticker.C
			}
		}
	}()
}
