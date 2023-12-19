package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

func main() {
	var port int
	flag.IntVar(&port,"port", 8080, "port number")
	flag.Parse()

	fmt.Println(fmt.Sprintf("Running http server on port %v...", port))
	http.HandleFunc("/hello", hello)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil); err != nil {
		os.Exit(1)
	}
}
