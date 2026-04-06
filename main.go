package main

import (
	"fmt"
	"net/http"
)

func main() {
	servermux := http.NewServeMux()
	server := &http.Server{
		Handler: servermux,
		Addr: ":8080",
	}
	err := server.ListenAndServe()
	fmt.Println(err)
}