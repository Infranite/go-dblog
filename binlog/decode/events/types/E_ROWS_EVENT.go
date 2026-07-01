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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
)

// ColumnValue is a decoded column in a rows event.
// Raw references the original event buffer for variable-width values.
type ColumnValue struct {
	Type    uint8
	Meta    uint16
	Null    bool
	Skipped bool
	Value   any
	Raw     []byte
}

// DecimalString return decimal column as string
func (v ColumnValue) DecimalString() (string, bool) {
	s, ok := v.Value.(string)
	return s, ok && v.Type == common.MySQLTypeNewDecimal
}

// Bit return bit column as uint64
func (v ColumnValue) Bit() (uint64, bool) {
	n, ok := v.Value.(uint64)
	return n, ok && v.Type == common.MySQLTypeBit
}

// JSONBinary return JSON binary payload
func (v ColumnValue) JSONBinary() ([]byte, bool) {
	return v.Raw, v.Type == common.MySQLTypeJSON && v.Raw != nil
}

// Geometry return SRID and WKB payload
func (v ColumnValue) Geometry() (uint32, []byte, bool) {
	if v.Type != common.MySQLTypeGeometry || len(v.Raw) < 4 {
		return 0, nil, false
	}
	return binary.LittleEndian.Uint32(v.Raw), v.Raw[4:], true
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
	Rows       [][]ColumnValue
	BeforeRows [][]ColumnValue
	AfterRows  [][]ColumnValue

	// DecodeError records non-fatal metadata errors
	DecodeError string
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
		common.PartialUpdateRowsEvent,
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
	case common.WriteRowsEventV2, common.UpdateRowsEventV2, common.DeleteRowsEventV2, common.PartialUpdateRowsEvent:
		e.Version = 2
	}

	return e
}

func (e *BinRowsEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if opt.Description == nil {
		return nil, fmt.Errorf("invalid binlog version: binary log version info not found")
	}
	if opt.EventType == 0 {
		return nil, fmt.Errorf("empty rows event type")
	}
	var tables map[uint64]*TableMapEvent
	if opt.EventContext != nil {
		tables = opt.TableInfo
	}
	return decodeRowsEvent(opt.Data, opt.Description, tables, opt.EventType)
}

func decodeRowsEvent(data []byte, h *FmtDescEvent, tables map[uint64]*TableMapEvent, typ uint8) (*BinRowsEvent, error) {
	event := &BinRowsEvent{}
	event = event.Init(h, typ)

	// set table id
	pos := event.tableIDLen
	if len(data) < pos+2 {
		return nil, io.ErrUnexpectedEOF
	}
	event.TableID = common.FixedLengthInt(data[:pos])

	// set flags
	event.Flags = binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	// set extraDataLength
	if event.Version == 2 {
		if len(data) < pos+2 {
			return nil, io.ErrUnexpectedEOF
		}
		extraDataLen := binary.LittleEndian.Uint16(data[pos:])
		pos += 2
		if extraDataLen < 2 || len(data) < pos+int(extraDataLen-2) {
			return nil, io.ErrUnexpectedEOF
		}

		event.ExtraData = data[pos : pos+int(extraDataLen-2)]
		pos += int(extraDataLen - 2)
	}

	// body
	var n int
	if len(data) <= pos {
		return nil, io.ErrUnexpectedEOF
	}
	event.ColumnCount, _, n = common.LengthEncodedInt(data[pos:])
	pos += n

	// columns-present-bitmap1
	bitCount := common.BitmapByteSize(int(event.ColumnCount))
	if len(data) < pos+bitCount {
		return nil, io.ErrUnexpectedEOF
	}
	event.ColumnsBitmap1 = data[pos : pos+bitCount]
	pos += bitCount

	// columns-present-bitmap2
	if isUpdateRowsEvent(typ) {
		if len(data) < pos+bitCount {
			return nil, io.ErrUnexpectedEOF
		}
		event.ColumnsBitmap2 = data[pos : pos+bitCount]
		pos += bitCount
	}

	table, ok := tables[event.TableID]
	if !ok {
		event.DecodeError = fmt.Sprintf("invalid table id %d: no corresponding table map event", event.TableID)
		return event, nil
	}
	if table.ColumnCount < event.ColumnCount {
		return nil, fmt.Errorf("table map column count %d is smaller than rows event column count %d", table.ColumnCount, event.ColumnCount)
	}

	for pos < len(data) {
		row, read, err := DecodeRowValues(data[pos:], table, event.ColumnCount, event.ColumnsBitmap1)
		if err != nil {
			return nil, err
		}
		if read == 0 {
			return nil, fmt.Errorf("rows event decoder made no progress")
		}
		pos += read

		if isUpdateRowsEvent(typ) {
			event.BeforeRows = append(event.BeforeRows, row)
			row, read, err = DecodeRowValues(data[pos:], table, event.ColumnCount, event.ColumnsBitmap2)
			if err != nil {
				return nil, err
			}
			if read == 0 {
				return nil, fmt.Errorf("rows event decoder made no progress")
			}
			pos += read
			event.AfterRows = append(event.AfterRows, row)
			continue
		}
		event.Rows = append(event.Rows, row)
	}

	return event, nil
}

