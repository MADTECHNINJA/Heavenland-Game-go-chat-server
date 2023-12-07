package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var addr = flag.String("chatserver", "0.0.0.0:8080", "http service address")

func loadEnv() string {
	if env, ok := os.LookupEnv("env"); ok {
		return env
	}
	return "dev"
}

func main() {
	// create and run server
	flag.Parse()
	server := newServer()
	go server.run()

	// load environment configs
	env_file := loadEnv()
	fmt.Println(env_file)
	env_err := godotenv.Load("." + env_file)
	if env_err != nil {
		log.Fatal("Error loading ." + env_file + " file")
	}
	fmt.Println(os.Getenv("HEAVENLAND_URL"))

	// listen on the /chat for incoming websocket connections
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		serveWs(server, w, r)
	})

	// in case of error log it
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
