package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/godynheil/kk/internal/app"
)

func init() {
	if runtime.GOOS == "windows" {
		handle := syscall.Handle(os.Stdout.Fd())
		var mode uint32
		if err := syscall.GetConsoleMode(handle, &mode); err == nil {
			mode |= 0x0004
			_, _, _ = syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode").Call(uintptr(handle), uintptr(mode))
		}
	}
}

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "kk:", err)
		os.Exit(1)
	}
}
