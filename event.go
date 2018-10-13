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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

// BinEventBody descripted event body
type BinEventBody interface {
	isEventBody()
}

// BinEvent binary log event definition
type BinEvent struct {
	Header *BinEventHeader
	Body   BinEventBody
}

// BinEventBodyData save the event data when the event type not supported yet.
type BinEventBodyData []byte

func (body BinEventBodyData) isEventBody() {}

func decodeUnSupportEvent(rd io.Reader, header *BinEventHeader, desc *BinFmtDescEvent) (BinEventBodyData, error) {
	size := header.EventSize - desc.EventHeaderLength
	body, err := ReadNBytes(rd, size)
	if err != nil {
		return nil, err
	}

	return body, err
}

// mysql binlog version > 1 (version > mysql 4.0.0), size = 19
var defaultEventHeaderSize int64 = 19

// BinEventHeader bianry log header definition
// https://dev.mysql.com/doc/internals/en/binlog-event-header.html
type BinEventHeader struct {
	Timestamp int64
	EventType uint8
	ServerID  int64
	EventSize int64
	LogPos    int64
	Flag      uint16
}

func (header *BinEventHeader) String() string {
	return fmt.Sprintf("Time:%s, ServerID:%d, EventSize:%d, LogPos:%d, Flag:0x%x",
		time.Unix(header.Timestamp, 0),
		header.Timestamp,
		header.EventSize,
		header.LogPos,
		header.Flag,
	)
}

