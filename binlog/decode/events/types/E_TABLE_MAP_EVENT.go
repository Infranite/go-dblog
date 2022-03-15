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
	"fmt"
	"io"

	"github.com/liipx/go-mysql-binlog/binlog/common"
)

// TableMapEvent is the definition of TABLE_MAP_EVENT
// https://dev.mysql.com/doc/internals/en/table-map-event.html
type TableMapEvent struct {
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

func init() {
	Register(new(TableMapEvent))
}

// GetEventType return base env type
func (e *TableMapEvent) GetEventType() []uint8 {
	return []uint8{common.TableMapEvent}
}

// init BinTableMapEvent tableIDLen
func (e *TableMapEvent) init(h *FmtDescEvent) *TableMapEvent {
	if int(h.EventTypeHeader[common.TableMapEvent-1]) == 6 {
		e.tableIDLen = 4
	} else {
		e.tableIDLen = 6
	}
	return e
}

func (e *TableMapEvent) decodeMeta(data []byte) error {
	pos := 0
	e.ColumnMetaDef = make([]uint16, e.ColumnCount)
	for i, t := range e.ColumnTypeDef {
		switch t {
		case common.MySQLTypeString:
			// real type
			e.ColumnMetaDef[i] = uint16(data[pos]) << 8
			// pack or field length
			e.ColumnMetaDef[i] += uint16(data[pos+1])
			pos += 2

		case common.MySQLTypeNewDecimal:
			// precision
			e.ColumnMetaDef[i] = uint16(data[pos]) << 8
			// decimals
			e.ColumnMetaDef[i] += uint16(data[pos+1])
			pos += 2

		case common.MySQLTypeVarString, common.MySQLTypeVarchar, common.MySQLTypeBit:
			e.ColumnMetaDef[i] = binary.LittleEndian.Uint16(data[pos:])
			pos += 2

		case common.MySQLTypeBlob, common.MySQLTypeDouble, common.MySQLTypeFloat, common.MySQLTypeGeometry, common.MySQLTypeJSON:
			e.ColumnMetaDef[i] = uint16(data[pos])
			pos++

		case common.MySQLTypeTime2, common.MySQLTypeDatetime2, common.MySQLTypeTimestamp2:
			e.ColumnMetaDef[i] = uint16(data[pos])
			pos++

		case common.MySQLTypeNewDate, common.MySQLTypeEnum, common.MySQLTypeSet,
			common.MySQLTypeTinyBlob, common.MySQLTypeMediumBlob, common.MySQLTypeLongBlob:
			return fmt.Errorf("unsupport type in binlog %d", t)

		default:
			e.ColumnMetaDef[i] = 0
		}
	}
	return nil
}

func (e *TableMapEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	event := &TableMapEvent{}

	// set table id
	event = event.init(opt.Description)
	pos := event.tableIDLen
	event.TableID = common.FixedLengthInt(opt.Data[:pos])

	// set flags
	event.Flags = binary.LittleEndian.Uint16(opt.Data[pos:])
	pos += 2

	// set schema && skip 0x00
	schemaLength := int(opt.Data[pos])
	pos++
	event.Schema = string(opt.Data[pos : pos+schemaLength])
	pos += schemaLength + 1

	// set table && skip 0x00
	tableLength := int(opt.Data[pos])
	pos++
	event.Table = string(opt.Data[pos : pos+tableLength])
	pos += tableLength + 1

	// set column count
	var n int
	event.ColumnCount, _, n = common.LengthEncodedInt(opt.Data[pos:])
	pos += n

	// column_type_def (string.var_len)
	// array of column definitions, one byte per field type
	event.ColumnTypeDef = opt.Data[pos : pos+int(event.ColumnCount)]
	pos += int(event.ColumnCount)

	// decode column meta
	var err error
	var metaData []byte
	if metaData, _, n, err = common.LengthEnodedString(opt.Data[pos:]); err != nil {
		return nil, err
	}

	if err := event.decodeMeta(metaData); err != nil {
		return nil, err
	}

	pos += n

	// null_bitmap (string.var_len) [len=(column_count + 8) / 7]
	if len(opt.Data[pos:]) == common.BitmapByteSize(int(event.ColumnCount)) {
		event.NullBitmap = opt.Data[pos:]
		opt.TableInfo[event.TableID] = event
		return event, nil
	}

	return nil, io.EOF
}
