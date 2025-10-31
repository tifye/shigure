package sshapp

import (
	"context"
	"errors"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	wishTea "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

type SSHApp struct {
	s      *ssh.Server
	logger *log.Logger
}

type SSHAppOptions struct {
	Host        string
	Port        string
	HostKeyPath string
}

func NewSSHApp(opts SSHAppOptions, logger *log.Logger) (*SSHApp, error) {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(opts.Host, opts.Port)),
		wish.WithHostKeyPath(opts.HostKeyPath),
		wish.WithMiddleware(
			wishTea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.MiddlewareWithLogger(logger),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new wish server: %s", err)
	}

	return &SSHApp{
		s:      s,
		logger: logger,
	}, nil
}

func (s *SSHApp) Start() {
	go func() {
		err := s.s.ListenAndServe()
		if err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			// todo: should either signal a retry to notify in someway
			s.logger.Error("Could not serve server", "error", err)
		}
	}()
}

func (s *SSHApp) Stop(ctx context.Context) error {
	err := s.s.Shutdown(ctx)
	if err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	renderer := wishTea.MakeRenderer(s)

	m := model{
		term:       pty.Term,
		profile:    renderer.ColorProfile().Name(),
		width:      uint(pty.Window.Width),
		height:     uint(pty.Window.Height),
		isDarkMode: renderer.HasDarkBackground(),
	}

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type model struct {
	term       string
	profile    string
	width      uint
	height     uint
	isDarkMode bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	colorScheme := "Light mode"
	if m.isDarkMode {
		colorScheme = "Dark mode"
	}
	return fmt.Sprintf("%s %s %s [w,h][%d,%d]", m.term, m.profile, colorScheme, m.width, m.height)
}
