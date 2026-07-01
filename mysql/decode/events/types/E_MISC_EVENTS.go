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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog/mysql/common"
)

const (
	gtidSIDLength                = 16
	logicalTimestampTypeCode     = 2
	logicalTimestampLength       = 8
	commitTimestampLength        = 7
	immediateTimestampHasOrigin  = uint64(1) << 55
	immediateServerHasOrigin     = uint32(1) << 31
	transactionContextHeaderSize = 18
	viewChangeHeaderSize         = 52
)

func init() {
	Register(new(StartEventV3))
	Register(new(StopEvent))
	Register(new(LoadEvent))
	Register(new(FileEvent))
	Register(new(ExecLoadEvent))
	Register(new(RandEvent))
	Register(new(UserVarEvent))
	Register(new(ExecuteLoadQueryEvent))
	Register(new(IncidentEvent))
	Register(new(HeartbeatEvent))
	Register(new(RowsQueryEvent))
	Register(new(GTIDLogEvent))
	Register(new(PreviousGTIDsEvent))
	Register(new(TransactionContextEvent))
	Register(new(ViewChangeEvent))
	Register(new(XAPrepareLogEvent))
	Register(new(TransactionPayloadEvent))
}

func requireData(data []byte, n int) error {
	if len(data) < n {
		return fmt.Errorf("event data too short: got %d, need %d", len(data), n)
	}
	return nil
}

func trimNULString(data []byte) string {
	return string(bytes.TrimRight(data, "\x00"))
}

func splitNULStrings(data []byte, count int) ([]string, int) {
	out := make([]string, 0, count)
	pos := 0
	for len(out) < count && pos < len(data) {
		next := bytes.IndexByte(data[pos:], 0)
		if next < 0 {
			out = append(out, string(data[pos:]))
			pos = len(data)
			break
		}
		out = append(out, string(data[pos:pos+next]))
		pos += next + 1
	}
	return out, pos
}

type StartEventV3 struct {
	BaseEventBody
	BinlogVersion uint16
	MySQLVersion  string
	CreateTime    uint32
}

func (e *StartEventV3) GetEventType() []uint8 {
	return []uint8{common.StartEventV3}
}

func (e *StartEventV3) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 56); err != nil {
		return nil, err
	}
	return &StartEventV3{
		BaseEventBody: BaseEventBody{data: opt.Data},
		BinlogVersion: binary.LittleEndian.Uint16(opt.Data),
		MySQLVersion:  trimNULString(opt.Data[2:52]),
		CreateTime:    binary.LittleEndian.Uint32(opt.Data[52:]),
	}, nil
}

type StopEvent struct {
	BaseEventBody
	EventType uint8
}

func (e *StopEvent) GetEventType() []uint8 {
	return []uint8{common.StopEvent, common.SlaveEvent, common.IgnorableEvent}
}

func (e *StopEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	return &StopEvent{BaseEventBody: BaseEventBody{data: opt.Data}, EventType: opt.EventType}, nil
}

type LoadEvent struct {
	BaseEventBody
	EventType        uint8
	SlaveProxyID     uint32
	ExecutionTime    uint32
	SkipLines        uint32
	TableNameLength  uint8
	SchemaLength     uint8
	ColumnCount      uint32
	FieldTerminator  []byte
	EnclosedBy       []byte
	LineTerminator   []byte
	LineStart        []byte
	EscapedBy        []byte
	OptFlags         uint8
	EmptyFlags       uint8
	FieldNameLengths []byte
	FieldNames       []string
	TableName        string
	SchemaName       string
	FileName         string
	Payload          []byte
}

func (e *LoadEvent) GetEventType() []uint8 {
	return []uint8{common.LoadEvent, common.NewLoadEvent}
}

