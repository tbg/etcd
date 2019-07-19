// +build !windows

package pty

import (
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Start assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
func Start(c *exec.Cmd) (pty *os.File, err error) {
	pty, tty, err := Open()
	if err != nil {
		return nil, err
	}
	go func() {
		<-time.After(10 * time.Second)
		tty.Close()
	}()
	so := io.MultiWriter(tty, os.Stdout)
	se := io.MultiWriter(tty, os.Stderr)
	c.Stdout = so
	c.Stderr = se
	//c.Stdout = tty
	c.Stdin = tty
	//c.Stderr = tty
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setctty = true
	c.SysProcAttr.Setsid = true
	err = c.Start()
	if err != nil {
		pty.Close()
		return nil, err
	}
	return pty, err
}
