package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func EscapeCharacter() rune {
	switch runtime.GOOS {
	case "windows":
		return '^'
	default:
		return '\\'
	}
}

type CmdI interface {
	Cmd() *exec.Cmd
	Start() error
	Wait() error
}

func NewCmdI2(args []string) CmdI {
	cmd := exec.Command(args[0], args[1:]...)
	return NewBasicCmd(cmd)
}

func NewCmdIShell(ctx context.Context, args ...string) CmdI {
	c := NewCmdI2(args)
	c = NewNoHangPipeCmd(c, true, true, true) // active only if the pipe is not nil (ex: cmd.stdin)
	if ctx != nil {
		// NOTE: not using exec.CommandContext because the ctx is dealt with in NewCtxCmd to better handle termination
		c = NewCtxCmd(ctx, c)
	}
	return NewShellCmd(c, true)
}

type BasicCmd struct {
	cmd *exec.Cmd
}

func NewBasicCmd(cmd *exec.Cmd) *BasicCmd {
	return &BasicCmd{cmd: cmd}
}
func (c *BasicCmd) Cmd() *exec.Cmd {
	return c.cmd
}
func (c *BasicCmd) Start() error {
	return c.cmd.Start()
}
func (c *BasicCmd) Wait() error {
	return c.cmd.Wait()
}

type ShellCmd struct {
	CmdI
}

func NewShellCmd(cmdi CmdI, scriptArgs bool) *ShellCmd {
	c := &ShellCmd{CmdI: cmdi}
	cmd := c.CmdI.Cmd()

	// update cmd.path with shell executable
	name := cmd.Args[0]
	if lp, err := exec.LookPath(name); err != nil {
		cmd.Path = name
		cmd.Err = err
	} else {
		cmd.Path = lp
		cmd.Err = nil // clear explicitly, exec.command can set this at init when it doesn't find the exec with the original args
	}

	return c
}

// Old note: explanations on possible hangs.
// https://github.com/golang/go/issues/18874#issuecomment-277280139

//CtxCmd behaviour is somewhat equivalent to:
// 	cmd := exec.CommandContext(ctx, args[0], args[1:]...))
// 	cmd.WaitDelay = X * time.Second
//but it has a custom error msg and sends a term signal that can include the process group. Beware that if using this, the exec.cmd should probably not be started with exec.commandcontext, since that will have the ctx cancel run first (before this handler) and when it gets here the process is already canceled.

type CtxCmd struct {
	CmdI
	ctx context.Context
}

func NewCtxCmd(ctx context.Context, cmdi CmdI) *CtxCmd {
	c := &CtxCmd{CmdI: cmdi, ctx: ctx}

	// SetupExecCmdSysProcAttr(c.CmdI.Cmd())

	return c
}
func (c *CtxCmd) Start() error {
	return c.CmdI.Start()
}
func (c *CtxCmd) Wait() error {
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- c.CmdI.Wait()
	}()
	select {
	case err := <-waitCh:
		return err
	case <-c.ctx.Done():
		_ = c.CmdI.Cmd().Process.Kill()

		// wait for the possibility of wait returning after kill
		timeout := 3 * time.Second
		select {
		case err := <-waitCh:
			return err
		case <-time.After(timeout):
			// warn about the process not returning
			s := fmt.Sprintf("termination timeout (%v): process has not returned from wait (ex: a subprocess might be keeping a file descriptor open). Beware that these processes might produce output visible here.\n", timeout)
			//c.printf(s)

			// exit now (leaks waitCh go routine)
			//return c.ctx.Err()
			return errors.New(s)

			//// wait forever
			//return <-waitCh
		}
	}
}

func NewNoHangStdinCmd(cmdi CmdI) *NoHangPipeCmd {
	return &NoHangPipeCmd{CmdI: cmdi, doIn: true}
}

type NoHangPipeCmd struct {
	CmdI
	doIn, doOut, doErr bool
	stdin              io.WriteCloser
	//stdout             io.ReadCloser
	//stderr             io.ReadCloser
	//outPipes           sync.WaitGroup // stdout/stderr pipe wait
}

func NewNoHangPipeCmd(cmdi CmdI, doIn, doOut, doErr bool) *NoHangPipeCmd {
	return &NoHangPipeCmd{CmdI: cmdi, doIn: doIn, doOut: doOut, doErr: doErr}
}
func (c *NoHangPipeCmd) Start() error {
	cmd := c.Cmd()
	if c.doIn && cmd.Stdin != nil {
		r := cmd.Stdin
		cmd.Stdin = nil // cmd wants nil here
		wc, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		c.stdin = wc
		go func() {
			_, _ = io.Copy(wc, r)
			_ = wc.Close()
		}()
	}
	//if c.doOut && cmd.Stdout != nil {
	//	w := cmd.Stdout
	//	cmd.Stdout = nil // cmd wants nil here
	//	rc, err := cmd.StdoutPipe()
	//	if err != nil {
	//		return err
	//	}
	//	c.stdout = rc
	//	c.outPipes.Add(1)
	//	go func() {
	//		defer c.outPipes.Done()
	//		_, _ = io.Copy(w, rc)
	//		_ = rc.Close()
	//	}()
	//}
	//if c.doErr && cmd.Stderr != nil {
	//	w := cmd.Stderr
	//	cmd.Stderr = nil // cmd wants nil here
	//	rc, err := cmd.StderrPipe()
	//	if err != nil {
	//		return err
	//	}
	//	c.stderr = rc
	//	c.outPipes.Add(1)
	//	go func() {
	//		defer c.outPipes.Done()
	//		_, _ = io.Copy(w, rc)
	//		_ = rc.Close()
	//	}()
	//}
	return c.CmdI.Start()
}

//func (c *NoHangPipeCmd) Wait() error {
//	//c.outPipes.Wait() // wait for stdout/stderr pipes before calling wait
//	return c.CmdI.Wait()
//}

// some commands will not exit unless the stdin is closed, allow access
func (c *NoHangPipeCmd) CloseStdin() error {
	if c.stdin != nil {
		return c.stdin.Close()
	}
	return nil
}

func RunCmdI(ci CmdI) error {
	if err := ci.Start(); err != nil {
		return err
	}
	return ci.Wait()
}
func RunCmdIOutputs(c CmdI) (sout []byte, serr []byte, _ error) {
	obuf := &bytes.Buffer{}
	ebuf := &bytes.Buffer{}

	cmd := c.Cmd()
	if cmd.Stdout != nil {
		return nil, nil, fmt.Errorf("stdout already set")
	}
	if cmd.Stderr != nil {
		return nil, nil, fmt.Errorf("stderr already set")
	}
	cmd.Stdout = obuf
	cmd.Stderr = ebuf

	err := RunCmdI(c)
	return obuf.Bytes(), ebuf.Bytes(), err
}
func RunCmdICombineStderrErr(c CmdI) ([]byte, error) {
	bout, berr, err := RunCmdIOutputs(c)
	if err != nil {
		serr := strings.TrimSpace(string(berr))
		if serr != "" {
			err = fmt.Errorf("%w: stderr(%v)", err, serr)
		}
		return nil, err
	}
	return bout, nil
}

func RunCmdStdin(ctx context.Context, dir string, rd io.Reader, args ...string) ([]byte, error) {
	c := NewCmdIShell(ctx, args...)
	c.Cmd().Dir = dir
	c.Cmd().Stdin = rd
	return RunCmdICombineStderrErr(c)
}
