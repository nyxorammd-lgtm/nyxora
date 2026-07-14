package transport

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

type SSH struct {
	BaseTransport
	user      string
	password  string
	localPort int
}

func NewSSH() *SSH {
	return &SSH{
		BaseTransport: NewBase("ssh", "ssh", 22, ScoringWeights{0.30, 0.30, 0.15, 0.25}, 0),
		user:          "root",
		localPort:     1080,
	}
}

func (s *SSH) Name() string  { return s.BaseName() }
func (s *SSH) Type() string { return s.BaseType() }

func (s *SSH) Init(cfg map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &s.port)
	}
	if u, ok := cfg["user"]; ok {
		s.user = u
	}
	if pw, ok := cfg["password"]; ok {
		s.password = pw
	}
	log.Printf("[ssh] initialized (port: %d, user: %s)", s.port, s.user)
	return nil
}

func (s *SSH) Connect(remoteAddr string) error {
	ctx := s.CancelContext()
	s.KillOldProcess()
	if err := s.BaseConnectInit(remoteAddr); err != nil {
		return err
	}

	s.mu.Lock()
	password := s.password
	user := s.user
	localPort := s.localPort
	port := s.port
	s.mu.Unlock()

	usePassword := password != "" && CommandExists("sshpass")

	if usePassword {
		tunnelCmd := exec.Command("sshpass",
			"-e",
			"ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "ServerAliveInterval=10",
			"-o", "ServerAliveCountMax=3",
			"-o", "ExitOnForwardFailure=yes",
			"-N",
			"-D", fmt.Sprintf("127.0.0.1:%d", localPort),
			"-p", fmt.Sprintf("%d", port),
			fmt.Sprintf("%s@%s", user, remoteAddr),
		)
		tunnelCmd.Env = append(os.Environ(), "SSHPASS="+password)

		s.SetCmd(tunnelCmd)
		s.Logf("starting password tunnel %s@%s:%d -> 127.0.0.1:%d",
			user, remoteAddr, port, localPort)

		if err := tunnelCmd.Start(); err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			s.Logf("tunnel start error: %v", err)
			s.SetStatusFailed()
			return nil
		}

		s.SetStatusActive()
		s.KillOnCancel()

		go func() {
			if err := tunnelCmd.Wait(); err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				s.Logf("tunnel exited: %v", err)
				s.SetStatusFailed()
			}
		}()

		s.Logf("connecting via sshpass tunnel to %s:%d", remoteAddr, s.port)
		return nil
	}

	s.Logf("sshpass not available, using ping-only mode")
	s.SetStatusActive()
	return nil
}

func (s *SSH) Disconnect() error {
	return s.BaseDisconnect()
}

func (s *SSH) Status() Status  { return s.BaseStatus() }
func (s *SSH) Metrics() *Metrics { return s.BaseMetrics() }
func (s *SSH) Health() bool    { return s.BaseHealth() }
func (s *SSH) Score() float64  { return s.BaseScore() }
