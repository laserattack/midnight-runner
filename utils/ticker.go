// Package utils: some utils
package utils

import "time"

//  NOTE: Ticker
//  TODO: Написать тесты для тикера
// Run func at the specified interval in a separate goroutine
// Returns the channel for stopping the ticker

func Ticker(fn func(), interval time.Duration) chan struct{} {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	stopChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				fn()
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	return stopChan
}
