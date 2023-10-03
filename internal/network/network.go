package network

import "os"

type Network interface {
	SupportsOffload() bool
	Tap() *os.File
	Close() error
}