func (e *LoadEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 18); err != nil {
		return nil, err
	}
	event := &LoadEvent{BaseEventBody: BaseEventBody{data: opt.Data}, EventType: opt.EventType}
	pos := 0
	event.SlaveProxyID = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.ExecutionTime = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.SkipLines = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.TableNameLength = opt.Data[pos]
	pos++
	event.SchemaLength = opt.Data[pos]
	pos++
	event.ColumnCount = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4

	readFixed := func() []byte {
		v := opt.Data[pos : pos+1]
		pos++
		return v
	}
	readLen := func() ([]byte, error) {
		if err := requireData(opt.Data[pos:], 1); err != nil {
			return nil, err
		}
		n := int(opt.Data[pos])
		pos++
		if err := requireData(opt.Data[pos:], n); err != nil {
			return nil, err
		}
		v := opt.Data[pos : pos+n]
		pos += n
		return v, nil
	}

	var err error
	if opt.EventType == common.NewLoadEvent {
		if event.FieldTerminator, err = readLen(); err != nil {
			return nil, err
		}
		if event.EnclosedBy, err = readLen(); err != nil {
			return nil, err
		}
		if event.LineTerminator, err = readLen(); err != nil {
			return nil, err
		}
		if event.LineStart, err = readLen(); err != nil {
			return nil, err
		}
		if event.EscapedBy, err = readLen(); err != nil {
			return nil, err
		}
	} else {
		if err := requireData(opt.Data[pos:], 7); err != nil {
			return nil, err
		}
		event.FieldTerminator = readFixed()
		event.EnclosedBy = readFixed()
		event.LineTerminator = readFixed()
		event.LineStart = readFixed()
		event.EscapedBy = readFixed()
	}

	need := 1 + int(event.ColumnCount)
	if opt.EventType == common.LoadEvent {
		need++
	}
	if err := requireData(opt.Data[pos:], need); err != nil {
		return nil, err
	}
	event.OptFlags = opt.Data[pos]
	pos++
	if opt.EventType == common.LoadEvent {
		event.EmptyFlags = opt.Data[pos]
		pos++
	}
	if err := requireData(opt.Data[pos:], int(event.ColumnCount)); err != nil {
		return nil, err
	}
	event.FieldNameLengths = opt.Data[pos : pos+int(event.ColumnCount)]
	pos += int(event.ColumnCount)
	var fieldNameBytes int
	event.FieldNames, fieldNameBytes = splitNULStrings(opt.Data[pos:], int(event.ColumnCount))
	pos += fieldNameBytes
	if err := requireData(opt.Data[pos:], int(event.TableNameLength)+1+int(event.SchemaLength)+1); err != nil {
		return nil, err
	}
	event.TableName = trimNULString(opt.Data[pos : pos+int(event.TableNameLength)+1])
	pos += int(event.TableNameLength) + 1
	event.SchemaName = trimNULString(opt.Data[pos : pos+int(event.SchemaLength)+1])
	pos += int(event.SchemaLength) + 1
	event.FileName = trimNULString(opt.Data[pos:])
	event.Payload = opt.Data[pos:]
	return event, nil
}

type FileEvent struct {
	BaseEventBody
	EventType uint8
	FileID    uint32
	BlockData []byte
}

func (e *FileEvent) GetEventType() []uint8 {
	return []uint8{common.CreateFileEvent, common.AppendBlockEvent, common.BeginLoadQueryEvent, common.DeleteFileEvent}
}

func (e *FileEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 4); err != nil {
		return nil, err
	}
	return &FileEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		EventType:     opt.EventType,
		FileID:        binary.LittleEndian.Uint32(opt.Data),
		BlockData:     opt.Data[4:],
	}, nil
}

type ExecLoadEvent struct {
	BaseEventBody
	FileID uint32
}

func (e *ExecLoadEvent) GetEventType() []uint8 {
	return []uint8{common.ExecLoadEvent}
}

func (e *ExecLoadEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 4); err != nil {
		return nil, err
	}
	return &ExecLoadEvent{BaseEventBody: BaseEventBody{data: opt.Data}, FileID: binary.LittleEndian.Uint32(opt.Data)}, nil
}

type RandEvent struct {
	BaseEventBody
	Seed1 uint64
	Seed2 uint64
}

func (e *RandEvent) GetEventType() []uint8 {
	return []uint8{common.RandEvent}
}

func (e *RandEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 16); err != nil {
		return nil, err
	}
	return &RandEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		Seed1:         binary.LittleEndian.Uint64(opt.Data),
		Seed2:         binary.LittleEndian.Uint64(opt.Data[8:]),
	}, nil
}

type UserVarEvent struct {
	BaseEventBody
	Name     string
	IsNull   bool
	Type     uint8
	Charset  uint32
	Value    []byte
	Flags    uint8
	HasFlags bool
}

func (e *UserVarEvent) GetEventType() []uint8 {
	return []uint8{common.UserVarEvent}
}

