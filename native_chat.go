package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"miya-desktop/internal/conversation"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

func (s *nativeUI) chatView(c *ui.Context) {
	t := c.Theme()
	snapshot := s.conversations[s.selected]
	conv := snapshot.Conversation
	title := conv.Title
	if title == "" {
		title = "Conversation"
	}
	ui.Row(c.Key("chat-header")).Padding(16, 24).Gap(12).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Column(c).Grow(1).Gap(3).Children(func() {
			ui.Text(c, title).Bold().FontSize(18)
			detail := conv.Model
			if detail == "" {
				detail = s.selectedAgent
			}
			ui.Text(c, detail).FontSize(11).TextColor(t.TextMuted)
		})
		if s.sending {
			ui.Text(c, "Responding…").TextColor(t.Accent).FontSize(12)
		}
		if s.selected != "" && ui.Button(c, "Copy conversation").Clicked() {
			c.Services().WriteClipboard(conversationText(conv))
		}
	})
	if s.selected == "" {
		ui.Column(c.Key("empty")).Grow(1).Center().Gap(12).Padding(28).Children(func() {
			ui.Text(c, "Start a conversation").FontSize(25).Bold()
			ui.Text(c, "Choose an agent and create a session, or resume a conversation from the sidebar.").MaxWidth(430).TextColor(t.TextMuted)
			ui.PrimaryButton(c, "Create session").Disabled(s.busy || len(s.endpoints) == 0).OnClick(s.newSession)
			if len(s.endpoints) == 0 {
				ui.Button(c, "Configure a provider").OnClick(func() { s.page = "Providers" })
			}
		})
		return
	}
	s.timeline.Key = func(i int) any { return conv.Messages[i].ID }
	s.timeline.FollowEnd = true
	s.timeline.FollowAnimation = &ui.Smooth
	ui.List(c.Key("timeline:"+s.selected), &s.timeline, len(conv.Messages), func(i int) {
		s.messageView(c, conv.Messages[i])
	}).Grow(1).MinHeight(0).Padding(20, 28).Gap(20).Children(func() {
		if len(conv.Messages) == 0 {
			ui.Text(c, "Your conversation will appear here.").TextColor(t.TextMuted).Padding(16)
		}
	})
	ui.Row(c.Key("usage")).Padding(5, 24).Gap(16).Children(func() {
		if conv.Usage != nil {
			ui.Textf(c, "Context: %v / %v tokens", conv.Usage.Used, conv.Usage.Size).FontSize(11).TextColor(t.TextMuted)
		}
		if conv.Mode != nil {
			ui.Textf(c, "Mode: %s", conv.Mode.CurrentModeID).FontSize(11).TextColor(t.TextMuted)
		}
		if snapshot.StopReason != "" {
			ui.Text(c, "Stopped: "+snapshot.StopReason).FontSize(11).TextColor(t.TextMuted)
		}
	})
	ui.Column(c.Key("composer")).Padding(12, 24, 16, 24).Gap(8).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.TextArea(c.Key("message"), &s.draft).Label("Message").Placeholder("Message your agent…").
			Lines(2, 7).Disabled(s.busy || s.sending).OnTextKey(func(k ui.TextKey) bool {
			if k.Key == ui.KeyEnter && (k.Mods == 0 || k.Mods == ui.Cmd) {
				text := k.Text()
				if strings.TrimSpace(text) != "" && !s.busy && !s.sending {
					s.send(text)
					k.SetText("")
				}
				return true
			}
			return false
		})
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "Enter to send · Shift+Enter for a new line").FontSize(11).TextColor(t.TextMuted).Grow(1)
			if s.sending {
				ui.Button(c, "Stop").Disabled(s.cancelling).OnClick(s.cancel)
			} else {
				ui.PrimaryButton(c, "Send").Disabled(s.busy || strings.TrimSpace(s.draft) == "").OnClick(func() { s.send(s.draft) })
			}
		})
	})
}

