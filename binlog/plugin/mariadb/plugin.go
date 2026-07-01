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

package mariadb

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/events/types"
)

const (
	AnnotateRowsEvent         uint8 = 160
	BinlogCheckpointEvent     uint8 = 161
	GTIDEvent                 uint8 = 162
	GTIDListEvent             uint8 = 163
	StartEncryptionEvent      uint8 = 164
	QueryCompressedEvent      uint8 = 165
	WriteRowsCompressedEvent  uint8 = 166
	UpdateRowsCompressedEvent uint8 = 167
	DeleteRowsCompressedEvent uint8 = 168
)

const (
	gtidFlagGroupCommitID = 2
)

type plugin struct{}

// Plugin return MariaDB event plugin
func Plugin() types.EventPlugin {
	return plugin{}
}

func (plugin) Name() string {
	return "mariadb"
}

func (plugin) Match(fde *types.FmtDescEvent) bool {
	return fde != nil && strings.Contains(fde.MySQLVersion, "MariaDB")
}

func (plugin) Register(registry *types.EventRegistry) {
	registry.Register(new(AnnotateRowsLogEvent))
	registry.Register(new(BinlogCheckpointLogEvent))
	registry.Register(new(GTIDLogEvent))
	registry.Register(new(GTIDListLogEvent))
	registry.Register(new(StartEncryptionLogEvent))
	registry.Register(new(QueryCompressedLogEvent))
	registry.Register(new(RowsCompressedLogEvent))

	registry.RegisterName(AnnotateRowsEvent, "MARIADB_ANNOTATE_ROWS_EVENT")
	registry.RegisterName(BinlogCheckpointEvent, "MARIADB_BINLOG_CHECKPOINT_EVENT")
	registry.RegisterName(GTIDEvent, "MARIADB_GTID_EVENT")
	registry.RegisterName(GTIDListEvent, "MARIADB_GTID_LIST_EVENT")
	registry.RegisterName(StartEncryptionEvent, "MARIADB_START_ENCRYPTION_EVENT")
	registry.RegisterName(QueryCompressedEvent, "MARIADB_QUERY_COMPRESSED_EVENT")
	registry.RegisterName(WriteRowsCompressedEvent, "MARIADB_WRITE_ROWS_COMPRESSED_EVENT_V1")
	registry.RegisterName(UpdateRowsCompressedEvent, "MARIADB_UPDATE_ROWS_COMPRESSED_EVENT_V1")
	registry.RegisterName(DeleteRowsCompressedEvent, "MARIADB_DELETE_ROWS_COMPRESSED_EVENT_V1")
}

func requireData(data []byte, n int) error {
	if len(data) < n {
		return fmt.Errorf("event data too short: got %d, need %d", len(data), n)
	}
	return nil
}

// AnnotateRowsLogEvent is the definition of MariaDB annotate rows event
type AnnotateRowsLogEvent struct {
	Raw   []byte
	Query string
}

func (e *AnnotateRowsLogEvent) GetEventType() []uint8 {
	return []uint8{AnnotateRowsEvent}
}

func (e *AnnotateRowsLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	return &AnnotateRowsLogEvent{Raw: opt.Data, Query: string(opt.Data)}, nil
}

func (e *AnnotateRowsLogEvent) Encode() []byte {
	return e.Raw
}

// BinlogCheckpointLogEvent is the definition of MariaDB binlog checkpoint event
type BinlogCheckpointLogEvent struct {
	Raw      []byte
	FileName string
}

func (e *BinlogCheckpointLogEvent) GetEventType() []uint8 {
	return []uint8{BinlogCheckpointEvent}
}

func (e *BinlogCheckpointLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	if err := requireData(opt.Data, 4); err != nil {
		return nil, err
	}
	fileNameLength := int(binary.LittleEndian.Uint32(opt.Data))
	if err := requireData(opt.Data[4:], fileNameLength); err != nil {
		return nil, err
	}
	return &BinlogCheckpointLogEvent{
		Raw:      opt.Data,
		FileName: string(opt.Data[4 : 4+fileNameLength]),
	}, nil
}

func (e *BinlogCheckpointLogEvent) Encode() []byte {
	return e.Raw
}

