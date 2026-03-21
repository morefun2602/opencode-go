package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Client is a Language Server Protocol client communicating via stdio.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	log    *slog.Logger

	mu       sync.Mutex
	nextID   atomic.Int64
	pending  map[int64]chan json.RawMessage
	diagMu   sync.RWMutex
	diagCache map[string][]Diagnostic

	initialized bool
}

// Diagnostic represents an LSP diagnostic.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"`
}

// Range represents a range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Location represents a location in a file.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// DocumentSymbol represents a symbol in a document.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient starts an LSP server process and initializes the connection.
func NewClient(ctx context.Context, command string, args []string, workspaceRoot string, log *slog.Logger) (*Client, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start lsp server: %w", err)
	}

	c := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		log:       log,
		pending:   make(map[int64]chan json.RawMessage),
		diagCache: make(map[string][]Diagnostic),
	}

	go c.readLoop()

	if err := c.initialize(ctx, workspaceRoot); err != nil {
		c.Close()
		return nil, fmt.Errorf("lsp initialize: %w", err)
	}

	return c, nil
}

func (c *Client) initialize(ctx context.Context, workspaceRoot string) error {
	params := map[string]any{
		"processId": nil,
		"rootUri":   "file://" + workspaceRoot,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"publishDiagnostics": map[string]any{},
				"definition":         map[string]any{},
				"references":         map[string]any{},
				"documentSymbol":     map[string]any{},
			},
		},
	}
	_, err := c.call(ctx, "initialize", params)
	if err != nil {
		return err
	}
	_ = c.notify("initialized", map[string]any{})
	c.initialized = true
	return nil
}

// GetDiagnostics returns cached diagnostics for a file.
func (c *Client) GetDiagnostics(uri string) []Diagnostic {
	c.diagMu.RLock()
	defer c.diagMu.RUnlock()
	return c.diagCache[uri]
}

// Definition sends textDocument/definition request.
func (c *Client) Definition(ctx context.Context, uri string, line, character int) ([]Location, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": character},
	}
	raw, err := c.call(ctx, "textDocument/definition", params)
	if err != nil {
		return nil, err
	}
	var locations []Location
	if err := json.Unmarshal(raw, &locations); err != nil {
		var single Location
		if err2 := json.Unmarshal(raw, &single); err2 == nil {
			return []Location{single}, nil
		}
		return nil, fmt.Errorf("parse definition: %w", err)
	}
	return locations, nil
}

// References sends textDocument/references request.
func (c *Client) References(ctx context.Context, uri string, line, character int) ([]Location, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": character},
		"context":      map[string]any{"includeDeclaration": true},
	}
	raw, err := c.call(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}
	var locations []Location
	if err := json.Unmarshal(raw, &locations); err != nil {
		return nil, fmt.Errorf("parse references: %w", err)
	}
	return locations, nil
}

// DocumentSymbols sends textDocument/documentSymbol request.
func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
	}
	raw, err := c.call(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}
	var symbols []DocumentSymbol
	if err := json.Unmarshal(raw, &symbols); err != nil {
		return nil, fmt.Errorf("parse symbols: %w", err)
	}
	return symbols, nil
}

// Close shuts down the LSP server.
func (c *Client) Close() error {
	if c.initialized {
		ctx := context.Background()
		_, _ = c.call(ctx, "shutdown", nil)
		_ = c.notify("exit", nil)
	}
	c.stdin.Close()
	return c.cmd.Wait()
}

func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan json.RawMessage, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	if err := c.send(req); err != nil {
		return nil, err
	}

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) notify(method string, params any) error {
	req := jsonRPCRequest{JSONRPC: "2.0", Method: method, Params: params}
	return c.send(req)
}

func (c *Client) send(req jsonRPCRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(b))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(b)
	return err
}

func (c *Client) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		contentLen := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				v := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLen, _ = strconv.Atoi(v)
			}
		}
		if contentLen == 0 {
			continue
		}

		body := make([]byte, contentLen)
		if _, err := io.ReadFull(reader, body); err != nil {
			return
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		if resp.Method == "textDocument/publishDiagnostics" {
			c.handleDiagnostics(resp.Params)
			continue
		}

		if resp.ID > 0 {
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			c.mu.Unlock()
			if ok {
				if resp.Error != nil {
					ch <- json.RawMessage(fmt.Sprintf(`{"error": "%s"}`, resp.Error.Message))
				} else {
					ch <- resp.Result
				}
			}
		}
	}
}

func (c *Client) handleDiagnostics(params json.RawMessage) {
	var p struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	c.diagMu.Lock()
	c.diagCache[p.URI] = p.Diagnostics
	c.diagMu.Unlock()
}
