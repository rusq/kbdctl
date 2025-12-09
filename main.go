package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime/trace"
	"time"

	"github.com/rusq/kbdctl/kbd"
)

var (
	dumpConfig = flag.Bool("dump-config", false, "dump keyboard configuration")
	setTime    = flag.Bool("set-time", false, "set keyboard time to current system time")
	tracefile  = flag.String("trace", "", "write trace to `file`")
)

func main() {
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *tracefile != "" {
		f, err := os.Create(*tracefile)
		if err != nil {
			log.Fatalf("failed to create trace file: %v", err)
		}
		defer f.Close()
		if err := trace.Start(f); err != nil {
			log.Fatalf("failed to start trace: %v", err)
		}
		defer trace.Stop()
	}

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("OK")
}

func run(ctx context.Context) error {
	kb, err := kbd.NewKeyboard()
	if err != nil {
		return fmt.Errorf("failed to open keyboard: %w", err)
	}
	defer func() {
		if err := kb.Close(); err != nil {
			slog.Error("failed to close keyboard", "error", err)
		}
	}()

	if *dumpConfig {
		cfg, err := kb.LoadConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		fmt.Printf("% x\n", cfg)
		return nil
	}

	if *setTime {
		t := time.Now()
		if err := kb.SetTime(ctx, t); err != nil {
			return fmt.Errorf("failed to set time: %w", err)
		}
		slog.Info("time set", "time", t, "took", time.Since(t))
		return nil
	}

	return nil
}
