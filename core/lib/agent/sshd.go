package agent

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

// SSHD start a ssh server to provide shell access for clients
// the server binds local interface only
func SSHD(shell, port string) (err error) {
	ssh.Handle(func(s ssh.Session) {
		exe, err := exec.LookPath(shell)
		if err != nil {
			res := fmt.Sprintf("%s not found (%v), aborting", shell, err)
			log.Print(res)
			io.WriteString(s, res)
			return
		}

		cmd := exec.Command(exe)
		ptyReq, winCh, isPTY := s.Pty()
		log.Printf("Got an SSH PTY request: %s", ptyReq.Term)
		if isPTY {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		}
		f, err := pty.Start(cmd)
		if err != nil {
			err = fmt.Errorf("Start PTY: %v", err)
			io.WriteString(s, err.Error())
			return
		}
		go func() {
			for win := range winCh {
				setWinsize(f, win.Width, win.Height)
			}
		}()
		go func() {
			io.Copy(f, s) // stdin
		}()
		io.Copy(s, f) // stdout
		cmd.Wait()
	})

	log.Printf("Starting SSHD on port %s...", port)
	return ssh.ListenAndServe("127.0.0.1:"+port, nil)
}