func (s *nativeUI) messageView(c *ui.Context, message conversation.Message) {
	t := c.Theme()
	ui.Column(c.Key(message.ID)).Gap(7).MinWidth(0).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			role := "Miya"
			if message.Role == conversation.RoleUser {
				role = "You"
			}
			ui.Text(c, role).Bold().FontSize(12)
			if message.Status == conversation.MessageStreaming {
				ui.Text(c, "Streaming").FontSize(11).TextColor(t.Accent)
			}
			if message.Status == conversation.MessageFailed {
				ui.Text(c, "Failed").FontSize(11).TextColor(t.Danger)
			}
			ui.Spacer(c)
			if ui.Button(c, "Copy").Label("Copy message " + message.ID).FontSize(10).Clicked() {
				c.Services().WriteClipboard(messageText(message))
			}
		})
		for _, block := range message.Blocks {
			block := block
			ui.Column(c.Key(block.ID)).MinWidth(0).Gap(5).Children(func() {
				switch block.Type {
				case conversation.BlockText:
					ui.Text(c, block.Content).Selectable()
				case conversation.BlockMarkdown:
					if message.Status == conversation.MessageStreaming {
						markdown.Stream(c, block.Content, &s.md)
					} else {
						markdown.View(c, block.Content, &s.md)
					}
				case conversation.BlockThought:
					open := s.expanded[block.ID]
					if ui.Collapsible(c, "Thoughts", &open, func() { ui.Text(c, block.Content).Selectable().TextColor(t.TextMuted) }).Changed() {
						s.expanded[block.ID] = open
					}
				case conversation.BlockToolCall:
					s.toolView(c, block)
				case conversation.BlockPlan:
					if block.Plan != nil {
						ui.Text(c, "Plan").Bold().FontSize(12)
						for i, entry := range block.Plan.Entries {
							ui.Row(c.Key(i)).Gap(8).Children(func() {
								status := "○"
								if entry.Status == "completed" {
									status = "✓"
								} else if entry.Status == "in_progress" {
									status = "●"
								}
								ui.Text(c, status).TextColor(t.Accent)
								ui.Text(c, entry.Content).Selectable().Grow(1)
								ui.Text(c, string(entry.Status)).FontSize(10).TextColor(t.TextMuted)
							})
						}
					}
				case conversation.BlockImage, conversation.BlockAudio, conversation.BlockResource:
					s.attachmentView(c, block)
				case conversation.BlockError:
					ui.Text(c, block.Content).TextColor(t.Danger).Selectable()
				default:
					if block.Content != "" {
						ui.Text(c, block.Content).Selectable()
					}
				}
			})
		}
	})
}

func (s *nativeUI) toolView(c *ui.Context, block conversation.Block) {
	if block.Tool == nil {
		return
	}
	tool := block.Tool
	t := c.Theme()
	ui.Column(c).Padding(10, 12).Gap(6).Background(t.Surface).Radius(6).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			title := tool.Title
			if title == "" {
				title = string(tool.Kind)
			}
			ui.Text(c, title).Bold().FontSize(12).Grow(1)
			color := t.TextMuted
			if tool.Status == "failed" {
				color = t.Danger
			} else if tool.Status == "completed" {
				color = t.Success
			}
			ui.Text(c, string(tool.Status)).FontSize(11).TextColor(color)
		})
		open := s.expanded[block.ID]
		if ui.Collapsible(c, "Tool details", &open, func() {
			if len(tool.RawInput) > 0 {
				ui.Text(c, "Input").Bold().FontSize(11)
				ui.Text(c, prettyJSON(tool.RawInput)).Selectable().FontSize(11)
			}
			if len(tool.RawOutput) > 0 {
				ui.Text(c, "Output").Bold().FontSize(11)
				ui.Text(c, prettyJSON(tool.RawOutput)).Selectable().FontSize(11)
			}
			if len(tool.Content) > 0 {
				ui.Text(c, prettyJSONValue(tool.Content)).Selectable().FontSize(11)
			}
		}).Changed() {
			s.expanded[block.ID] = open
		}
	})
}