func (e *UserVarEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 5); err != nil {
		return nil, err
	}
	event := &UserVarEvent{BaseEventBody: BaseEventBody{data: opt.Data}}
	pos := 0
	nameLength := int(binary.LittleEndian.Uint32(opt.Data[pos:]))
	pos += 4
	if err := requireData(opt.Data[pos:], nameLength+1); err != nil {
		return nil, err
	}
	event.Name = string(opt.Data[pos : pos+nameLength])
	pos += nameLength
	event.IsNull = opt.Data[pos] != 0
	pos++
	if event.IsNull {
		return event, nil
	}
	if err := requireData(opt.Data[pos:], 9); err != nil {
		return nil, err
	}
	event.Type = opt.Data[pos]
	pos++
	event.Charset = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	valueLength := int(binary.LittleEndian.Uint32(opt.Data[pos:]))
	pos += 4
	if err := requireData(opt.Data[pos:], valueLength); err != nil {
		return nil, err
	}
	event.Value = opt.Data[pos : pos+valueLength]
	pos += valueLength
	if pos < len(opt.Data) {
		event.Flags = opt.Data[pos]
		event.HasFlags = true
	}
	return event, nil
}

type ExecuteLoadQueryEvent struct {
	BaseEventBody
	SlaveProxyID     uint32
	ExecutionTime    uint32
	SchemaLength     uint8
	ErrorCode        uint16
	StatusVars       []byte
	FileID           uint32
	StartPos         uint32
	EndPos           uint32
	DupHandlingFlags uint8
	Schema           string
	Query            string
}

func (e *ExecuteLoadQueryEvent) GetEventType() []uint8 {
	return []uint8{common.ExecuteLoadQueryEvent}
}

func (e *ExecuteLoadQueryEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 26); err != nil {
		return nil, err
	}
	event := &ExecuteLoadQueryEvent{BaseEventBody: BaseEventBody{data: opt.Data}}
	pos := 0
	event.SlaveProxyID = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.ExecutionTime = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.SchemaLength = opt.Data[pos]
	pos++
	event.ErrorCode = binary.LittleEndian.Uint16(opt.Data[pos:])
	pos += 2
	statusVarsLength := int(binary.LittleEndian.Uint16(opt.Data[pos:]))
	pos += 2
	event.FileID = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.StartPos = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.EndPos = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.DupHandlingFlags = opt.Data[pos]
	pos++
	if err := requireData(opt.Data[pos:], statusVarsLength+int(event.SchemaLength)+1); err != nil {
		return nil, err
	}
	event.StatusVars = opt.Data[pos : pos+statusVarsLength]
	pos += statusVarsLength
	event.Schema = string(opt.Data[pos : pos+int(event.SchemaLength)])
	pos += int(event.SchemaLength) + 1
	event.Query = string(opt.Data[pos:])
	return event, nil
}

type IncidentEvent struct {
	BaseEventBody
	Type    uint16
	Message string
}

func (e *IncidentEvent) GetEventType() []uint8 {
	return []uint8{common.IncidentEvent}
}

func (e *IncidentEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 3); err != nil {
		return nil, err
	}
	messageLength := int(opt.Data[2])
	if err := requireData(opt.Data[3:], messageLength); err != nil {
		return nil, err
	}
	return &IncidentEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		Type:          binary.LittleEndian.Uint16(opt.Data),
		Message:       string(opt.Data[3 : 3+messageLength]),
	}, nil
}

type HeartbeatEvent struct {
	BaseEventBody
	EventType uint8
	LogIdent  string
	Payload   []byte
}

func (e *HeartbeatEvent) GetEventType() []uint8 {
	return []uint8{common.HeartbeatEvent, common.HeartbeatEventV2}
}

func (e *HeartbeatEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	return &HeartbeatEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		EventType:     opt.EventType,
		LogIdent:      trimNULString(opt.Data),
		Payload:       opt.Data,
	}, nil
}

type RowsQueryEvent struct {
	BaseEventBody
	Query string
}

func (e *RowsQueryEvent) GetEventType() []uint8 {
	return []uint8{common.RowsQueryEvent}
}

func (e *RowsQueryEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	return &RowsQueryEvent{BaseEventBody: BaseEventBody{data: opt.Data}, Query: string(opt.Data)}, nil
}

type GTIDLogEvent struct {
	BaseEventBody
	EventType                uint8
	CommitFlag               uint8
	SID                      []byte
	GNO                      int64
	LastCommitted            int64
	SequenceNumber           int64
	ImmediateCommitTimestamp uint64
	OriginalCommitTimestamp  uint64
	TransactionLength        uint64
	ImmediateServerVersion   uint32
	OriginalServerVersion    uint32
	TaggedPayload            []byte
}

func (e *GTIDLogEvent) GetEventType() []uint8 {
	return []uint8{common.GTIDEvent, common.AnonymousGTIDEvent, common.GTIDTaggedLogEvent}
}

