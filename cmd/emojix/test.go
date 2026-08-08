package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runTest() error {
	out, err := exec.Command("gofmt", "-l", ".").Output()
	if err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		return fmt.Errorf("gofmt needed:\n%s", files)
	}
	fmt.Println("fmt ok")

	if err := run("go", "vet", "./..."); err != nil {
		return err
	}
	fmt.Println("vet ok")

	if err := run("go", "test",
		"-race", "-cover", "-count=1", "-shuffle=on",
		"-coverpkg=./...", "-coverprofile=coverage.out",
		"./...",
	); err != nil {
		return err
	}

	out, err = exec.Command("go", "tool", "cover", "-func=coverage.out").Output()
	if err != nil {
		return err
	}
	// last line is the total summary
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	fmt.Println(lines[len(lines)-1])
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
