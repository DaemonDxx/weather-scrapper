package main

import (
	"flag"
	"os"
	"temperature/internal/scrapper"
)

func main() {
	configPath := flag.String("f", "./configs/default.yml", "Config file path")
	flag.Parse()

	config := scrapper.Config{}
	err := config.LoadFromFile(*configPath)
	if err != nil {
		panic(err)
	}
	s := scrapper.New(&config)
	go s.Run()

	exit := make(chan int)
	os.Exit(<-exit)
}
