package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v0"
)

var (
	wasmPath      = flag.String("wasm", "plain.go.wasm", "path to the transport WASM file")
	cancelDialCtx = flag.Bool("cancel-dial-ctx", true, "whether to cancel the dial context after message 1")
	useTCP        = flag.Bool("use-tcp", false, "test with TCP instead of WATER")
)

func fail(msg string, a ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", a...)
	os.Exit(1)
}

// echo will write all data read from conn back into conn. Assumes the peer will write first. Stops
// when the peer closes the connection or an error occurs (errors are printed to stderr). Closes
// conn before returning.
func echo(conn net.Conn) {
	defer conn.Close()
	b := make([]byte, 1024)

	for {
		n, err := conn.Read(b)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "echo: read error:", err)
			return
		}

		_, err = conn.Write(b[:n])
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "echo: write error:", err)
			return
		}
	}
}

type dialer[T net.Conn] interface {
	DialContext(ctx context.Context, network, addr string) (T, error)
}

func setUpWater(ctx context.Context) (net.Listener, dialer[water.Conn], error) {
	wasm, err := os.ReadFile(*wasmPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read WASM file: %w", err)
	}

	cfg := &water.Config{
		TransportModuleBin: wasm,
	}

	l, err := cfg.ListenContext(ctx, "tcp", "localhost:0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start listener: %w", err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Fprintln(os.Stderr, "listener accept error:", err)
				return
			}
			go echo(conn)
		}
	}()

	d, err := water.NewDialerWithContext(ctx, cfg)
	if err != nil {
		l.Close()
		return nil, nil, fmt.Errorf("failed to create dialer: %w", err)
	}

	return l, d, err
}

func setUpTCP(ctx context.Context) (net.Listener, dialer[net.Conn], error) {
	l, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "localhost:0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start listener: %w", err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Fprintln(os.Stderr, "listener accept error:", err)
				return
			}
			go echo(conn)
		}
	}()

	return l, &net.Dialer{}, err
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var conn net.Conn

	dialCtx, cancelDial := context.WithCancel(ctx)
	defer cancelDial()

	// Set up a connection to a local echo listener using either TCP or WATER. In either case, dial
	// using dialCtx.
	if *useTCP {
		l, d, err := setUpTCP(ctx)
		if err != nil {
			fail("tcp set up failed: %v", err)
		}
		defer l.Close()

		conn, err = d.DialContext(dialCtx, "tcp", l.Addr().String())
		if err != nil {
			fail("dial failed: %v", err)
		}

	} else {
		l, d, err := setUpWater(ctx)
		if err != nil {
			fail("water set up failed: %v", err)
		}
		defer l.Close()

		conn, err = d.DialContext(dialCtx, "tcp", l.Addr().String())
		if err != nil {
			fail("dial failed: %v", err)
		}
	}

	// Send 3 messages and print the results. After the second message, cancel the dial context (if
	// the relevant flag is set). This should not affect use of conn, which is already open.
	b := make([]byte, 1024)
	for i := 0; i < 3; i++ {
		msg := fmt.Sprintf("message %d", i)

		_, err := conn.Write([]byte(msg))
		if err != nil {
			fail("write failed: %v", err)
		}
		fmt.Println("write succeeded:", msg)

		n, err := conn.Read(b)
		if err != nil {
			fail("read failed: %v", err)
		}
		fmt.Println("read succeeded:", string(b[:n]))

		if i == 1 && *cancelDialCtx {
			fmt.Println("cancelling dial")
			cancelDial()
		}

		// Sleep for good measure.
		time.Sleep(time.Second)
	}
}
