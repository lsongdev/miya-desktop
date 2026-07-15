package main

import (
	"context"
	"fmt"
	"os"

	"wails-app/internal/agent"
)

// App struct
type App struct {
	ctx     context.Context
	manager *agent.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.manager = agent.New(ctx)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) ConnectAgent(command string) error {
	return a.manager.Connect(command)
}

func (a *App) InitializeAgent(name, version string) (*agent.AgentInfo, error) {
	return a.manager.Initialize(name, version)
}

func (a *App) CreateSession(cwd string) (*agent.Session, error) {
	return a.manager.NewSession(cwd)
}

func (a *App) DefaultCwd() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		return cwd
	}
	return "."
}

func (a *App) LoadSession(sessionID, cwd string) error {
	return a.manager.LoadSession(sessionID, cwd)
}

func (a *App) SendPrompt(sessionID, message string) error {
	return a.manager.Prompt(sessionID, message)
}

func (a *App) GetConversation(sessionID string) (*agent.Conversation, error) {
	return a.manager.GetConversation(sessionID)
}

func (a *App) ListSessions() ([]agent.Session, error) {
	return a.manager.ListSessions()
}

func (a *App) CloseSession(sessionID string) error {
	return a.manager.CloseSession(sessionID)
}

func (a *App) CancelSession(sessionID string) error {
	return a.manager.CancelSession(sessionID)
}

func (a *App) DeleteSession(sessionID string) error {
	return a.manager.DeleteSession(sessionID)
}

func (a *App) DisconnectAgent() error {
	return a.manager.Disconnect()
}

func (a *App) ReconnectAgent() error {
	return a.manager.Reconnect()
}
