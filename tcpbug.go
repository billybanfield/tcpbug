package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
)

var server http.Server
var serverMu *sync.Mutex

var connCount int
var activeConns int
var maxActiveConns int

func main() {
	runTCPTester()
}

func runTCPTester() {
	transport := &http.Transport{
		MaxConnsPerHost: 1,
	}
	serverMu = &sync.Mutex{}
	client := &http.Client{}
	client.Transport = transport

	fmt.Println("---")
	fmt.Println("Starting server...")
	fmt.Println("---")
	startServer()

	// Perform concurrent requests against the running server.
	// Reuses the same connection as expected.
	runRequests(client, 5000, 100)

	serverMu.Lock()
	connCountFirstBatch := connCount
	connCount = 0
	maxActiveConnsFirstBatch := maxActiveConns
	maxActiveConns = 0
	serverMu.Unlock()

	fmt.Println("---")
	fmt.Printf("Total connections in first request batch %d\n", connCountFirstBatch)
	fmt.Printf("Max active connections in first request batch %d\n", maxActiveConnsFirstBatch)

	fmt.Println("---")
	fmt.Println("Stopping server...")
	fmt.Println("---")
	stopServer()

	runRequests(client, 5000, 100)

	fmt.Println("---")
	fmt.Println("Restarting server...")
	fmt.Println("---")
	startServer()

	// Perform concurrent requests against the running server.
	// Dials new connections and doesn't respect MaxConnsPerHost.
	runRequests(client, 1000, 100)
	serverMu.Lock()
	connCountSecondBatch := connCount
	maxActiveConnsSecondBatch := maxActiveConns
	serverMu.Unlock()
	fmt.Printf("Total connections in second request batch %d\n", connCountSecondBatch)
	fmt.Printf("Max active connections in second request batch %d\n", maxActiveConnsSecondBatch)

	fmt.Println("---")
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func startServer() {
	h := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
	}
	cs := func(conn net.Conn, state http.ConnState) {
		serverMu.Lock()
		defer serverMu.Unlock()
		switch state {
		case http.StateNew:
			connCount++
			activeConns++
			if activeConns > maxActiveConns {
				maxActiveConns = activeConns
			}
		case http.StateClosed:
			activeConns--
		}
	}
	server = http.Server{
		Addr:      "127.0.0.1:8080",
		Handler:   http.HandlerFunc(h),
		ConnState: cs,
	}
	go server.ListenAndServe()
}

func stopServer() {
	server.Close()
}

func runRequests(client *http.Client, numRequests, numWorkers int) {
	ch := make(chan struct{})
	wg := sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			for range ch {
				doRequest(client)
			}
			wg.Done()
		}()
	}
	for i := 0; i < numRequests; i++ {
		ch <- struct{}{}
	}
	close(ch)
	wg.Wait()
}

func doRequest(client *http.Client) error {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	checkError(err)
	_, err = client.Do(req)
	return err
}
