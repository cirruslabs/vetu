package network

import "os"

type Network interface {
	Tap() *os.File
	Close() error
}
