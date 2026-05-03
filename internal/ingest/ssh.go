package ingest

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

type SSHProvider struct {
	options sourceOptions
}

func (s *SSHProvider) Name() string {
	return s.options.name
}

func (s *SSHProvider) Start(ctx context.Context, out chan<- model.Event) error {
	authMethod, err := s.authMethod()
	if err != nil {
		return err
	}

	hostKeyCallback, err := s.hostKeyCallback()
	if err != nil {
		return err
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.options.ssh.Host, s.options.ssh.Port), &ssh.ClientConfig{
		User:            s.options.ssh.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
		Timeout:         5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("dial ssh source %q: %w", s.options.name, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create ssh session for %q: %w", s.options.name, err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open ssh stdout pipe for %q: %w", s.options.name, err)
	}

	command := s.options.ssh.Command
	if command == "" {
		if s.options.once {
			command = fmt.Sprintf("cat %s", shellEscape(s.options.path))
		} else {
			command = fmt.Sprintf("tail -n 0 -F %s", shellEscape(s.options.path))
		}
	}

	if err := session.Start(command); err != nil {
		return fmt.Errorf("start remote command for %q: %w", s.options.name, err)
	}

	go func() {
		<-ctx.Done()
		_ = session.Close()
		_ = client.Close()
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || shouldIgnore(s.options.ignore, line) {
			continue
		}
		event := model.Event{Source: s.options.name, Raw: line, Timestamp: time.Now().UTC()}
		if err := emitEvent(ctx, out, event); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan ssh output for %q: %w", s.options.name, err)
	}

	if err := session.Wait(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("wait on ssh session for %q: %w", s.options.name, err)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (s *SSHProvider) authMethod() (ssh.AuthMethod, error) {
	if s.options.ssh.Password != "" {
		return ssh.Password(s.options.ssh.Password), nil
	}

	privateKey, err := os.ReadFile(s.options.ssh.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key for source %q: %w", s.options.name, err)
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key for source %q: %w", s.options.name, err)
	}
	return ssh.PublicKeys(signer), nil
}

func (s *SSHProvider) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if s.options.ssh.KnownHostsPath == "" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	return knownhosts.New(s.options.ssh.KnownHostsPath)
}

func shellEscape(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