func decodeEventHeader(rd io.Reader, desc *BinFmtDescEvent) (*BinEventHeader, error) {
	// set header size
	headerSize := defaultEventHeaderSize
	if desc != nil && desc.EventHeaderLength != defaultEventHeaderSize {
		headerSize = desc.EventHeaderLength
	}

	// read header
	header, err := ReadNBytes(rd, int64(headerSize))
	if err != nil {
		return nil, err
	}

	var pos int
	eventHeader := &BinEventHeader{}

	// timestamp
	eventHeader.Timestamp = int64(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4

	// event_type
	eventHeader.EventType = header[pos]
	pos++

	// serverId
	eventHeader.ServerID = int64(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4

	// event_size
	eventHeader.EventSize = int64(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4

	// version > 2
	if headerSize > 13 {
		// log_pos
		eventHeader.LogPos = int64(binary.LittleEndian.Uint32(header[pos:]))
		pos += 4

		// flags
		eventHeader.Flag = binary.LittleEndian.Uint16(header[pos:])
		pos += 2
	}

	return eventHeader, nil
}

type BaseEvent struct {
	ChecksumType byte
	ChecksumVal  []byte
}

func (event *BaseEvent) isEventBody() {}

// BinFmtDescEvent is the definition of FORMAT_DESCRIPTION_EVENT
// https://dev.mysql.com/doc/internals/en/format-description-event.html
type BinFmtDescEvent struct {
	BaseEvent
	BinlogVersion         int
	MySQLVersion          string
	CreateTime            int64
	EventHeaderLength     int64
	EventTypeHeaderLength []byte
}

func decodeFmtDescEvent(rd io.Reader, header *BinEventHeader) (*BinFmtDescEvent, error) {

	data, err := ReadNBytes(rd, 2+50+4+1)
	if err != nil {
		return nil, err
	}

	var startPos int
	var endPos int
	desc := &BinFmtDescEvent{}

	// binlog-version
	endPos += 2
	desc.BinlogVersion = int(binary.LittleEndian.Uint16(data[startPos:endPos]))

	// mysql-server version
	startPos = endPos
	endPos += 50
	desc.MySQLVersion = string(bytes.Trim(data[startPos:endPos], string(0x00)))

	// create timestamp
	startPos = endPos
	endPos += 4
	desc.CreateTime = int64(binary.LittleEndian.Uint32(data[startPos:endPos]))

	// event header length
	startPos = endPos
	endPos++
	desc.EventHeaderLength = int64(data[startPos])

	// event type header lengths
	bodySize := header.EventSize - desc.EventHeaderLength - 57
	body, err := ReadNBytes(rd, bodySize)
	if err != nil {
		return desc, err
	}

	if hasChecksum(desc.MySQLVersion) {
		index := len(body) - 5
		desc.ChecksumType = body[index]
		desc.ChecksumVal = body[index+1:]
		desc.EventTypeHeaderLength = body[:index]
	} else {
		desc.ChecksumType = BinlogChecksumAlgUndef
		desc.EventTypeHeaderLength = body
	}

	return desc, nil
}

// BinQueryEvent is the definition of QUERY_EVENT
// https://dev.mysql.com/doc/internals/en/query-event.html
type BinQueryEvent struct {
	BaseEvent
	SlaveProxyID     int64
	ExecutionTime    int64
	ErrorCode        uint16
	statusVarsLength int
	StatusVars       []byte
	Schema           string
	Query            string
}

func decodeQueryEvent(rd io.Reader, header *BinEventHeader, desc *BinFmtDescEvent) (*BinQueryEvent, error) {
	// got event body size && read body
	eventSize := header.EventSize - desc.EventHeaderLength
	body, err := ReadNBytes(rd, eventSize)
	if err != nil {
		return nil, err
	}

	var pos int
	event := &BinQueryEvent{}

	// slave_proxy_id
	event.SlaveProxyID = int64(binary.LittleEndian.Uint32(body[pos:]))
	pos += 4

	// execution time
	event.ExecutionTime = int64(binary.LittleEndian.Uint32(body[pos:]))
	pos += 4

	// schema length
	schemaLength := int(uint8(body[pos]))
	pos++

	// error-code
	event.ErrorCode = binary.LittleEndian.Uint16(body[pos:])
	pos += 2

	if desc.BinlogVersion >= 4 {
		// status-vars length
		event.statusVarsLength = int(binary.LittleEndian.Uint16(body[pos:]))
		pos += 2

		// status-vars
		event.StatusVars = body[pos : pos+event.statusVarsLength]
		pos += event.statusVarsLength
	}

	// schema
	event.Schema = string(body[pos : pos+schemaLength])
	pos += schemaLength

	// ignore 0x00
	pos++

	// query
	if desc.ChecksumType != BinlogChecksumAlgUndef {
		event.Query = string(body[pos : len(body)-4])
	} else {
		event.Query = string(body[pos:])
	}

	return event, nil
}

// Statue will format status_vars of QUERY_EVENT
// TODO decode QUERY_EVENT status_var
func (event *BinQueryEvent) Statue() error {
	fmt.Println(event.statusVarsLength)
	for i := 0; i < event.statusVarsLength; {
		// got status_vars key
		k := event.StatusVars[i]
		i++

		// decode values
		switch k {
		case QFlags2Code:
			v := event.StatusVars[i : i+4]
			i += 4

			// TODO
			fmt.Println("QFlags2Code", v)
		case QSQLModeCode:
			v := event.StatusVars[i : i+8]
			i += 8

			// TODO
			fmt.Println("QSQLModeCode", v)
		case QCatalog:
			n := int(event.StatusVars[i])
			v := string(event.StatusVars[i+1 : i+1+n])
			i += 1 + n + 1

			// TODO
			fmt.Println("QCatalog", v)
		case QAutoIncrement:
			increment := binary.LittleEndian.Uint32(event.StatusVars[i:])
			offset := binary.LittleEndian.Uint32(event.StatusVars[i+2:])

			// TODO
			fmt.Printf("QAutoIncrement %d, %d\n", increment, offset)
		case QCharsetCode:
			clientCharSet := event.StatusVars[i : i+2]
			i += 2
			collationConnection := event.StatusVars[i : i+2]
			i += 2
			collationServer := event.StatusVars[i : i+2]
			i += 2

			// TODO
			fmt.Println("QCharsetCode", clientCharSet, collationConnection, collationServer)
		case QTimeZoneCode:
			n := int(event.StatusVars[i])
			v := string(event.StatusVars[i+1 : i+1+n])
			i += 1 + n

			// TODO
			fmt.Println("QTimeZoneCode", v)
		case QCatalogNZCode:
			n := int(event.StatusVars[i])
			v := string(event.StatusVars[i+1 : i+1+n])
			i += 1 + n

			// TODO
			fmt.Printf("QCatalogNZCode %s\n", v)
		case QLCTimeNamesCode:
			// TODO
			fmt.Println("QLCTimeNamesCode")
		case QCharsetDatabaseCode:
			// TODO
			fmt.Println("QCharsetDatabaseCode")
		case QTableMapForUpdateCode:
			// TODO
			fmt.Println("QTableMapForUpdateCode")
		case QMasterDataWrittenCode:
			// TODO
			fmt.Println("QMasterDataWrittenCode")
		case QInvokers:
			// TODO
			fmt.Println("QInvokers")
		case QUpdatedDBNames:
			// TODO
			fmt.Println("QUpdatedDBNames")
		case QMicroseconds:
			microseconds := binary.LittleEndian.Uint32(event.StatusVars[i:])
			i += 3

			// TODO
			fmt.Printf("QMicroseconds %d\n", microseconds)
		default:
			return fmt.Errorf("unknown status var %x", k)
		}
	}

	return nil
}

// BinXIDEvent is the definition of XID_EVENT
// https://dev.mysql.com/doc/internals/en/xid-event.html
// Transaction ID for 2PC, written whenever a COMMIT is expected.
type BinXIDEvent struct {
	BaseEvent
	XID uint64
}

func decodeXIDEvent(rd io.Reader, header *BinEventHeader, desc *BinFmtDescEvent) (*BinXIDEvent, error) {
	body, err := ReadNBytes(rd, header.EventSize-desc.EventHeaderLength)
	if err != nil {
		return nil, err
	}

	event := &BinXIDEvent{
		XID: binary.LittleEndian.Uint64(body),
	}

	if desc.ChecksumType != BinlogChecksumAlgUndef {
		index := len(body) - 5
		event.ChecksumType = body[index]
		event.ChecksumVal = body[index+1:]
	}

	return event, nil
}

// BinIntvarEvent is the definition of INTVAR_EVENT
// https://dev.mysql.com/doc/internals/en/xid-event.html
// Transaction ID for 2PC, written whenever a COMMIT is expected.
type BinIntvarEvent struct {
	BaseEvent
	Type  uint8
	Value uint64
}

func decodeIntvarEvent(rd io.Reader, header *BinEventHeader, desc *BinFmtDescEvent) (*BinIntvarEvent, error) {
	body, err := ReadNBytes(rd, header.EventSize-desc.EventHeaderLength)
	if err != nil {
		return nil, err
	}

	event := &BinIntvarEvent{}
	event.Type = body[0]
	event.Value = binary.LittleEndian.Uint64(body[1:])

	if desc.ChecksumType != BinlogChecksumAlgUndef {
		index := len(body) - 5
		event.ChecksumType = body[index]
		event.ChecksumVal = body[index+1:]
	}

	return event, nil
}

// TODO: BinIntvarEvent.Type format

// BinRotateEvent is the definition of ROTATE_EVENT
// https://dev.mysql.com/doc/internals/en/rotate-event.html
// The rotate event is added to the binlog as last event to tell the reader what binlog to request next.
type BinRotateEvent struct {
	BaseEvent
	Position uint64
	FileName string
}

func decodeRotateEvent(rd io.Reader, desc *BinFmtDescEvent) (*BinRotateEvent, error) {
	event := &BinRotateEvent{}
	if desc.BinlogVersion > 1 {
		header, err := ReadNBytes(rd, 8)
		if err != nil {
			return nil, err
		}
		event.Position = binary.LittleEndian.Uint64(header)
	}

	// cause file name length cant be so long, we could turn uint64 to int64
	name, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	event.FileName = strings.TrimSpace(string(name))
	return event, nil
}

// BinPreGTIDsEvent is the definition of PREVIOUS_GTIDS_EVENT
// TODO: PREVIOUS_GTIDS_EVENT
type BinPreGTIDsEvent struct{ BaseEvent }
