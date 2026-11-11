package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
)

type DesktopNotification struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	ConversationID string `json:"conversationId"`
}

func (a *App) SendDesktopNotification(note DesktopNotification) error {
	if !mygo.NotificationsSupported() {
		return fmt.Errorf("desktop notifications are not available")
	}
	return mygo.NewNotification(mygo.NotificationOptions{
		ID: note.ID, Title: note.Title, Body: note.Body, Group: note.ConversationID,
	}).Show()
}

func (a *App) CheckForUpdates(_ context.Context) (*UpdateCheckResult, error) {
	if mygo.IsDev() || !isReleaseVersion(appVersion) {
		return nil, fmt.Errorf("updates are only available in release builds")
	}
	if !mygo.Updater.Enabled() {
		if err := a.OpenAttachment("https://github.com/lsongdev/miya-desktop/releases"); err != nil {
			return nil, fmt.Errorf("open releases: %w", err)
		}
		return &UpdateCheckResult{
			Status: "manual", State: "manual", CurrentVersion: appVersion,
			Message: "Release downloads opened in your browser. This build does not support signed automatic updates.",
		}, nil
	}
	updater.CheckForUpdates()
	return &UpdateCheckResult{
		Status: "checking", State: "checking", CurrentVersion: appVersion,
		Message: "Check the update window for download and installation progress.",
	}, nil
}

func (a *App) Version() string {
	return appVersion
}

func openWithDefaultApp(target string) error {
	var err error
	lower := strings.ToLower(target)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		err = mygo.Shell.OpenExternal(target)
	} else {
		err = mygo.Shell.OpenPath(target)
	}
	if err != nil {
		return fmt.Errorf("open attachment: %w", err)
	}
	return nil
}
