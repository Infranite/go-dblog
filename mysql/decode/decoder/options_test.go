package decoder

import (
	"testing"
	"time"

	"github.com/Infranite/go-dblog/mysql/decode/events"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

type testPlugin struct{}

func (testPlugin) Name() string { return "test" }

func (testPlugin) Match(*types.FmtDescEvent) bool { return false }

func (testPlugin) Register(*types.EventRegistry) {}

func TestDecodeOptions(t *testing.T) {
	t.Parallel()

	header := &events.EventHeader{Timestamp: 20, EventSize: 10, LogPos: 30}
	if !(*BinFileDecodeOption)(nil).NeedStart(header) {
		t.Fatal("nil option should start")
	}
	if (*BinFileDecodeOption)(nil).NeedStop(header) {
		t.Fatal("nil option should not stop")
	}

	option := &BinFileDecodeOption{}
	WithStartPos(25)(option)
	if option.NeedStart(header) {
		t.Fatal("start pos should skip event")
	}
	WithStartPos(20)(option)
	WithEndPos(29)(option)
	if !option.NeedStop(header) {
		t.Fatal("end pos should stop event")
	}
	WithStartTime(time.Unix(21, 0))(option)
	if option.NeedStart(header) {
		t.Fatal("start time should skip event")
	}
	WithEndTime(time.Unix(20, 0))(option)
	if !option.NeedStop(header) {
		t.Fatal("end time should stop event")
	}
	WithEventCompatibilityMode(EventCompatibilityStrict)(option)
	if option.CompatibilityMode != EventCompatibilityStrict {
		t.Fatal("compatibility mode not set")
	}
	WithEventPlugins(testPlugin{})(option)
	if len(option.EventPlugins) != 1 {
		t.Fatal("event plugins not set")
	}
}
