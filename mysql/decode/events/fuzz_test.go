package events

import "testing"

func FuzzDecodeEventHeader(f *testing.F) {
	f.Add([]byte{}, 0)
	f.Add([]byte{1, 2, 3}, 3)
	f.Add([]byte{
		1, 0, 0, 0,
		2,
		3, 0, 0, 0,
		19, 0, 0, 0,
		20, 0, 0, 0,
		0, 0,
	}, 19)

	f.Fuzz(func(t *testing.T, data []byte, size int) {
		if size < 0 {
			size = -size
		}
		header, err := DecodeEventHeader(data, int64(size%32))
		if err != nil {
			return
		}
		if header == nil {
			t.Fatal("nil header without error")
		}
		if got := len(header.Data); got != len(data) {
			t.Fatalf("header data length = %d, want %d", got, len(data))
		}
	})
}
