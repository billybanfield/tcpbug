package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"
)

var server http.Server
var connCount int32
var dialCompletedCount int32
var dialCount int32

func main() {
	fmt.Println("---")
	fmt.Println("Starting server...")
	fmt.Println("---")
	startServer()
	ctx, cancelFunc := context.WithCancel(context.Background())
	client, trace := annotatedClientAndTrace()

	runRequests(client, trace, ctx, 10, 1)

	fmt.Println("---")
	fmt.Println("Stopping server...")
	fmt.Println("---")
	stopServer()

	runRequests(client, trace, ctx, 10, 2)

	ctx, cancelFunc = context.WithCancel(context.Background())
	fmt.Println("---")
	fmt.Println("Restarting server...")
	fmt.Println("---")
	startServer()
	runRequests(client, trace, ctx, 100, 2)
	cancelFunc()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func doRequest(client *http.Client, trace *httptrace.ClientTrace) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	checkError(err)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	client.Do(req)
}

func runRequests(client *http.Client, trace *httptrace.ClientTrace, ctx context.Context, numRequests, numWorkers int) {
	ch := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			for range ch {
				select {
				case <-ctx.Done():
					return
				default:
					doRequest(client, trace)
				}
			}
		}()
		wg.Done()
	}
	for i := 0; i < numRequests; i++ {
		ch <- struct{}{}
	}
	close(ch)
	wg.Wait()
}

func annotatedClientAndTrace() (*http.Client, *httptrace.ClientTrace) {
	dc := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext

	wrappedDialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
		totalDials := atomic.AddInt32(&dialCount, 1)
		fmt.Printf("Starting dial number %v\n", totalDials)
		conn, err := dc(ctx, network, address)
		if err == nil {
			curr := atomic.AddInt32(&dialCompletedCount, 1)
			fmt.Printf("Dial Done with no error: %v %v total dials: %v dials completed: %v\n", network, address, totalDials, curr)
		} else {
			curr := atomic.LoadInt32(&dialCompletedCount)
			fmt.Printf("Dial Done with error: %v %v err:%v, total dials: %v, dials completed: %v\n", network, address, err, totalDials, curr)
		}
		return conn, err
	}

	transport := &http.Transport{
		MaxConnsPerHost: 1,
		MaxIdleConns:    1,
		DialContext:     wrappedDialContext,
	}
	client := &http.Client{}
	client.Transport = transport

	trace := &httptrace.ClientTrace{
		ConnectStart: func(network, addr string) {
			fmt.Printf("Connect start: %v %v \n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				curr := atomic.AddInt32(&connCount, 1)
				fmt.Printf("Connect Done with no error: %v %v total conns: %v\n", network, addr, curr)
			} else {
				fmt.Printf("Connect Done error: %v %v err: %v\n", network, addr, err)
			}
		},
		GetConn: func(hostPort string) {
			fmt.Printf("Getting connection %v\n", hostPort)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			if !connInfo.Reused {
				fmt.Printf("Got New Conn \n")
			}
		},
	}

	return client, trace
}
func startServer() {
	h := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
	}
	server = http.Server{Addr: "127.0.0.1:8080", Handler: http.HandlerFunc(h)}
	go func() {
		server.ListenAndServe()
	}()
}

func stopServer() {
	server.Close()
}
