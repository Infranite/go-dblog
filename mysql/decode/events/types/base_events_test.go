package types

import (
	"encoding/binary"
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
)

func TestBaseEventBody(t *testing.T) {
	t.Parallel()

	body, err := new(BaseEventBody).Decode(WithData([]byte{1, 2}))
	if err != nil {
		t.Fatal(err)
	}
	if string(body.Encode()) != "\x01\x02" {
		t.Fatalf("Encode = %v", body.Encode())
	}
}

func TestSmallEventDecoders(t *testing.T) {
	t.Parallel()

	intvarBody, err := new(IntvarEvent).Decode(WithData(append([]byte{1}, le64(8)...)))
	if err != nil {
		t.Fatal(err)
	}
	if intvarBody.(*IntvarEvent).Value != 8 {
		t.Fatalf("intvar = %#v", intvarBody)
	}

	xidBody, err := new(XIDEvent).Decode(WithData(le64(10)))
	if err != nil {
		t.Fatal(err)
	}
	if xidBody.(*XIDEvent).XID != 10 {
		t.Fatalf("xid = %#v", xidBody)
	}

	rotateData := append(le64(4), "mysql-bin.2  "...)
	rotateBody, err := new(RotateEvent).Decode(WithData(rotateData), WithContext(&EventContext{Description: &FmtDescEvent{BinlogVersion: 4}}))
	if err != nil {
		t.Fatal(err)
	}
	if rotateBody.(*RotateEvent).Position != 4 || rotateBody.(*RotateEvent).FileName != "mysql-bin.2" {
		t.Fatalf("rotate = %#v", rotateBody)
	}
}

func TestFormatDescriptionAndContext(t *testing.T) {
	t.Parallel()

	data := make([]byte, 57+common.GTIDTaggedLogEvent)
	binary.LittleEndian.PutUint16(data, 4)
	copy(data[2:], "8.0.36")
	binary.LittleEndian.PutUint32(data[52:], 123)
	data[56] = byte(common.DefaultEventHeaderSize)
	data[57+common.QueryEvent-1] = 13

	ctx := NewEventContext()
	body, err := new(FmtDescEvent).Decode(WithData(data), WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	fde := body.(*FmtDescEvent)
	if fde.BinlogVersion != 4 || fde.MySQLVersion != "8.0.36" || !fde.HasCheckSum {
		t.Fatalf("fde = %#v", fde)
	}
	if !ctx.HasCheckSum() || ctx.GetEventHeaderLength() != common.DefaultEventHeaderSize || !ctx.KnowsEventType(common.QueryEvent) {
		t.Fatalf("context = %#v", ctx)
	}
	if ctx.EventPostHeaderLength(0) != -1 || ctx.EventPostHeaderLength(250) != -1 {
		t.Fatalf("invalid post header lengths")
	}
}

func TestMetadataAndUnsupportedEvents(t *testing.T) {
	t.Parallel()

	ctx := NewEventContext()
	ctx.Description = &FmtDescEvent{EventTypeHeader: make([]byte, 200)}
	ctx.Description.EventTypeHeader[100] = 2
	body, err := (&MetadataEvent{EventType: 101}).Decode(WithData([]byte{1, 2, 3}), WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	metadata := body.(*MetadataEvent)
	if metadata.PostHeaderLength != 2 || string(metadata.PostHeader) != "\x01\x02" || string(metadata.Payload) != "\x03" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if metadata.GetEventType()[0] != 101 {
		t.Fatalf("metadata type = %v", metadata.GetEventType())
	}

	body, err = (&UnsupportedEvent{EventType: 9}).Decode(WithData([]byte{4}), WithEventType(10))
	if err != nil {
		t.Fatal(err)
	}
	if body.(*UnsupportedEvent).EventType != 10 || string(body.Encode()) != "\x04" {
		t.Fatalf("unsupported = %#v", body)
	}
}

func TestEventRegistryAndContext(t *testing.T) {
	t.Parallel()

	registry := NewEventRegistry()
	registry.Register(&testEventBody{eventType: 201})
	registry.RegisterName(202, "CUSTOM")
	if !registry.KnowsEventType(201) || !registry.KnowsEventType(202) || registry.EventTypeName(202) != "CUSTOM" {
		t.Fatalf("registry lookup failed")
	}
	if _, ok := registry.GetEventBodyDecoder(203).(*MetadataEvent); !ok {
		t.Fatalf("unknown decoder type = %T", registry.GetEventBodyDecoder(203))
	}
	clone := registry.Clone()
	clone.RegisterName(203, "CLONE_ONLY")
	if registry.EventTypeName(203) == "CLONE_ONLY" {
		t.Fatal("clone polluted registry")
	}
}
