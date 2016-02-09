package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/asadovsky/cdb/server/hub"
)

var port = flag.Int("port", 0, "")

func main() {
	flag.Parse()
	addr := fmt.Sprintf("localhost:%d", *port)
	if err := hub.Serve(addr); err != nil {
		log.Fatal(err)
	}
}
