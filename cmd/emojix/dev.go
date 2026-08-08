package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ponytail: 300ms poll is fine for local dev; fsnotify if it ever feels laggy.
func dev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	dbName := fs.String("db", "emojix.db", "sqlite file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	var cmd *exec.Cmd
	start := func() error {
		cmd = exec.Command("go", "run", "./cmd/emojix", "serve", "-db", *dbName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Start()
	}
	stop := func() {
		if cmd == nil || cmd.Process == nil {
			return
		}
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}

	stamp := sourceStamp()
	if err := start(); err != nil {
		return err
	}
	fmt.Println("dev: watching .go/.gohtml")

	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-sig:
			stop()
			return nil
		case <-tick.C:
			if s := sourceStamp(); s != stamp {
				stamp = s
				fmt.Println("dev: reload")
				stop()
				if err := start(); err != nil {
					return err
				}
			}
		}
	}
}

func sourceStamp() string {
	var b strings.Builder
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".gohtml":
			fmt.Fprintf(&b, "%s\x00%s\x00", path, info.ModTime())
		}
		return nil
	})
	return b.String()
}
