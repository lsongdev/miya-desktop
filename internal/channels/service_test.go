package channels

import "testing"

func TestStatusSnapshotDoesNotShareChannelEvents(t *testing.T) {
	service := NewService(nil, nil)
	service.status.ChannelEvents = map[string]ChannelEvent{"telegram": {Channel: "telegram", Status: "connected"}}
	snapshot := service.Status()
	service.status.ChannelEvents["telegram"] = ChannelEvent{Channel: "telegram", Status: "disconnected"}
	if snapshot.ChannelEvents["telegram"].Status != "connected" {
		t.Fatal("status changed after snapshot was returned")
	}
	snapshot.ChannelEvents["new"] = ChannelEvent{}
	if _, exists := service.status.ChannelEvents["new"]; exists {
		t.Fatal("caller modified gateway state through status snapshot")
	}
}
