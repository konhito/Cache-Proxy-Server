# Caching Proxy Server in Go

This project demonstrates how to build a **Caching Proxy Server** using **Go** programming language. The proxy server acts as an intermediary between clients and a backend server, caching responses to improve performance and reduce load on the backend.

---

## Introduction

A **proxy server** is an intermediary that sits between a client (e.g., a web browser) and a backend server. It forwards client requests to the server and returns the server's responses to the client. A **caching proxy server** enhances this by storing responses for specific requests, allowing subsequent identical requests to be served from the cache, reducing latency and backend server load.

This article walks through building a caching proxy server in Go, with the following features:

- Forwarding HTTP requests to a backend server.
- Caching responses in memory using an LRU (Least Recently Used) cache.
- Handling concurrent requests safely.
- Supporting cache invalidation based on time-to-live (TTL).

---

## Project Architecture

The proxy server will:

1. Accept HTTP requests from clients.
2. Check if the response for a request is available in the cache.
3. If cached, return the cached response.
4. If not cached, forward the request to the backend server, cache the response, and return it to the client.

Below is a diagram illustrating the architecture:

![Proxy Server Architecture](https://miro.medium.com/v2/resize:fit:1100/format:webp/1*2_qWV3VEFWU1VYNBfvoQ9w.png)\---

## Prerequisites

To follow along, you’ll need:

- **Go**: Version 1.16 or higher installed.
- A basic understanding of HTTP and Go programming.
- A backend server to test the proxy (you can use a simple server or a public API like `httpbin.org`).

---

## Implementation

Let’s build the caching proxy server step by step.

### Step 1: Setting Up the Project

Create a new Go project:

```bash
mkdir caching-proxy
cd caching-proxy
go mod init caching-proxy
```

Install dependencies for HTTP handling and LRU cache:

```bash
go get github.com/hashicorp/golang-lru
```

### Step 2: Creating the Cache

We’ll use the `golang-lru` package to implement an LRU cache for storing responses. The cache will store responses with a key (e.g., request URL) and a TTL for expiration.

```go
package main

import (
	"container/list"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	Response []byte
	Expires  time.Time
}

// Cache manages the LRU cache
type Cache struct {
	lru  *lru.Cache
	mu   sync.RWMutex
	size int
	ttl  time.Duration
}

// NewCache creates a new cache with the given size and TTL
func NewCache(size int, ttl time.Duration) (*Cache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &Cache{
		lru:  cache,
		size: size,
		ttl:  ttl,
	}, nil
}

// Set adds a response to the cache
func (c *Cache) Set(key string, response []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Add(key, CacheEntry{
		Response: response,
		Expires:  time.Now().Add(c.ttl),
	})
}

// Get retrieves a response from the cache
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.lru.Get(key)
	if !ok {
		return nil, false
	}

	entry := value.(CacheEntry)
	if time.Now().After(entry.Expires) {
		c.lru.Remove(key)
		return nil, false
	}

	return entry.Response, true
}
```

This code defines a `Cache` struct with an LRU cache and thread-safe operations using a `sync.RWMutex`. Responses are stored with a TTL to ensure they expire after a specified duration.

### Step 3: Building the Proxy Server

The proxy server will:

- Listen for incoming HTTP requests.
- Check the cache for a response.
- Forward requests to the backend server if not cached.
- Cache the response and return it to the client.

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type ProxyServer struct {
	cache       *Cache
	backendURL  string
	client      *http.Client
}

// NewProxyServer creates a new proxy server
func NewProxyServer(backendURL string, cacheSize int, cacheTTL time.Duration) (*ProxyServer, error) {
	cache, err := NewCache(cacheSize, cacheTTL)
	if err != nil {
		return nil, err
	}

	return &ProxyServer{
		cache:      cache,
		backendURL: backendURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// ServeHTTP handles incoming HTTP requests
func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cacheKey := r.URL.String()

	// Check cache
	if response, ok := p.cache.Get(cacheKey); ok {
		w.Write(response)
		log.Printf("Cache hit for %s", cacheKey)
		return
	}

	// Forward request to backend
	backendURL := p.backendURL + r.URL.String()
	req, err := http.NewRequest(r.Method, backendURL, r.Body)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		return
	}

	// Cache the response
	p.cache.Set(cacheKey, body)

	// Write response to client
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
	log.Printf("Cache miss for %s, response cached", cacheKey)
}

func main() {
	proxy, err := NewProxyServer("http://httpbin.org", 100, 5*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting proxy server on :8080")
	if err := http.ListenAndServe(":8080", proxy); err != nil {
		log.Fatal(err)
	}
}
```

This code sets up an HTTP server that:

- Uses the `ProxyServer` struct to handle requests.
- Checks the cache for a response using the request URL as the key.
- Forwards requests to the backend (`httpbin.org` in this example) if not cached.
- Caches the response and sends it to the client.

### Step 4: Testing the Proxy Server

1. Run the proxy server:

```bash
go run main.go
```

2. Send requests to the proxy server using `curl` or a browser:

```bash
curl http://localhost:8080/get
```

3. Observe the logs to see cache hits and misses.

Example output:

```
2023/10/01 12:00:00 Starting proxy server on :8080
2023/10/01 12:00:01 Cache miss for /get, response cached
2023/10/01 12:00:02 Cache hit for /get
```

The first request results in a cache miss, fetching the response from the backend and caching it. Subsequent requests within the TTL (5 minutes) result in cache hits, serving the cached response.

---

## Performance Considerations

- **Cache Size**: The LRU cache has a fixed size (100 in this example). Adjust based on memory constraints.
- **TTL**: Set an appropriate TTL (e.g., 5 minutes) to balance freshness and performance.
- **Concurrency**: The `sync.RWMutex` ensures thread-safe cache access.
- **Error Handling**: The code includes basic error handling; enhance it for production use.

---

## Enhancements

To make the proxy server production-ready, consider:

- Adding cache invalidation endpoints (e.g., clear cache via an API).
- Supporting cache headers (e.g., `Cache-Control`, `ETag`).
- Implementing persistent storage for the cache (e.g., Redis).
- Adding metrics for cache hit/miss rates.

---

## Conclusion

This tutorial demonstrated how to build a caching proxy server in Go using an LRU cache. The server efficiently handles HTTP requests by caching responses, reducing latency, and minimizing backend load. You can extend this project by adding features like cache invalidation, persistent storage, or advanced routing.

Feel free to explore the code and adapt it to your needs. Happy coding!

---

## References

- Go Documentation
- HashiCorp golang-lru

httpbin.org (used as the backend server for testing)
