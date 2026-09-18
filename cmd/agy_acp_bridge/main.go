package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/coder/acp-go-sdk"
)

type SessionInfo struct {
	Cwd string
}

type AgyAgent struct {
	conn     *acp.AgentSideConnection
	sessions map[string]*SessionInfo
	mu       sync.Mutex
	agyPath  string
}

func NewAgyAgent(agyPath string) *AgyAgent {
	return &AgyAgent{
		sessions: make(map[string]*SessionInfo),
		agyPath:  agyPath,
	}
}

func (a *AgyAgent) SetConn(conn *acp.AgentSideConnection) {
	a.conn = conn
}

func (a *AgyAgent) Initialize(ctx context.Context, req acp.InitializeRequest) (acp.InitializeResponse, error) {
	title := "Google Antigravity"
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentInfo: &acp.Implementation{
			Name:    "antigravity-acp",
			Title:   &title,
			Version: "1.2.5",
		},
		AgentCapabilities: acp.AgentCapabilities{
			PromptCapabilities: acp.PromptCapabilities{
				Image: true,
			},
		},
	}, nil
}

func (a *AgyAgent) Authenticate(ctx context.Context, req acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *AgyAgent) Logout(ctx context.Context, req acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, nil
}

func (a *AgyAgent) NewSession(ctx context.Context, req acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sessID := fmt.Sprintf("agy-sess-%d", len(a.sessions)+1)
	a.sessions[sessID] = &SessionInfo{
		Cwd: req.Cwd,
	}

	return acp.NewSessionResponse{
		SessionId: acp.SessionId(sessID),
	}, nil
}

func (a *AgyAgent) CloseSession(ctx context.Context, req acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, string(req.SessionId))
	return acp.CloseSessionResponse{}, nil
}

func (a *AgyAgent) ListSessions(ctx context.Context, req acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{
		Sessions: []acp.SessionInfo{},
	}, nil
}

func (a *AgyAgent) ResumeSession(ctx context.Context, req acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, nil
}

func (a *AgyAgent) SetSessionMode(ctx context.Context, req acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

func (a *AgyAgent) SetSessionConfigOption(ctx context.Context, req acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

func (a *AgyAgent) Cancel(ctx context.Context, notif acp.CancelNotification) error {
	return nil
}

func (a *AgyAgent) Prompt(ctx context.Context, req acp.PromptRequest) (acp.PromptResponse, error) {
	a.mu.Lock()
	sess, ok := a.sessions[string(req.SessionId)]
	a.mu.Unlock()

	var promptText strings.Builder
	var imagePaths []string
	for _, block := range req.Prompt {
		if block.Text != nil {
			promptText.WriteString(block.Text.Text)
			promptText.WriteString("\n")
		}
		if block.Image != nil && block.Image.Data != "" {
			data, err := base64.StdEncoding.DecodeString(block.Image.Data)
			if err == nil {
				ext := ".png"
				if strings.Contains(block.Image.MimeType, "jpeg") || strings.Contains(block.Image.MimeType, "jpg") {
					ext = ".jpg"
				} else if strings.Contains(block.Image.MimeType, "webp") {
					ext = ".webp"
				}
				tmpFile, err := os.CreateTemp("", "typstify_paste_*"+ext)
				if err == nil {
					_, _ = tmpFile.Write(data)
					tmpPath := tmpFile.Name()
					_ = tmpFile.Close()
					imagePaths = append(imagePaths, tmpPath)
				}
			}
		}
	}

	var fullPrompt strings.Builder
	for _, imgPath := range imagePaths {
		fullPrompt.WriteString(imgPath)
		fullPrompt.WriteString(" ")
	}
	fullPrompt.WriteString(promptText.String())

	rawPrompt := strings.TrimSpace(fullPrompt.String())
	if rawPrompt == "" {
		return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
	}

	cwd := ""
	if ok && sess != nil && sess.Cwd != "" {
		cwd = sess.Cwd
	}

	cmd := exec.CommandContext(ctx, a.agyPath, "--dangerously-skip-permissions", "--print", rawPrompt)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, err
	}

	reader := bufio.NewReader(stdout)
	buf := make([]byte, 256)

	for {
		n, err := reader.Read(buf)
		if n > 0 && a.conn != nil {
			chunk := string(buf[:n])
			_ = a.conn.SessionUpdate(ctx, acp.SessionNotification{
				SessionId: req.SessionId,
				Update: acp.SessionUpdate{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock(chunk),
					},
				},
			})
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
	}

	_ = cmd.Wait()
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
}

func findAgyPath() string {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "AppData", "Local", "agy", "bin", "agy.EXE")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	p2 := filepath.Join(home, "AppData", "Local", "agy", "bin", "agy.exe")
	if _, err := os.Stat(p2); err == nil {
		return p2
	}
	if lp, err := exec.LookPath("agy"); err == nil {
		return lp
	}
	return "agy.EXE"
}

func main() {
	agyPath := findAgyPath()
	agent := NewAgyAgent(agyPath)

	conn := acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
	agent.SetConn(conn)

	<-conn.Done()
}
