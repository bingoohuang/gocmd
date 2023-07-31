package main

import (
	"context"
	"flag"
	"github.com/bingoohuang/gocmd"
	"github.com/bingoohuang/gocmd/linestream"
	"github.com/bingoohuang/gocmd/shellquote"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	flag.Parse()
	args := flag.Args()

	var (
		err     error
		timeout time.Duration
	)

	var options []func(*gocmd.Cmd)
	if env := os.Getenv("TIMEOUT"); env != "" {
		timeout, err = time.ParseDuration(env)
		if err != nil {
			log.Fatalf("parse $TIMEOUT=%s: %v", env, err)
		}
		options = append(options, gocmd.WithTimeout(timeout))
	}

	if env := os.Getenv("WORKING_DIR"); env != "" {
		options = append(options, gocmd.WithWorkingDir(env))
	}

	if env := os.Getenv("LINES"); env == "1" {
		options = append(options, gocmd.WithStdout(linestream.New(func(line string) {
			log.Printf("line: %s", line)
		})))
	}

	shell := shellquote.QuoteMust(args...)

	if env := os.Getenv("NOSH"); env == "1" {
		shell = ""
		options = append(options, gocmd.WithCmd(exec.Command(args[0], args[1:]...)))
	}

	if shell != "" {
		log.Printf("shell: %q", shell)
	}

	cmd := gocmd.New(shell, options...)
	if err := cmd.Run(context.Background()); err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Printf("stdout: %s", cmd.Stdout())
	log.Printf("stderr: %s", cmd.Stderr())
	log.Printf("exitCode: %d", cmd.ExitCode())
}