func isUpdateRowsEvent(typ uint8) bool {
	return typ == common.UpdateRowsEventV1 ||
		typ == common.UpdateRowsEventV2 ||
		typ == common.PartialUpdateRowsEvent
}

// DecodeRowValues decode one row with table map metadata
func DecodeRowValues(data []byte, table *TableMapEvent, columnCount uint64, bitmap []byte) ([]ColumnValue, int, error) {
	presentColumns := 0
	for i := 0; i < int(columnCount); i++ {
		if bitmapBit(bitmap, i) {
			presentColumns++
		}
	}

	nullBitmapLen := common.BitmapByteSize(presentColumns)
	if len(data) < nullBitmapLen {
		return nil, 0, io.ErrUnexpectedEOF
	}
	nullBitmap := data[:nullBitmapLen]
	pos := nullBitmapLen
	nullIndex := 0

	row := make([]ColumnValue, columnCount)
	for i := 0; i < int(columnCount); i++ {
		value := ColumnValue{
			Type: table.ColumnTypeDef[i],
			Meta: table.ColumnMetaDef[i],
		}
		if !bitmapBit(bitmap, i) {
			value.Skipped = true
			row[i] = value
			continue
		}

		if bitmapBit(nullBitmap, nullIndex) {
			value.Null = true
			nullIndex++
			row[i] = value
			continue
		}
		nullIndex++

		read, err := decodeColumnValue(data[pos:], &value)
		if err != nil {
			return nil, 0, err
		}
		pos += read
		row[i] = value
	}
	return row, pos, nil
}

