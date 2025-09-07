package main

import (
 "flag"
 "fmt"
 "io"
 "log"
 "net/http"
 "os"
 "time"

 "github.com/avii09/proxy_server/cache"
)

var (
 originServer string
 port         string
)

func main() {
 // Parse command-line arguments.
 // This is done to dynamically set the port and origin server URL, instead of hardcoding them.

 // user will start server => go run server/main.go --port <port_no> --origin <origin_server_url>
 flag.StringVar(&port, "port", "8080", "Port on which the proxy server will run")
 flag.StringVar(&originServer, "origin", "", "URL of the origin server")
 flag.Parse()

 // Check if the --origin flag is provided
 if originServer == "" {
  fmt.Println("Error: --origin flag is required")
  os.Exit(1)
 }

 // initialize the cache
 cache.InitRedis()

 // Start the proxy server
 fmt.Printf("Caching proxy server running on port %s, forwarding to %s\n", port, originServer)
 http.HandleFunc("/", handleRequest)

 log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleRequest will forward the incoming req to the origin server and return the response
func handleRequest(w http.ResponseWriter, r *http.Request) {
 // construct the target URL
 targetURL := originServer + r.URL.Path

 // try to get cached response
 cachedResponse, err := cache.GetClient().Get(cache.Ctx, targetURL).Result()
 if err == nil {
  fmt.Println("Cache HIT")
  w.Write([]byte(cachedResponse)) // send cached response to client
  return
 }

 fmt.Println("Cache MISS")
 // forward request to the origin server
 resp, err := http.Get(targetURL)
 if err != nil {
  http.Error(w, "Error contacting origin server", http.StatusBadGateway)
  return
 }
 defer resp.Body.Close()

 // copy response headers
 for key, values := range resp.Header {
  for _, value := range values {
   w.Header().Add(key, value)
  }
 }

 // copy response status code
 w.WriteHeader(resp.StatusCode)

 // read response body
 body, _ := io.ReadAll(resp.Body)

 // store response in Redis cache
 cache.GetClient().Set(cache.Ctx, targetURL, body, 300*time.Second) // cache for 5 mins

 // send response to client
 w.Write(body)
}
