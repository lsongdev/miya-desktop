package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/mp3"
	_ "github.com/egoist/mygo/audio/vorbis"
	"github.com/egoist/mygo/ui"
	"miya-desktop/internal/conversation"
)

func (s *nativeUI) attachmentBytes(block conversation.Block) ([]byte, error) {
	if block.Data != "" {
		return decodeAttachment(block.Data)
	}
	if strings.HasPrefix(block.URI, "file://") {
		attachment, err := s.backend.ReadAttachment(block.URI, 20<<20)
		if err != nil {
			return nil, err
		}
		return attachment.Data, nil
	}
	if !validHTTPURL(block.URI) {
		return nil, fmt.Errorf("unsupported attachment URL")
	}
	client := http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(block.URI)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("attachment: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (20<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 20<<20 {
		return nil, fmt.Errorf("attachment is too large")
	}
	return data, nil
}

func (s *nativeUI) audioView(c *ui.Context, key string, block conversation.Block) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if s.audioKey == key && s.audio != nil {
			player := s.audio
			label := "Play audio"
			if player.Playing() {
				label = "Pause audio"
				c.After(200 * time.Millisecond)
			}
			ui.Button(c, label).OnClick(func() {
				if player.Playing() {
					player.Pause()
				} else {
					player.Play()
				}
			})
			ui.Button(c, "Stop audio").OnClick(func() { player.Stop() })
			ui.Textf(c, "%s / %s", player.Position().Round(time.Second), player.Duration().Round(time.Second)).FontSize(11).TextColor(c.Theme().TextMuted)
		} else {
			ui.Button(c, "Play audio").Disabled(s.busy).OnClick(func() { s.playAudio(key, block) })
		}
	})
}

func (s *nativeUI) playAudio(key string, block conversation.Block) {
	s.task("Loading audio…", func() (func(), error) {
		data, err := s.attachmentBytes(block)
		if err != nil {
			return nil, err
		}
		player, err := audio.OpenStream(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return func() {
			if s.audio != nil {
				s.audio.Close()
			}
			s.audio, s.audioKey = player, key
			player.OnEnd(func() { s.post(func() {}) })
			player.Play()
		}, nil
	})
}