func decodeColumnValue(data []byte, value *ColumnValue) (int, error) {
	tp := value.Type
	meta := value.Meta
	stringLength := int(meta)

	if tp == common.MySQLTypeString && meta >= 256 {
		b0 := uint8(meta >> 8)
		b1 := uint8(meta)
		if b0&0x30 != 0x30 {
			stringLength = int(uint16(b1) | (uint16((b0&0x30)^0x30) << 4))
			tp = b0 | 0x30
		} else {
			stringLength = int(b1)
			tp = b0
		}
	}

	switch tp {
	case common.MySQLTypeNull:
		value.Null = true
		return 0, nil
	case common.MySQLTypeTiny:
		if len(data) < 1 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = int64(int8(data[0]))
		return 1, nil
	case common.MySQLTypeShort:
		if len(data) < 2 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = int64(int16(binary.LittleEndian.Uint16(data)))
		return 2, nil
	case common.MySQLTypeLong:
		if len(data) < 4 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = int64(int32(binary.LittleEndian.Uint32(data)))
		return 4, nil
	case common.MySQLTypeLonglong:
		if len(data) < 8 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = int64(binary.LittleEndian.Uint64(data))
		return 8, nil
	case common.MySQLTypeInt24:
		if len(data) < 3 {
			return 0, io.ErrUnexpectedEOF
		}
		v := int32(data[0]) | int32(data[1])<<8 | int32(data[2])<<16
		if v&0x800000 != 0 {
			v |= ^0xffffff
		}
		value.Value = int64(v)
		return 3, nil
	case common.MySQLTypeYear:
		if len(data) < 1 {
			return 0, io.ErrUnexpectedEOF
		}
		year := int(data[0])
		if year != 0 {
			year += 1900
		}
		value.Value = year
		return 1, nil
	case common.MySQLTypeFloat:
		if len(data) < 4 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = math.Float32frombits(binary.LittleEndian.Uint32(data))
		return 4, nil
	case common.MySQLTypeDouble:
		if len(data) < 8 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = math.Float64frombits(binary.LittleEndian.Uint64(data))
		return 8, nil
	case common.MySQLTypeTimestamp:
		if len(data) < 4 {
			return 0, io.ErrUnexpectedEOF
		}
		sec := binary.LittleEndian.Uint32(data)
		if sec != 0 {
			value.Value = time.Unix(int64(sec), 0).UTC()
		}
		return 4, nil
	case common.MySQLTypeDate, common.MySQLTypeNewDate:
		if len(data) < 3 {
			return 0, io.ErrUnexpectedEOF
		}
		v := uint32(common.FixedLengthInt(data[:3]))
		if v != 0 {
			value.Value = time.Date(int(v/(16*32)), time.Month(v/32%16), int(v%32), 0, 0, 0, 0, time.UTC)
		}
		return 3, nil
	case common.MySQLTypeDatetime:
		if len(data) < 8 {
			return 0, io.ErrUnexpectedEOF
		}
		v := binary.LittleEndian.Uint64(data)
		if v != 0 {
			d := v / 1000000
			t := v % 1000000
			value.Value = time.Date(
				int(d/10000),
				time.Month((d%10000)/100),
				int(d%100),
				int(t/10000),
				int((t%10000)/100),
				int(t%100),
				0,
				time.UTC,
			)
		}
		return 8, nil
	case common.MySQLTypeTime:
		if len(data) < 3 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = uint32(common.FixedLengthInt(data[:3]))
		return 3, nil
	case common.MySQLTypeTimestamp2:
		return decodeRawFixed(data, value, 4+fractionalBytes(meta))
	case common.MySQLTypeDatetime2:
		return decodeRawFixed(data, value, 5+fractionalBytes(meta))
	case common.MySQLTypeTime2:
		return decodeRawFixed(data, value, 3+fractionalBytes(meta))
	case common.MySQLTypeNewDecimal:
		size, ok := decimalBinarySize(meta)
		if !ok {
			return 0, fmt.Errorf("invalid decimal metadata %d", meta)
		}
		if err := requireData(data, size); err != nil {
			return 0, err
		}
		value.Raw = data[:size]
		value.Value = decodeDecimalString(value.Raw, int(meta>>8), int(meta&0xff))
		return size, nil
	case common.MySQLTypeBit:
		nbits := ((meta >> 8) * 8) + (meta & 0xff)
		size := int(nbits+7) / 8
		if err := requireData(data, size); err != nil {
			return 0, err
		}
		value.Raw = data[:size]
		value.Value = bigEndianUint(value.Raw)
		return size, nil
	case common.MySQLTypeEnum:
		return decodeEnumValue(data, value, int(meta&0xff))
	case common.MySQLTypeSet:
		return decodeRawFixed(data, value, int(meta&0xff))
	case common.MySQLTypeBlob, common.MySQLTypeTinyBlob, common.MySQLTypeMediumBlob,
		common.MySQLTypeLongBlob, common.MySQLTypeJSON, common.MySQLTypeGeometry:
		return decodeLengthPrefixedRaw(data, value, blobLengthSize(tp, meta))
	case common.MySQLTypeVarchar, common.MySQLTypeVarString, common.MySQLTypeString:
		return decodeStringRaw(data, value, stringLength)
	default:
		return 0, fmt.Errorf("unsupported row column type %d", tp)
	}
}

func bitmapBit(bitmap []byte, i int) bool {
	if i < 0 || i/8 >= len(bitmap) {
		return false
	}
	return bitmap[i/8]&(1<<(uint(i)&7)) != 0
}

func decodeRawFixed(data []byte, value *ColumnValue, size int) (int, error) {
	if size < 0 || len(data) < size {
		return 0, io.ErrUnexpectedEOF
	}
	value.Raw = data[:size]
	return size, nil
}

func decodeStringRaw(data []byte, value *ColumnValue, maxLength int) (int, error) {
	lengthSize := 1
	if maxLength > 255 {
		lengthSize = 2
	}
	if len(data) < lengthSize {
		return 0, io.ErrUnexpectedEOF
	}
	length := int(data[0])
	if lengthSize == 2 {
		length = int(binary.LittleEndian.Uint16(data))
	}
	if len(data) < lengthSize+length {
		return 0, io.ErrUnexpectedEOF
	}
	value.Raw = data[lengthSize : lengthSize+length]
	return lengthSize + length, nil
}