func (e *GTIDLogEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 1+gtidSIDLength+8); err != nil {
		return nil, err
	}
	event := &GTIDLogEvent{BaseEventBody: BaseEventBody{data: opt.Data}, EventType: opt.EventType}
	pos := 0
	event.CommitFlag = opt.Data[pos]
	pos++
	event.SID = opt.Data[pos : pos+gtidSIDLength]
	pos += gtidSIDLength
	event.GNO = int64(binary.LittleEndian.Uint64(opt.Data[pos:]))
	pos += 8

	if pos < len(opt.Data) && opt.Data[pos] == logicalTimestampTypeCode {
		pos++
		if err := requireData(opt.Data[pos:], logicalTimestampLength*2); err != nil {
			return nil, err
		}
		event.LastCommitted = int64(binary.LittleEndian.Uint64(opt.Data[pos:]))
		pos += logicalTimestampLength
		event.SequenceNumber = int64(binary.LittleEndian.Uint64(opt.Data[pos:]))
		pos += logicalTimestampLength
		if len(opt.Data[pos:]) >= commitTimestampLength {
			event.ImmediateCommitTimestamp = common.FixedLengthInt(opt.Data[pos : pos+commitTimestampLength])
			pos += commitTimestampLength
			if event.ImmediateCommitTimestamp&immediateTimestampHasOrigin != 0 {
				event.ImmediateCommitTimestamp &^= immediateTimestampHasOrigin
				if err := requireData(opt.Data[pos:], commitTimestampLength); err != nil {
					return nil, err
				}
				event.OriginalCommitTimestamp = common.FixedLengthInt(opt.Data[pos : pos+commitTimestampLength])
				pos += commitTimestampLength
			} else {
				event.OriginalCommitTimestamp = event.ImmediateCommitTimestamp
			}
		}
		if len(opt.Data[pos:]) > 0 {
			var n int
			event.TransactionLength, _, n = common.LengthEncodedInt(opt.Data[pos:])
			pos += n
		}
		if len(opt.Data[pos:]) >= 4 {
			event.ImmediateServerVersion = binary.LittleEndian.Uint32(opt.Data[pos:])
			pos += 4
			if event.ImmediateServerVersion&immediateServerHasOrigin != 0 {
				event.ImmediateServerVersion &^= immediateServerHasOrigin
				if err := requireData(opt.Data[pos:], 4); err != nil {
					return nil, err
				}
				event.OriginalServerVersion = binary.LittleEndian.Uint32(opt.Data[pos:])
				pos += 4
			} else {
				event.OriginalServerVersion = event.ImmediateServerVersion
			}
		}
	}
	if pos < len(opt.Data) {
		event.TaggedPayload = opt.Data[pos:]
	}
	return event, nil
}

func (e *GTIDLogEvent) SIDString() string {
	if len(e.SID) != gtidSIDLength {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(e.SID[0:4]),
		hex.EncodeToString(e.SID[4:6]),
		hex.EncodeToString(e.SID[6:8]),
		hex.EncodeToString(e.SID[8:10]),
		hex.EncodeToString(e.SID[10:]),
	)
}

type GTIDInterval struct {
	Start uint64
	Stop  uint64
}

type PreviousGTIDSet struct {
	SID       string
	Intervals []GTIDInterval
}

type PreviousGTIDsEvent struct {
	BaseEventBody
	Sets []PreviousGTIDSet
}

func (e *PreviousGTIDsEvent) GetEventType() []uint8 {
	return []uint8{common.PreviousGTIDEvent}
}

func (e *PreviousGTIDsEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 8); err != nil {
		return nil, err
	}
	event := &PreviousGTIDsEvent{BaseEventBody: BaseEventBody{data: opt.Data}}
	pos := 0
	sidCount := binary.LittleEndian.Uint64(opt.Data[pos:])
	pos += 8
	event.Sets = make([]PreviousGTIDSet, 0, int(sidCount))
	for i := uint64(0); i < sidCount; i++ {
		if err := requireData(opt.Data[pos:], gtidSIDLength+8); err != nil {
			return nil, err
		}
		sid := fmt.Sprintf("%s-%s-%s-%s-%s",
			hex.EncodeToString(opt.Data[pos:pos+4]),
			hex.EncodeToString(opt.Data[pos+4:pos+6]),
			hex.EncodeToString(opt.Data[pos+6:pos+8]),
			hex.EncodeToString(opt.Data[pos+8:pos+10]),
			hex.EncodeToString(opt.Data[pos+10:pos+16]),
		)
		pos += gtidSIDLength
		intervalCount := binary.LittleEndian.Uint64(opt.Data[pos:])
		pos += 8
		set := PreviousGTIDSet{SID: sid, Intervals: make([]GTIDInterval, 0, int(intervalCount))}
		for j := uint64(0); j < intervalCount; j++ {
			if err := requireData(opt.Data[pos:], 16); err != nil {
				return nil, err
			}
			set.Intervals = append(set.Intervals, GTIDInterval{
				Start: binary.LittleEndian.Uint64(opt.Data[pos:]),
				Stop:  binary.LittleEndian.Uint64(opt.Data[pos+8:]),
			})
			pos += 16
		}
		event.Sets = append(event.Sets, set)
	}
	return event, nil
}

