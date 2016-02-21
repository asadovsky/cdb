package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/asadovsky/cdb/server/hub"
)

var (
	port      = flag.Int("port", 0, "")
	peerAddrs = flag.String("peer-addrs", "", "comma-separated peer addrs")
)

func main() {
	flag.Parse()
	addr := fmt.Sprintf("localhost:%d", *port)
	if err := hub.Serve(addr, strings.Split(*peerAddrs, ",")); err != nil {
		log.Fatal(err)
	}
}