// GTIDLogEvent is the definition of MariaDB GTID event
type GTIDLogEvent struct {
	Raw       []byte
	Sequence  uint64
	DomainID  uint32
	Flags     uint8
	CommitID  uint64
	ExtraData []byte
}

func (e *GTIDLogEvent) GetEventType() []uint8 {
	return []uint8{GTIDEvent}
}

func (e *GTIDLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	if err := requireData(opt.Data, 19); err != nil {
		return nil, err
	}
	event := &GTIDLogEvent{
		Raw:      opt.Data,
		Sequence: binary.LittleEndian.Uint64(opt.Data),
		DomainID: binary.LittleEndian.Uint32(opt.Data[8:]),
		Flags:    opt.Data[12],
	}
	if event.Flags&gtidFlagGroupCommitID != 0 {
		event.CommitID = binary.LittleEndian.Uint64(opt.Data[13:])
		event.ExtraData = opt.Data[21:]
		return event, nil
	}
	event.ExtraData = opt.Data[13:]
	return event, nil
}

func (e *GTIDLogEvent) Encode() []byte {
	return e.Raw
}

// GTID is the definition of MariaDB GTID list item
type GTID struct {
	DomainID uint32
	ServerID uint32
	Sequence uint64
}

// GTIDListLogEvent is the definition of MariaDB GTID list event
type GTIDListLogEvent struct {
	Raw   []byte
	GTIDs []GTID
}

func (e *GTIDListLogEvent) GetEventType() []uint8 {
	return []uint8{GTIDListEvent}
}

func (e *GTIDListLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	if err := requireData(opt.Data, 4); err != nil {
		return nil, err
	}
	count := int(binary.LittleEndian.Uint32(opt.Data))
	if err := requireData(opt.Data[4:], count*16); err != nil {
		return nil, err
	}
	event := &GTIDListLogEvent{
		Raw:   opt.Data,
		GTIDs: make([]GTID, 0, count),
	}
	pos := 4
	for i := 0; i < count; i++ {
		event.GTIDs = append(event.GTIDs, GTID{
			DomainID: binary.LittleEndian.Uint32(opt.Data[pos:]),
			ServerID: binary.LittleEndian.Uint32(opt.Data[pos+4:]),
			Sequence: binary.LittleEndian.Uint64(opt.Data[pos+8:]),
		})
		pos += 16
	}
	return event, nil
}

func (e *GTIDListLogEvent) Encode() []byte {
	return e.Raw
}

// StartEncryptionLogEvent is the definition of MariaDB start encryption event
type StartEncryptionLogEvent struct {
	Raw        []byte
	Scheme     uint8
	KeyVersion uint32
	Nonce      []byte
}

func (e *StartEncryptionLogEvent) GetEventType() []uint8 {
	return []uint8{StartEncryptionEvent}
}

func (e *StartEncryptionLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	if err := requireData(opt.Data, 17); err != nil {
		return nil, err
	}
	return &StartEncryptionLogEvent{
		Raw:        opt.Data,
		Scheme:     opt.Data[0],
		KeyVersion: binary.LittleEndian.Uint32(opt.Data[1:]),
		Nonce:      opt.Data[5:17],
	}, nil
}

func (e *StartEncryptionLogEvent) Encode() []byte {
	return e.Raw
}

// QueryCompressedLogEvent is the definition of MariaDB compressed query event
type QueryCompressedLogEvent struct {
	Raw                 []byte
	Payload             []byte
	UncompressedPayload []byte
	Query               *types.QueryEvent
}

func (e *QueryCompressedLogEvent) GetEventType() []uint8 {
	return []uint8{QueryCompressedEvent}
}

func (e *QueryCompressedLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	payload, err := inflateZlib(opt.Data)
	if err != nil {
		return nil, err
	}
	event := &QueryCompressedLogEvent{Raw: opt.Data, Payload: opt.Data, UncompressedPayload: payload}
	body, err := new(types.QueryEvent).Decode(types.WithData(payload), types.WithContext(opt.EventContext))
	if err != nil {
		return nil, err
	}
	event.Query = body.(*types.QueryEvent)
	return event, nil
}