func (s *nativeUI) attachmentView(c *ui.Context, block conversation.Block) {
	name := block.Name
	if name == "" {
		name = block.Type + " attachment"
	}
	cacheKey := s.selected + "/" + block.ID
	if block.Type == conversation.BlockAudio {
		s.audioView(c, cacheKey, block)
	}
	if block.Type == conversation.BlockImage {
		if image := s.images[cacheKey]; image != nil {
			ui.Image(c, image).MaxWidth(600).MaxHeight(380).Fit(ui.Contain).Radius(6)
		} else {
			if _, requested := s.imageErrors[cacheKey]; !requested {
				s.loadAttachmentImage(cacheKey, block)
			}
			ui.Text(c, s.imageErrors[cacheKey]).FontSize(11).TextColor(c.Theme().TextMuted)
		}
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, name).FontSize(12).Grow(1)
		if block.URI != "" {
			ui.Button(c, "Open").Label("Open " + name).OnClick(func() { s.openAttachment(block.URI) })
		}
		if block.Data != "" {
			ui.Button(c, "Save…").Label("Save " + name).OnClick(func() { s.saveAttachment(block) })
		}
	})
}

func (s *nativeUI) openAttachment(target string) {
	s.task("Opening attachment…", func() (func(), error) { return nil, s.backend.OpenAttachment(target) })
}

func (s *nativeUI) loadAttachmentImage(key string, block conversation.Block) {
	s.imageErrors[key] = "Loading image…"
	s.launch(func() {
		data, err := s.attachmentBytes(block)
		var bitmap *ui.Bitmap
		if err == nil {
			bitmap, err = ui.DecodeBitmap(data)
		}
		s.post(func() {
			if err != nil {
				s.imageErrors[key] = err.Error()
			} else {
				s.images[key] = bitmap
				s.imageErrors[key] = ""
			}
		})
	})
}

func decodeAttachment(data string) ([]byte, error) {
	if strings.HasPrefix(data, "data:") {
		_, data, _ = strings.Cut(data, ",")
	}
	if len(data) > 30<<20 {
		return nil, fmt.Errorf("attachment exceeds preview limit")
	}
	return base64.StdEncoding.DecodeString(data)
}

func (s *nativeUI) saveAttachment(block conversation.Block) {
	window := s.win
	s.task("Saving attachment…", func() (func(), error) {
		data, err := decodeAttachment(block.Data)
		if err != nil {
			return nil, err
		}
		name := block.Name
		if name == "" {
			name = "attachment"
		}
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: window, DefaultPath: name})
		if err != nil || path == "" {
			return nil, err
		}
		return nil, os.WriteFile(path, data, 0600)
	})
}

func (s *nativeUI) markdownImage(target string) (ui.ImageSource, error) {
	data, err := s.attachmentBytes(conversation.Block{URI: target})
	if err != nil {
		return nil, err
	}
	return ui.DecodeBitmap(data)
}

func messageText(message conversation.Message) string {
	var parts []string
	for _, block := range message.Blocks {
		if block.Content != "" {
			parts = append(parts, block.Content)
		}
		if block.Tool != nil {
			parts = append(parts, block.Tool.Title+" ("+string(block.Tool.Status)+")")
		}
		if block.Plan != nil {
			for _, entry := range block.Plan.Entries {
				parts = append(parts, "- "+entry.Content+" ["+string(entry.Status)+"]")
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

func conversationText(conv conversation.Conversation) string {
	var parts []string
	for _, message := range conv.Messages {
		parts = append(parts, message.Role+":\n"+messageText(message))
	}
	return strings.Join(parts, "\n\n")
}

func prettyJSON(data []byte) string {
	var formatted bytes.Buffer
	if json.Indent(&formatted, data, "", "  ") == nil {
		return formatted.String()
	}
	return string(data)
}
func prettyJSONValue(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}
