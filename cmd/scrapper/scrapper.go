package main

import (
	"log"
	"os"
	"temperature/internal/scrapper"
)

func main() {
	config := scrapper.Config{}
	err := config.Init()
	if err != nil {
		log.Fatalln(err)
	}
	s := scrapper.New(&config)
	go s.Run()

	exit := make(chan int)
	os.Exit(<-exit)
}
