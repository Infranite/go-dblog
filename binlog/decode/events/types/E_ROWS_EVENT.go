/*
Copyright 2018 liipx(lipengxiang)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"encoding/binary"

	"github.com/liipx/go-mysql-binlog/binlog/common"
)

// BinRowsEvent describe MySQL ROWS_EVENT
// https://dev.mysql.com/doc/internals/en/rows-event.html
type BinRowsEvent struct {
	BaseEventBody

	// header
	Version    int
	TableID    uint64
	tableIDLen int
	Flags      uint16

	// if version == 2
	ExtraData []byte

	// body
	ColumnCount    uint64
	ColumnsBitmap1 []byte

	// if UPDATE_ROWS_EVENTv1 or v2
	ColumnsBitmap2 []byte

	// rows
	// TODO
}

func init() {
	Register(new(BinRowsEvent))
}

// GetEventType return base env type
func (e *BinRowsEvent) GetEventType() []uint8 {
	return []uint8{
		common.WriteRowsEventV0, common.UpdateRowsEventV0, common.DeleteRowsEventV0,
		common.WriteRowsEventV1, common.UpdateRowsEventV1, common.DeleteRowsEventV1,
		common.WriteRowsEventV2, common.UpdateRowsEventV2, common.DeleteRowsEventV2,
	}
}

// Init BinRowsEvent, adding version and table_id length
func (e *BinRowsEvent) Init(h *FmtDescEvent, eventType uint8) *BinRowsEvent {
	if int(h.EventTypeHeader[eventType-1]) == 6 {
		e.tableIDLen = 4
	} else {
		e.tableIDLen = 6
	}

	switch eventType {
	case common.WriteRowsEventV0, common.UpdateRowsEventV0, common.DeleteRowsEventV0:
		e.Version = 0
	case common.WriteRowsEventV1, common.UpdateRowsEventV1, common.DeleteRowsEventV1:
		e.Version = 1
	case common.WriteRowsEventV2, common.UpdateRowsEventV2, common.DeleteRowsEventV2:
		e.Version = 2
	}

	return e
}

func decodeRowsEvent(data []byte, h *FmtDescEvent, typ uint8) (*BinRowsEvent, error) {
	event := &BinRowsEvent{}
	event = event.Init(h, typ)

	// set table id
	pos := event.tableIDLen
	event.TableID = common.FixedLengthInt(data[:pos])

	// set flags
	event.Flags = binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	// set extraDataLength
	if event.Version == 2 {
		extraDataLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2

		event.ExtraData = data[pos : pos+int(extraDataLen-2)]
		pos += int(extraDataLen - 2)
	}

	// body
	var n int
	event.ColumnCount, _, n = common.LengthEncodedInt(data[pos:])
	pos += n

	// columns-present-bitmap1
	bitCount := common.BitmapByteSize(int(event.ColumnCount))
	event.ColumnsBitmap1 = data[pos : pos+bitCount]
	pos += bitCount

	// columns-present-bitmap2
	if typ == common.UpdateRowsEventV1 || typ == common.UpdateRowsEventV2 {
		event.ColumnsBitmap2 = data[pos : pos+bitCount]
		// pos += bitCount
	}

	// TODO Unfinished

	return event, nil
}
