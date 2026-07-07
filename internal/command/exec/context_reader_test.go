package exec

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContextReaderNormalRead(t *testing.T) {
	r := newContextReader(bytes.NewReader([]byte("hello")))
	buf := make([]byte, 64)

	n, err := r.Read(context.Background(), buf)
	require.NoError(t, err)
	require.Equal(t, "hello", string(buf[:n]))
}

func TestContextReaderEOF(t *testing.T) {
	r := newContextReader(bytes.NewReader([]byte("hi")))
	buf := make([]byte, 64)

	// First read returns data
	n, err := r.Read(context.Background(), buf)
	require.NoError(t, err)
	require.Equal(t, "hi", string(buf[:n]))

	// Second read returns EOF
	_, err = r.Read(context.Background(), buf)
	require.ErrorIs(t, err, io.EOF)
}

func TestContextReaderCancellation(t *testing.T) {
	// Use a reader that blocks forever
	r, _ := io.Pipe()
	cr := newContextReader(r)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	buf := make([]byte, 64)
	_, err := cr.Read(ctx, buf)
	require.ErrorIs(t, err, context.Canceled)
}

func TestContextReaderCancellationUnblocksRead(t *testing.T) {
	// Use a reader that blocks forever
	r, _ := io.Pipe()
	cr := newContextReader(r)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	buf := make([]byte, 64)
	_, err := cr.Read(ctx, buf)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
