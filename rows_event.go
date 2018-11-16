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

package binlog

import (
	"encoding/binary"
	"fmt"
	"io"
)

func bitmapByteSize(columnCount int) int {
	return int(columnCount+7) / 8
}

// BinTableMapEvent is the definition of TABLE_MAP_EVENT
// https://dev.mysql.com/doc/internals/en/table-map-event.html
type BinTableMapEvent struct {
	BaseEventBody
	TableID       uint64
	tableIDLen    int
	Flags         uint16
	Schema        string
	Table         string
	ColumnCount   uint64
	ColumnTypeDef []byte
	ColumnMetaDef []uint16
	NullBitmap    []byte
}

// Init BinTableMapEvent tableIDLen
func (e *BinTableMapEvent) Init(h *BinFmtDescEvent) *BinTableMapEvent {
	if int(h.EventTypeHeader[TableMapEvent-1]) == 6 {
		e.tableIDLen = 4
	} else {
		e.tableIDLen = 6
	}
	return e
}

func decodeTableMapEvent(data []byte, h *BinFmtDescEvent) (*BinTableMapEvent, error) {
	event := &BinTableMapEvent{}

	// set table id
	event = event.Init(h)
	pos := event.tableIDLen
	event.TableID = FixedLengthInt(data[:pos])

	// set flags
	event.Flags = binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	// set schema && skip 0x00
	schemaLength := int(data[pos])
	pos++
	event.Schema = string(data[pos : pos+schemaLength])
	pos += schemaLength + 1

	// set table && skip 0x00
	tableLength := int(data[pos])
	pos++
	event.Table = string(data[pos : pos+tableLength])
	pos += tableLength + 1

	// set column count
	var n int
	event.ColumnCount, _, n = LengthEncodedInt(data[pos:])
	pos += n

	// column_type_def (string.var_len)
	// array of column definitions, one byte per field type
	event.ColumnTypeDef = data[pos : pos+int(event.ColumnCount)]
	pos += int(event.ColumnCount)

	// decode column meta
	var err error
	var metaData []byte
	if metaData, _, n, err = LengthEnodedString(data[pos:]); err != nil {
		return nil, err
	}

	if err := event.decodeMeta(metaData); err != nil {
		return nil, err
	}

	pos += n

	// null_bitmap (string.var_len) [len=(column_count + 8) / 7]
	if len(data[pos:]) == bitmapByteSize(int(event.ColumnCount)) {
		event.NullBitmap = data[pos:]
		return event, nil
	}

	return event, io.EOF
}

func (e *BinTableMapEvent) decodeMeta(data []byte) error {
	pos := 0
	e.ColumnMetaDef = make([]uint16, e.ColumnCount)
	for i, t := range e.ColumnTypeDef {
		switch t {
		case MySQLTypeString:
			// real type
			e.ColumnMetaDef[i] = uint16(data[pos]) << 8
			// pack or field length
			e.ColumnMetaDef[i] += uint16(data[pos+1])
			pos += 2
		case MySQLTypeNewDecimal:
			// precision
			e.ColumnMetaDef[i] = uint16(data[pos]) << 8
			// decimals
			e.ColumnMetaDef[i] += uint16(data[pos+1])
			pos += 2
		case MySQLTypeVarString, MySQLTypeVarchar, MySQLTypeBit:
			e.ColumnMetaDef[i] = binary.LittleEndian.Uint16(data[pos:])
			pos += 2
		case MySQLTypeBlob, MySQLTypeDouble, MySQLTypeFloat, MySQLTypeGeometry, MySQLTypeJSON:
			e.ColumnMetaDef[i] = uint16(data[pos])
			pos++
		case MySQLTypeTime2, MySQLTypeDatetime2, MySQLTypeTimestamp2:
			e.ColumnMetaDef[i] = uint16(data[pos])
			pos++
		case MySQLTypeNewDate, MySQLTypeEnum, MySQLTypeSet,
			MySQLTypeTinyBlob, MySQLTypeMediumBlob, MySQLTypeLongBlob:
			return fmt.Errorf("unsupport type in binlog %d", t)
		default:
			e.ColumnMetaDef[i] = 0
		}
	}
	return nil
}

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

// Init BinRowsEvent, adding version and table_id length
func (e *BinRowsEvent) Init(h *BinFmtDescEvent, eventType uint8) *BinRowsEvent {
	if int(h.EventTypeHeader[eventType-1]) == 6 {
		e.tableIDLen = 4
	} else {
		e.tableIDLen = 6
	}

	switch eventType {
	case WriteRowsEventV0, UpdateRowsEventV0, DeleteRowsEventV0:
		e.Version = 0
	case WriteRowsEventV1, UpdateRowsEventV1, DeleteRowsEventV1:
		e.Version = 1
	case WriteRowsEventV2, UpdateRowsEventV2, DeleteRowsEventV2:
		e.Version = 2
	}

	return e
}

func decodeRowsEvent(data []byte, h *BinFmtDescEvent, typ uint8) (*BinRowsEvent, error) {
	event := &BinRowsEvent{}
	event = event.Init(h, typ)

	// set table id
	pos := event.tableIDLen
	event.TableID = FixedLengthInt(data[:pos])

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
	event.ColumnCount, _, n = LengthEncodedInt(data[pos:])
	pos += n

	// columns-present-bitmap1
	bitCount := bitmapByteSize(int(event.ColumnCount))
	event.ColumnsBitmap1 = data[pos : pos+bitCount]
	pos += bitCount

	// columns-present-bitmap2
	if typ == UpdateRowsEventV1 || typ == UpdateRowsEventV2 {
		event.ColumnsBitmap2 = data[pos : pos+bitCount]
		// pos += bitCount
	}

	// TODO Unfinished

	return event, nil
}