func decodeLengthPrefixedRaw(data []byte, value *ColumnValue, lengthSize int) (int, error) {
	if lengthSize < 1 || lengthSize > 4 || len(data) < lengthSize {
		return 0, io.ErrUnexpectedEOF
	}
	length := int(common.FixedLengthInt(data[:lengthSize]))
	if len(data) < lengthSize+length {
		return 0, io.ErrUnexpectedEOF
	}
	value.Raw = data[lengthSize : lengthSize+length]
	return lengthSize + length, nil
}

func decodeEnumValue(data []byte, value *ColumnValue, size int) (int, error) {
	switch size {
	case 1:
		if len(data) < 1 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = uint64(data[0])
		return 1, nil
	case 2:
		if len(data) < 2 {
			return 0, io.ErrUnexpectedEOF
		}
		value.Value = uint64(binary.LittleEndian.Uint16(data))
		return 2, nil
	default:
		return 0, fmt.Errorf("unsupported enum size %d", size)
	}
}

func blobLengthSize(tp uint8, meta uint16) int {
	switch tp {
	case common.MySQLTypeTinyBlob:
		return 1
	case common.MySQLTypeMediumBlob:
		return 3
	case common.MySQLTypeLongBlob:
		return 4
	default:
		return int(meta)
	}
}

func fractionalBytes(dec uint16) int {
	return int(dec+1) / 2
}

func decimalBinarySize(meta uint16) (int, bool) {
	const digitsPerInteger = 9
	compressedBytes := [...]int{0, 1, 1, 2, 2, 3, 3, 4, 4, 4}
	precision := int(meta >> 8)
	scale := int(meta & 0xff)
	if precision < scale {
		return 0, false
	}
	integral := precision - scale
	uncompIntegral := integral / digitsPerInteger
	uncompFractional := scale / digitsPerInteger
	compIntegral := integral - uncompIntegral*digitsPerInteger
	compFractional := scale - uncompFractional*digitsPerInteger
	return uncompIntegral*4 + compressedBytes[compIntegral] +
		uncompFractional*4 + compressedBytes[compFractional], true
}

func decodeDecimalString(data []byte, precision, scale int) string {
	const digitsPerInteger = 9
	compressedBytes := [...]int{0, 1, 1, 2, 2, 3, 3, 4, 4, 4}
	integral := precision - scale
	uncompIntegral := integral / digitsPerInteger
	uncompFractional := scale / digitsPerInteger
	compIntegral := integral - uncompIntegral*digitsPerInteger
	compFractional := scale - uncompFractional*digitsPerInteger

	mask := byte(0)
	if data[0]&0x80 == 0 {
		mask = 0xff
	}
	buf := append([]byte(nil), data...)
	buf[0] ^= 0x80

	var out strings.Builder
	if mask == 0xff {
		out.WriteByte('-')
	}

	pos, n := readDecimalGroup(buf, 0, compressedBytes[compIntegral], mask)
	if n != 0 {
		out.WriteString(strconv.FormatUint(uint64(n), 10))
	}
	wroteIntegral := n != 0
	for i := 0; i < uncompIntegral; i++ {
		_, n = readDecimalGroup(buf, pos, 4, mask)
		pos += 4
		s := strconv.FormatUint(uint64(n), 10)
		if wroteIntegral {
			out.WriteString(strings.Repeat("0", digitsPerInteger-len(s)))
		}
		out.WriteString(s)
		wroteIntegral = true
	}
	if !wroteIntegral {
		out.WriteByte('0')
	}
	if scale == 0 {
		return out.String()
	}

	out.WriteByte('.')
	for i := 0; i < uncompFractional; i++ {
		_, n = readDecimalGroup(buf, pos, 4, mask)
		pos += 4
		s := strconv.FormatUint(uint64(n), 10)
		out.WriteString(strings.Repeat("0", digitsPerInteger-len(s)))
		out.WriteString(s)
	}
	size := compressedBytes[compFractional]
	if size > 0 {
		_, n = readDecimalGroup(buf, pos, size, mask)
		s := strconv.FormatUint(uint64(n), 10)
		out.WriteString(strings.Repeat("0", compFractional-len(s)))
		out.WriteString(s)
	}
	return out.String()
}

func readDecimalGroup(data []byte, pos, size int, mask byte) (int, uint32) {
	var n uint32
	for i := 0; i < size; i++ {
		n = n<<8 | uint32(data[pos+i]^mask)
	}
	return pos + size, n
}

func bigEndianUint(data []byte) uint64 {
	var n uint64
	for _, b := range data {
		n = n<<8 | uint64(b)
	}
	return n
}
