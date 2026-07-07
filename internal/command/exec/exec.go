package exec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	rpcgrpc "buf.build/gen/go/cirruslabs/tart-guest-agent/grpc/go/rpc/rpcgrpc"
	rpc "buf.build/gen/go/cirruslabs/tart-guest-agent/protocolbuffers/go/rpc"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	interactive bool
	tty         bool
)

type ExecCustomExitCodeError struct {
	ExitCode int32
}

func (e *ExecCustomExitCodeError) Error() string {
	return fmt.Sprintf("command exited with status %d", e.ExitCode)
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] <name> <command> [args...]",
		Short: "Execute a command in a running VM",
		RunE:  runExec,
		Args:  cobra.MinimumNArgs(2),
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Attach host's standard input to a remote command")
	cmd.Flags().BoolVarP(&tty, "tty", "t", false, "Allocate a remote pseudo-terminal (PTY)")

	return cmd
}

func runExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	commandName := args[1]
	commandArgs := args[2:]

	localName, err := localname.NewFromString(name)
	if err != nil {
		return err
	}

	// Open VM's directory
	vmDir, err := local.Open(localName)
	if err != nil {
		return err
	}

	// Ensure that the VM is running
	if !vmDir.Running() {
		return fmt.Errorf("VM %q is not running", name)
	}

	// Change the current working directory to a VM's base directory
	// to work around Unix domain socket 108-character length limit (SUN_LEN)
	if err := os.Chdir(vmDir.Path()); err != nil {
		return fmt.Errorf("failed to change directory to VM path: %w", err)
	}

	// Switch controlling terminal into raw mode when remote pseudo-terminal is requested
	var state *term.State
	stdinFd := int(os.Stdin.Fd())
	if tty && term.IsTerminal(stdinFd) {
		var err error
		state, err = term.MakeRaw(stdinFd)
		if err == nil {
			defer func() {
				// Restore terminal to its initial state
				_ = term.Restore(stdinFd, state)
			}()
		}
	}

	// Execute a command in a running VM
	controlSocketPath := "vsock.sock"
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "unix", controlSocketPath)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to VM's vsock: %w", err)
		}

		// Perform Cloud Hypervisor vsock CONNECT handshake
		_, err = fmt.Fprintf(conn, "CONNECT %d\n", 8080)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to send CONNECT command: %w", err)
		}

		reader := bufio.NewReader(conn)
		resp, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
		}

		if !strings.HasPrefix(resp, "OK") {
			conn.Close()
			return nil, fmt.Errorf("vsock connection failed: %s", strings.TrimSpace(resp))
		}

		return conn, nil
	}

	conn, err := grpc.NewClient("passthrough:///vsock",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer conn.Close()

	agentAsyncClient := rpcgrpc.NewAgentClient(conn)
	execCall, err := agentAsyncClient.Exec(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to start remote command stream: %w", err)
	}

	// Prepare the initial execution command
	reqCmd := &rpc.ExecRequest_Command{
		Name:        commandName,
		Args:        commandArgs,
		Interactive: interactive,
		Tty:         tty,
	}

	if tty && term.IsTerminal(stdinFd) {
		width, height, err := term.GetSize(stdinFd)
		if err == nil {
			reqCmd.TerminalSize = &rpc.TerminalSize{
				Cols: uint32(width),
				Rows: uint32(height),
			}
		}
	}

	// Send the command
	err = execCall.Send(&rpc.ExecRequest{
		Type: &rpc.ExecRequest_Command_{
			Command: reqCmd,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send command to guest agent: %w", err)
	}

	// Process command events and optionally send our standard input and/or terminal dimensions
	g, ctx := errgroup.WithContext(cmd.Context())

	// Stream host's standard input if interactive mode is enabled
	if interactive {
		stdinReader := newContextReader(os.Stdin)

		g.Go(func() error {
			buf := make([]byte, 64*1024)
			for {
				n, err := stdinReader.Read(ctx, buf)
				if n > 0 {
					if sendErr := execCall.Send(&rpc.ExecRequest{
						Type: &rpc.ExecRequest_StandardInput{
							StandardInput: &rpc.IOChunk{
								Data: buf[:n],
							},
						},
					}); sendErr != nil {
						return sendErr
					}
				}
				if err != nil {
					if errors.Is(err, io.EOF) {
						// Signal EOF as we're done reading standard input
						return execCall.Send(&rpc.ExecRequest{
							Type: &rpc.ExecRequest_StandardInput{
								StandardInput: &rpc.IOChunk{
									Data: []byte{},
								},
							},
						})
					}
					return err
				}
			}
		})
	}

	// Stream host's terminal dimensions if pseudo-terminal is requested
	if tty && term.IsTerminal(stdinFd) {
		g.Go(func() error {
			sigwinchChan := make(chan os.Signal, 1)
			signal.Notify(sigwinchChan, syscall.SIGWINCH)
			defer signal.Stop(sigwinchChan)

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-sigwinchChan:
					width, height, err := term.GetSize(stdinFd)
					if err == nil {
						sendErr := execCall.Send(&rpc.ExecRequest{
							Type: &rpc.ExecRequest_TerminalResize{
								TerminalResize: &rpc.TerminalSize{
									Cols: uint32(width),
									Rows: uint32(height),
								},
							},
						})
						if sendErr != nil {
							return sendErr
						}
					}
				}
			}
		})
	}

	// Process command events
	g.Go(func() error {
		for {
			response, err := execCall.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}

			switch val := response.Type.(type) {
			case *rpc.ExecResponse_StandardOutput:
				_, _ = os.Stdout.Write(val.StandardOutput.Data)
			case *rpc.ExecResponse_StandardError:
				_, _ = os.Stderr.Write(val.StandardError.Data)
			case *rpc.ExecResponse_Exit_:
				return &ExecCustomExitCodeError{ExitCode: val.Exit.Code}
			}
		}
	})

	// Wait for all tasks to complete
	if err := g.Wait(); err != nil {
		var exitErr *ExecCustomExitCodeError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode != 0 {
				return exitErr
			}
			return nil
		}
		return err
	}

	return nil
}

// contextReader wraps an io.Reader with context-aware reads.
// A background goroutine performs the blocking Read and delivers
// results to a channel, allowing Read to be interrupted via context.
type contextReader struct {
	ch chan contextReaderResult
}

type contextReaderResult struct {
	data []byte
	err  error
}

func newContextReader(r io.Reader) *contextReader {
	cr := &contextReader{ch: make(chan contextReaderResult, 1)}

	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				cr.ch <- contextReaderResult{data: data}
			}
			if err != nil {
				cr.ch <- contextReaderResult{err: err}
				return
			}
		}
	}()

	return cr
}

func (cr *contextReader) Read(ctx context.Context, buf []byte) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case result := <-cr.ch:
		if result.err != nil {
			return 0, result.err
		}
		return copy(buf, result.data), nil
	}
}
