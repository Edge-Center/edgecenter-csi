package controller

import (
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	thisInstance  = "63e02977-9bc0-4abc-9689-ae461606c052"
	otherInstance = "a894e30d-a039-42d3-8891-ead0669da5a3"
)

func attachments(serverIDs ...string) []edgecloudV2.Attachment {
	result := make([]edgecloudV2.Attachment, 0, len(serverIDs))
	for _, id := range serverIDs {
		result = append(result, edgecloudV2.Attachment{ServerID: id, Device: "/dev/sd" + id[:1]})
	}
	return result
}

func TestFindExtraAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments []edgecloudV2.Attachment
		wantFound   bool
	}{
		{"no attachments", nil, false},
		{"attached to this instance only", attachments(thisInstance), true},
		{"attached to another instance only", attachments(otherInstance), false},
		{"foreign record first", attachments(otherInstance, thisInstance), true},
		{"our record first", attachments(thisInstance, otherInstance), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachment, found := findAttachment(tt.attachments, thisInstance)
			if found != tt.wantFound {
				t.Fatalf("findAttachment() found = %v, want %v", found, tt.wantFound)
			}
			if found && attachment.ServerID != thisInstance {
				t.Fatalf("findAttachment() returned a record of %q, want %q", attachment.ServerID, thisInstance)
			}
		})
	}
}

func TestForeignServerIDs(t *testing.T) {
	tests := []struct {
		name        string
		attachments []edgecloudV2.Attachment
		want        int
	}{
		{"no attachments", nil, 0},
		{"attached to this instance only", attachments(thisInstance), 0},
		{"attached to another instance only", attachments(otherInstance), 1},
		{"attached to both", attachments(otherInstance, thisInstance), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foreign := foreignServerIDs(tt.attachments, thisInstance)
			if len(foreign) != tt.want {
				t.Fatalf("foreignServerIDs() = %v, want %d records", foreign, tt.want)
			}
			for _, id := range foreign {
				if id == thisInstance {
					t.Fatalf("foreignServerIDs() returned the requested instance %q", id)
				}
			}
		})
	}
}
