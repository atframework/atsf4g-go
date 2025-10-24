// client.go
package main

import (
	"os"
	"os/signal"
)

var interrupt chan os.Signal

func main() {
	interrupt = make(chan os.Signal)       // Channel to listen for interrupt signal to terminate gracefully
	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	ReadLine()
}
