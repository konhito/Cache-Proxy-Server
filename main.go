package mainAdd commentMore actions

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	originServer string
	port         string
)

func main() {
	flag.StringVar(&port, "port", "8080", "Port on which the proxy server will run")
	flag.StringVar(&originServer, "origin", "", "URL of the origin server")
	flag.Parse()

	if originServer == "" {
		fmt.Println("Error: --origin flag is required")
		os.Exit(1)
	}

	fmt.Printf("Caching proxy server running on port %s, forwarding to %s\n", port, originServer)
	http.HandleFunc("/", handleRequest)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	targetURL := originServer + r.URL.Path


	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, "Error contacting origin server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()


	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	
	w.WriteHeader(resp.StatusCode)

	
	io.Copy(w, resp.Body)
}
