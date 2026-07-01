package types

import "testing"

type testEventBody struct {
	BaseEventBody
	eventType uint8
}

func (e *testEventBody) GetEventType() []uint8 {
	return []uint8{e.eventType}
}

func TestEventRegistryCloneDoesNotPolluteDefaultRegistry(t *testing.T) {
	t.Parallel()

	registry := DefaultEventRegistry()
	registry.Register(&testEventBody{eventType: 160})
	registry.RegisterName(160, "MARIADB_TEST_EVENT")

	if _, ok := registry.GetEventBodyDecoder(160).(*testEventBody); !ok {
		t.Fatalf("custom decoder = %T, want *testEventBody", registry.GetEventBodyDecoder(160))
	}
	if registry.EventTypeName(160) != "MARIADB_TEST_EVENT" {
		t.Fatalf("custom event name = %s", registry.EventTypeName(160))
	}

	globalDecoder := GetEventBodyDecoder(160)
	if _, ok := globalDecoder.(*MetadataEvent); !ok {
		t.Fatalf("global decoder = %T, want *MetadataEvent", globalDecoder)
	}
}
