# CDB

CRDT database design exploration.

## Demo instructions

Fetch and build the code:

    GOPATH=~/dev/go
    mkdir -p ${GOPATH}/src/github.com/asadovsky
    cd ${GOPATH}/src/github.com/asadovsky
    git clone --recursive git@github.com:asadovsky/cdb.git
    cd cdb
    make build

Run two instances on one machine:

    # Run these commands in two separate terminals, and open the printed URLs.
    dist/demo -port=4001 -peer-addrs=localhost:4002
    dist/demo -port=4002 -peer-addrs=localhost:4001

Or, run instances on two different machines:

    # Run this command on Alice's machine.
    dist/demo -port 4001 -loopback=false

    # Run this command on Bob's machine, setting the -peer-addrs flag to Alice's
    # network address.
    dist/demo -port 4001 -loopback=false -peer-addrs=192.168.1.239:4001
