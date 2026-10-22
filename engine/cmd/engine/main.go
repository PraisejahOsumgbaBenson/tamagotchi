// Command engine runs the Tamagotchi and serves it to the shell over a
// WebSocket on localhost.
//
// The engine is the half of the app that knows how to keep a creature alive.
// It does not need a window, so it runs on its own, survives the shell being
// closed, and can be tested without a display.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/PraisejahOsumgbaBenson/tamagotchi/engine/internal/engine"
	"github.com/PraisejahOsumgbaBenson/tamagotchi/engine/internal/proto"
	"github.com/gorilla/websocket"
)

func main() {
	var (
		addr      = flag.String("addr", "127.0.0.1:8733", "address to listen on")
		statePath = flag.String("state", defaultStatePath(), "where to keep the save state")
		headless  = flag.Bool("headless", false, "print frames to stdout instead of serving them")
		frames    = flag.Int("frames", 0, "in headless mode, stop after this many frames")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	eng := engine.New(engine.Options{
		StatePath: *statePath,
	})

	stop := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		slog.Info("shutting down")
		close(stop)
	}()

	if *headless {
		runHeadless(eng, *frames)
		return
	}

	// The pet lives in this loop, so it has to start before anything that
	// blocks.
	go eng.Run(stop)

	// The server blocks until it stops, so it gets its own goroutine and its
	// error comes back on a channel.
	serveErrs := make(chan error, 1)
	go func() {
		serveErrs <- serve(eng, *addr)
	}()

	slog.Info("the engine is running", "address", "ws://"+*addr+"/ws", "state", *statePath)

	select {
	case err := <-serveErrs:
		if err != nil {
			slog.Error("the engine could not start", "error", err)
			os.Exit(1)
		}
	case <-stop:
	}

	// Give the pet a moment to be written out so it is still there
	// tomorrow.
	if err := eng.Save(); err != nil {
		slog.Error("could not save the state", "error", err)
	}
}

// serve runs the WebSocket server until the stop channel is closed.
func serve(eng *engine.Engine, addr string) error {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		// The shell is a local process talking to a local server. There is
		// no browser origin to check.
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("could not upgrade the connection", "error", err)
			return
		}
		handleConn(eng, conn)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"version": proto.ProtocolVersion,
		})
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		slog.Info("listening", "address", addr)
		errs <- server.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// handleConn serves one shell.
func handleConn(eng *engine.Engine, conn *websocket.Conn) {
	defer conn.Close()

	slog.Info("a shell connected")

	messages, cancel := eng.Subscribe()
	defer cancel()

	// Tell the shell what this is, then bring it up to date.
	send(conn, proto.Hello())
	send(conn, proto.Clock(eng.Now()))
	send(conn, eng.Snapshot())

	done := make(chan struct{})
	go func() {
		defer close(done)
		for m := range messages {
			if !send(conn, m) {
				return
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		m, err := proto.Decode(data)
		if err != nil {
			slog.Warn("ignoring a message it could not read", "error", err)
			continue
		}
		eng.Handle(m)
	}

	conn.Close()
	<-done
	slog.Info("a shell disconnected")
}

// send writes one message to the shell.
func send(conn *websocket.Conn, m proto.Message) bool {
	data, err := proto.Encode(m)
	if err != nil {
		slog.Warn("could not encode a message", "error", err)
		return true
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return false
	}
	return true
}

// runHeadless prints frames as JSON lines. It exists so that the engine can
// be driven without a display, which is how the behaviour gets tested and
// how a trace gets recorded.
func runHeadless(eng *engine.Engine, limit int) {
	encoder := json.NewEncoder(os.Stdout)
	messages, cancel := eng.Subscribe()
	defer cancel()

	ticker := time.NewTicker(time.Second / proto.FrameRate)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case m, ok := <-messages:
			if !ok {
				return
			}
			if m.Kind != proto.KindFrame {
				continue
			}
			if err := encoder.Encode(m); err != nil {
				fmt.Fprintf(os.Stderr, "engine: %v\n", err)
				os.Exit(1)
			}
			count++
			if limit > 0 && count >= limit {
				return
			}
		case <-ticker.C:
			eng.Step()
		}
	}
}

// defaultStatePath returns where the save state lives.
func defaultStatePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "tama-pet.json"
	}
	return filepath.Join(dir, "tama", "pet.json")
}
