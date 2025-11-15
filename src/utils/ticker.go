// Package utils: some utils
package utils

import "time"

// NOTE: Ticker

func Ticker(
	fn func(),
	interval time.Duration,
) (stopCh chan struct{}) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	stopCh = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				fn()
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	return stopCh
}
