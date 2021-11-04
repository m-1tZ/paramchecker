package main

import (
	"flag"
	"log"
	"net/http"
	"sync"
)

var (
	listen *string = flag.String("listen", "127.0.0.1:8888", "Listen host:port")
)

var remoteAddrs = make(map[string]bool)
var totConnections int
var lock sync.Mutex

func handler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	remoteAddrs[r.RemoteAddr] = true
	totConnections++
	log.Printf("%d / %d", totConnections, len(remoteAddrs))
	lock.Unlock()

	w.Write([]byte("foo\n"))
}

func main() {
	flag.Parse()

	http.HandleFunc("/", handler)
	http.ListenAndServe(*listen, nil)
}
