package channels

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Status struct {
	Running bool   `json:"running"`
	Command string `json:"command,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Service struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
	status Status
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) Start(ctx context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		s.status.Running = true
		return s.status, nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	cmd, label, err := newCommand(runCtx)
	if err != nil {
		cancel()
		s.status = Status{Error: err.Error()}
		return s.status, err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		s.status = Status{Command: label, Error: err.Error()}
		return s.status, fmt.Errorf("start miya-channels: %w", err)
	}

	s.cmd = cmd
	s.cancel = cancel
	s.status = Status{Running: true, Command: label, PID: cmd.Process.Pid}

	go s.wait(runCtx, cmd, label)
	return s.status, nil
}

func (s *Service) Stop() (Status, error) {
	s.mu.Lock()
	cmd := s.cmd
	cancel := s.cancel
	if cmd == nil || cmd.Process == nil {
		s.cmd = nil
		s.cancel = nil
		s.status.Running = false
		status := s.status
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return s.Status(), fmt.Errorf("stop miya-channels: %w", err)
	}

	s.mu.Lock()
	s.cmd = nil
	s.cancel = nil
	s.status.Running = false
	status := s.status
	s.mu.Unlock()
	return status, nil
}

func (s *Service) wait(ctx context.Context, cmd *exec.Cmd, label string) {
	err := cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != cmd {
		return
	}
	s.cmd = nil
	s.cancel = nil
	s.status.Running = false
	s.status.PID = 0
	if ctx.Err() == nil && err != nil {
		s.status.Error = err.Error()
		log.Printf("[channels] %s exited: %v", label, err)
	}
}

func newCommand(ctx context.Context) (*exec.Cmd, string, error) {
	wd, err := filepath.Abs(filepath.Join("..", "miya-channels"))
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err != nil {
		return nil, "", fmt.Errorf("miya-channels repo not found at %s", wd)
	}
	if _, err := exec.LookPath("go"); err != nil {
		return nil, "", fmt.Errorf("go is unavailable for running miya-channels")
	}
	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = wd
	return cmd, "go run ../miya-channels", nil
}
