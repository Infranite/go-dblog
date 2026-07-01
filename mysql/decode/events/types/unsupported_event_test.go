package types

import (
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
)

func TestKnownGTIDEventTypeDecodesAsTypedEvent(t *testing.T) {
	t.Parallel()

	bodyDecoder := GetEventBodyDecoder(common.GTIDEvent)
	if bodyDecoder == nil {
		t.Fatal("GTID_EVENT decoder is nil")
	}

	data := []byte{
		1,
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
		10, 0, 0, 0, 0, 0, 0, 0,
	}
	body, err := bodyDecoder.Decode(WithData(data), WithEventType(common.GTIDEvent))
	if err != nil {
		t.Fatal(err)
	}

	gtid, ok := body.(*GTIDLogEvent)
	if !ok {
		t.Fatalf("decoded body = %T, want *GTIDLogEvent", body)
	}
	if gtid.GNO != 10 {
		t.Fatalf("GNO = %d, want 10", gtid.GNO)
	}
}

func TestMetadataEventSplitsPostHeaderFromDescription(t *testing.T) {
	t.Parallel()

	bodyDecoder := GetEventBodyDecoder(43)
	if bodyDecoder == nil {
		t.Fatal("future event decoder is nil")
	}

	ctx := NewEventContext()
	ctx.Description = &FmtDescEvent{EventTypeHeader: make([]byte, 43)}
	ctx.Description.EventTypeHeader[42] = 2

	body, err := bodyDecoder.Decode(
		WithData([]byte{1, 2, 3, 4}),
		WithContext(ctx),
		WithEventType(43),
	)
	if err != nil {
		t.Fatal(err)
	}

	event, ok := body.(*MetadataEvent)
	if !ok {
		t.Fatalf("decoded body = %T, want *MetadataEvent", body)
	}
	if event.PostHeaderLength != 2 || len(event.PostHeader) != 2 || len(event.Payload) != 2 {
		t.Fatalf("metadata split = header %d/%d payload %d", event.PostHeaderLength, len(event.PostHeader), len(event.Payload))
	}
}