func (e *QueryCompressedLogEvent) Encode() []byte {
	return e.Raw
}

// RowsCompressedLogEvent is the definition of MariaDB compressed rows event
type RowsCompressedLogEvent struct {
	Raw                    []byte
	EventType              uint8
	TableID                uint64
	Flags                  uint16
	ColumnCount            uint64
	ColumnsBitmap1         []byte
	ColumnsBitmap2         []byte
	CompressionHeader      uint8
	CompressionAlgorithm   uint8
	CompressionHeaderSize  uint8
	UncompressedLengthData []byte
	CompressedPayload      []byte
	UncompressedPayload    []byte
	Rows                   [][]types.ColumnValue
	BeforeRows             [][]types.ColumnValue
	AfterRows              [][]types.ColumnValue
	DecodeError            string
}

func (e *RowsCompressedLogEvent) GetEventType() []uint8 {
	return []uint8{WriteRowsCompressedEvent, UpdateRowsCompressedEvent, DeleteRowsCompressedEvent}
}

func (e *RowsCompressedLogEvent) Decode(opts ...types.EventOptionFunc) (types.EventBody, error) {
	opt := types.NewOptionWith(opts...)
	if err := requireData(opt.Data, 9); err != nil {
		return nil, err
	}
	event := &RowsCompressedLogEvent{
		Raw:       opt.Data,
		EventType: opt.EventType,
		TableID:   common.FixedLengthInt(opt.Data[:6]),
		Flags:     binary.LittleEndian.Uint16(opt.Data[6:]),
	}
	pos := 8
	var n int
	event.ColumnCount, _, n = common.LengthEncodedInt(opt.Data[pos:])
	pos += n

	bitCount := common.BitmapByteSize(int(event.ColumnCount))
	if err := requireData(opt.Data[pos:], bitCount+1); err != nil {
		return nil, err
	}
	event.ColumnsBitmap1 = opt.Data[pos : pos+bitCount]
	pos += bitCount
	if opt.EventType == UpdateRowsCompressedEvent {
		if err := requireData(opt.Data[pos:], bitCount+1); err != nil {
			return nil, err
		}
		event.ColumnsBitmap2 = opt.Data[pos : pos+bitCount]
		pos += bitCount
	}

	event.CompressionHeader = opt.Data[pos]
	event.CompressionAlgorithm = (event.CompressionHeader & 0x70) >> 4
	event.CompressionHeaderSize = event.CompressionHeader & 0x07
	pos++
	headerSize := int(event.CompressionHeaderSize)
	if err := requireData(opt.Data[pos:], headerSize); err != nil {
		return nil, err
	}
	event.UncompressedLengthData = opt.Data[pos : pos+headerSize]
	pos += headerSize
	event.CompressedPayload = opt.Data[pos:]
	if event.CompressionAlgorithm != 0 {
		return event, nil
	}
	payload, err := inflateZlib(event.CompressedPayload)
	if err != nil {
		return nil, err
	}
	event.UncompressedPayload = payload
	if opt.EventContext == nil || opt.TableInfo == nil {
		event.DecodeError = fmt.Sprintf("invalid table id %d: no corresponding table map event", event.TableID)
		return event, nil
	}
	table, ok := opt.TableInfo[event.TableID]
	if !ok {
		event.DecodeError = fmt.Sprintf("invalid table id %d: no corresponding table map event", event.TableID)
		return event, nil
	}
	for pos = 0; pos < len(payload); {
		row, n, err := types.DecodeRowValues(payload[pos:], table, event.ColumnCount, event.ColumnsBitmap1)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("rows compressed event decoder made no progress")
		}
		pos += n
		if opt.EventType != UpdateRowsCompressedEvent {
			event.Rows = append(event.Rows, row)
			continue
		}
		event.BeforeRows = append(event.BeforeRows, row)
		row, n, err = types.DecodeRowValues(payload[pos:], table, event.ColumnCount, event.ColumnsBitmap2)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("rows compressed event decoder made no progress")
		}
		pos += n
		event.AfterRows = append(event.AfterRows, row)
	}
	return event, nil
}

func (e *RowsCompressedLogEvent) Encode() []byte {
	return e.Raw
}

func inflateZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
