package vardumper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

// PHPProcess manages a long-running PHP subprocess for deserializing VarDumper data.
type PHPProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

// ParseResult is the JSON response from the PHP parser.
type ParseResult struct {
	Type     string          `json:"type"`              // "string", "array", "object", "code", etc.
	Value    string          `json:"value"`             // Rendered value (primitive string or HTML)
	Label    *string         `json:"label,omitempty"`   // Optional label
	Language *string         `json:"language,omitempty"`
	Context  json.RawMessage `json:"context"`
	Project  *string         `json:"project,omitempty"`
	Error    string          `json:"error,omitempty"`   // Set on parse failure
}

// StartPHPProcess extracts the embedded PHP binary and starts it.
func StartPHPProcess() (*PHPProcess, error) {
	// Extract embedded binary to temp file.
	tmpFile, err := os.CreateTemp("", "vardumper-parser-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tmpFile.Write(phpBinary); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("write PHP binary: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("chmod PHP binary: %w", err)
	}

	slog.Info("starting PHP VarDumper parser", "path", tmpFile.Name())

	cmd := exec.Command(tmpFile.Name())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("start PHP process: %w", err)
	}

	return &PHPProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReaderSize(stdout, 1024*1024), // 1MB buffer for large HTML dumps
	}, nil
}

// Parse sends a base64 payload to the PHP process and reads the JSON result.
func (p *PHPProcess) Parse(base64Payload string) (*ParseResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Send payload line.
	if _, err := fmt.Fprintf(p.stdin, "%s\n", base64Payload); err != nil {
		return nil, fmt.Errorf("write to PHP: %w", err)
	}

	// Read response line.
	line, err := p.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read from PHP: %w", err)
	}

	var result ParseResult
	if err := json.Unmarshal([]byte(line), &result); err != nil {
		return nil, fmt.Errorf("unmarshal PHP response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("PHP parser error: %s", result.Error)
	}

	return &result, nil
}

// Stop terminates the PHP process.
func (p *PHPProcess) Stop() {
	p.stdin.Close()
	_ = p.cmd.Wait()
}
