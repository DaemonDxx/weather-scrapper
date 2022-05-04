package main

import (
	"flag"
	"os"
	"temperature/internal/app"
)

func main() {
	configPath := flag.String("f", "./configs/default.yml", "Config file path")
	flag.Parse()

	config := app.Config{}
	err := config.LoadFromFile(*configPath)
	if err != nil {
		panic(err)
	}
	signal := app.Run(&config)
	os.Exit(<-signal)
}
