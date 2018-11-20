package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/warent/proxycop/utility"
)

func main() {

	utility.InitializeDB()
	defer utility.CloseDB()

	go startProxy()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-sig:
			log.Fatalf("Signal (%d) received, stopping\n", s)
		}
	}
}
