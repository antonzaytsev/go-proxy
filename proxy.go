package main

import (
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

type SimpleResponse struct {
	Header     map[string][]string
	StatusCode int
	Body       string
}

var customTransport = http.DefaultTransport
var cacheStorage = cache.New(60*time.Minute, 120*time.Minute)

func main() {
	// Create a new HTTP server with the handleRequest function as the handler
	server := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(handleRequest),
	}

	// Start the server and log any errors
	log.Println("Starting proxy server on :8080")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting proxy server: ", err)
	}
}

func init() {
	log.Println("init function executed")
	// Here, you can customize the transport, e.g., set timeouts or enable/disable keep-alive
}

// Route request
func handleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.String()[1:]
	urlPathElements := strings.SplitN(urlPath, "/", 2)

	if urlPathElements[0] == "go" {
		log.Printf("Processing request to /go, %v\n", urlPathElements[1])
		handleProxyRequest(w, r, urlPathElements[1])
	} else {
		// For now support old approach
		handleProxyRequest(w, r, urlPath)
	}
}

func handleProxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	cachedResponse, found := cacheStorage.Get(targetURL)

	var simplifiedResponse SimpleResponse

	if found {
		simplifiedResponse, _ = cachedResponse.(SimpleResponse)
	} else {
		log.Println("Cache miss")
		// Build and send http proxy request
		resp, err := buildAndSendRequest(w, r, targetURL)
		if err != nil {
			return
		}

		simplifiedResponse = transformResponse(resp)

		cacheStorage.Set(targetURL, simplifiedResponse, cache.DefaultExpiration)
	}

	log.Println("simplifiedResponse", simplifiedResponse)

	copyProxyResponse(w, &simplifiedResponse)
}

func transformResponse(resp *http.Response) SimpleResponse {
	simplifiedResponse := SimpleResponse{}

	simplifiedResponse.Header = resp.Header

	log.Printf("statusCode: %v", resp.StatusCode)
	// Set the status code of the original response to the status code of the proxy response
	simplifiedResponse.StatusCode = resp.StatusCode

	// Ensure the body is closed after reading
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Couldn't close body")
		}
	}(resp.Body)

	// Copy the body of the proxy response to the original response
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Fatalf("Failed to read response body: %v", err)
		log.Printf("Failed to read response body: %v", err)
	}

	log.Printf("bytes type: %v, len: %v, content: %v, string: %v", reflect.TypeOf(bytes), len(bytes), bytes, string(bytes))
	simplifiedResponse.Body = string(bytes)

	log.Println("simplifiedResponse", simplifiedResponse)

	return simplifiedResponse
}

func buildAndSendRequest(w http.ResponseWriter, r *http.Request, targetURL string) (*http.Response, error) {
	// Create a new HTTP request with the same method, URL, and body as the original request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)

	if err != nil {
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return nil, err
	}

	// Copy the headers from the original request to the proxy request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Send the proxy request using the custom transport
	resp, err := customTransport.RoundTrip(proxyReq)
	if err != nil {
		log.Println(err)
		log.Println(targetURL)
		http.Error(w, "Error sending proxy request", http.StatusInternalServerError)
		return nil, err
	}

	return resp, nil
}

func copyProxyResponse(w http.ResponseWriter, simplifiedResponse *SimpleResponse) {
	// Copy the headers from the proxy response to the original response
	for name, values := range simplifiedResponse.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(simplifiedResponse.StatusCode)

	// Copy the body of the proxy response to the original response
	_, err := io.Copy(w, strings.NewReader(simplifiedResponse.Body))
	if err != nil {
		log.Fatalln("Error copying response body")
		return
	}
}