func (e *PreviousGTIDsEvent) String() string {
	sets := make([]string, 0, len(e.Sets))
	for _, set := range e.Sets {
		intervals := make([]string, 0, len(set.Intervals))
		for _, interval := range set.Intervals {
			if interval.Stop == interval.Start+1 {
				intervals = append(intervals, fmt.Sprintf("%d", interval.Start))
			} else {
				intervals = append(intervals, fmt.Sprintf("%d-%d", interval.Start, interval.Stop-1))
			}
		}
		sets = append(sets, set.SID+":"+strings.Join(intervals, ":"))
	}
	return strings.Join(sets, ",")
}

type TransactionContextEvent struct {
	BaseEventBody
	ThreadID  uint32
	GTIDSpec  uint32
	Immediate uint64
	Header    []byte
	Payload   []byte
}

func (e *TransactionContextEvent) GetEventType() []uint8 {
	return []uint8{common.TransactionContextEvent}
}

func (e *TransactionContextEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, transactionContextHeaderSize); err != nil {
		return nil, err
	}
	return &TransactionContextEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		ThreadID:      binary.LittleEndian.Uint32(opt.Data),
		GTIDSpec:      binary.LittleEndian.Uint32(opt.Data[4:]),
		Immediate:     binary.LittleEndian.Uint64(opt.Data[8:]),
		Header:        opt.Data[:transactionContextHeaderSize],
		Payload:       opt.Data[transactionContextHeaderSize:],
	}, nil
}

type ViewChangeEvent struct {
	BaseEventBody
	ViewID  []byte
	Header  []byte
	Payload []byte
}

func (e *ViewChangeEvent) GetEventType() []uint8 {
	return []uint8{common.ViewChangeEvent}
}

func (e *ViewChangeEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, viewChangeHeaderSize); err != nil {
		return nil, err
	}
	return &ViewChangeEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		ViewID:        opt.Data[:viewChangeHeaderSize],
		Header:        opt.Data[:viewChangeHeaderSize],
		Payload:       opt.Data[viewChangeHeaderSize:],
	}, nil
}

type XAPrepareLogEvent struct {
	BaseEventBody
	OnePhase    bool
	FormatID    uint32
	GTRIDLength uint32
	BQUALLength uint32
	GTRID       []byte
	BQUAL       []byte
}

func (e *XAPrepareLogEvent) GetEventType() []uint8 {
	return []uint8{common.XAPrepareLogEvent}
}

func (e *XAPrepareLogEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if err := requireData(opt.Data, 13); err != nil {
		return nil, err
	}
	event := &XAPrepareLogEvent{BaseEventBody: BaseEventBody{data: opt.Data}}
	pos := 0
	event.OnePhase = opt.Data[pos] != 0
	pos++
	event.FormatID = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.GTRIDLength = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	event.BQUALLength = binary.LittleEndian.Uint32(opt.Data[pos:])
	pos += 4
	total := int(event.GTRIDLength + event.BQUALLength)
	if err := requireData(opt.Data[pos:], total); err != nil {
		return nil, err
	}
	event.GTRID = opt.Data[pos : pos+int(event.GTRIDLength)]
	pos += int(event.GTRIDLength)
	event.BQUAL = opt.Data[pos : pos+int(event.BQUALLength)]
	return event, nil
}

type TransactionPayloadEvent struct {
	BaseEventBody
	EventType uint8
	Payload   []byte
}

func (e *TransactionPayloadEvent) GetEventType() []uint8 {
	return []uint8{common.TransactionPayloadEvent}
}

func (e *TransactionPayloadEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	return &TransactionPayloadEvent{BaseEventBody: BaseEventBody{data: opt.Data}, EventType: opt.EventType, Payload: opt.Data}, nil
}
