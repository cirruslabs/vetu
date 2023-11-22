package chunker

import (
	"bytes"
	"crypto/sha256"
	"github.com/opencontainers/go-digest"
	"gvisor.dev/gvisor/pkg/sync"
	"hash"
	"io"
)

// InitializeWriterFunc should initialize a new io.WriteCloser on each invocation,
// which will apply transformations (e.g. data compression) to it's input and then
// write the output to the outputWriter.
type InitializeWriterFunc func(outputWriter io.Writer) (io.WriteCloser, error)

type Chunker struct {
	// Settings
	chunkSize        int
	initializeWriter InitializeWriterFunc

	// State
	chunks  chan *Chunk
	emitted bool

	// Per-chunk state
	buf              *bytes.Buffer
	uncompressedSize int64
	uncompressedHash hash.Hash
	writer           io.WriteCloser

	// State protection
	mtx sync.Mutex
}

type Chunk struct {
	Data               []byte
	UncompressedSize   int64
	UncompressedDigest digest.Digest
}

func NewChunker(chunkSize int, initializeWriter InitializeWriterFunc) (*Chunker, error) {
	chunker := &Chunker{
		// Settings
		chunkSize:        chunkSize,
		initializeWriter: initializeWriter,

		// State
		chunks: make(chan *Chunk),
	}

	if err := chunker.resetPerChunkState(); err != nil {
		return nil, err
	}

	return chunker, nil
}

func (chunker *Chunker) Write(b []byte) (int, error) {
	chunker.mtx.Lock()
	defer chunker.mtx.Unlock()

	// Have we reached the target chunk size?
	if chunker.buf.Len() >= chunker.chunkSize {
		// We need to Close() the chunker.writer first before emitting a chunk,
		// otherwise the un-flushed state in the chunker.writer that it should
		// write to chunker.buf might be lost
		if err := chunker.writer.Close(); err != nil {
			return 0, err
		}

		// Emit a new chunk, blocking any new Write()'s
		// to prevent memory starvation
		chunker.chunks <- &Chunk{
			Data:               chunker.buf.Bytes(),
			UncompressedSize:   chunker.uncompressedSize,
			UncompressedDigest: digest.NewDigest(digest.SHA256, chunker.uncompressedHash),
		}
		chunker.emitted = true

		if err := chunker.resetPerChunkState(); err != nil {
			return 0, err
		}
	}

	// Update uncompressed chunk size
	chunker.uncompressedSize += int64(len(b))

	// Update uncompressed chunk hash
	n, err := chunker.uncompressedHash.Write(b)
	if err != nil {
		return n, err
	}

	return chunker.writer.Write(b)
}

func (chunker *Chunker) Chunks() chan *Chunk {
	return chunker.chunks
}

func (chunker *Chunker) Close() error {
	chunker.mtx.Lock()
	defer chunker.mtx.Unlock()

	// We need to Close() the chunker.writer first before emitting a chunk,
	// otherwise the un-flushed state in the chunker.writer that it should
	// write to chunker.buf might be lost
	if err := chunker.writer.Close(); err != nil {
		return err
	}

	// Only emit a last chunk if we have some data available
	// or there were no chunks emitted before
	if chunker.buf.Len() != 0 || !chunker.emitted {
		chunker.chunks <- &Chunk{
			Data:               chunker.buf.Bytes(),
			UncompressedSize:   chunker.uncompressedSize,
			UncompressedDigest: digest.NewDigest(digest.SHA256, chunker.uncompressedHash),
		}
	}

	close(chunker.chunks)

	return chunker.resetPerChunkState()
}

func (chunker *Chunker) resetPerChunkState() error {
	chunker.buf = &bytes.Buffer{}
	chunker.uncompressedSize = 0
	chunker.uncompressedHash = sha256.New()

	writer, err := chunker.initializeWriter(chunker.buf)
	if err != nil {
		return err
	}
	chunker.writer = writer

	return nil
}
