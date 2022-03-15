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

package events

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/liipx/go-mysql-binlog/binlog/common"
)

// mysql binlog version > 1 (version > mysql 4.0.0), size = 19
var defaultEventHeaderSize int64 = 19

// EventHeader binary log header definition
// https://dev.mysql.com/doc/internals/en/binlog-event-header.html
type EventHeader struct {
	Timestamp int64
	EventType uint8
	ServerID  int64
	EventSize int64
	LogPos    int64
	Flag      uint16
	Data      []byte
}

// Type function will translate event type into string
func (header *EventHeader) Type() string {
	return common.EventType2Str[header.EventType]
}

// String interface implement
func (header *EventHeader) String() string {
	return fmt.Sprintf("Type:%s, Time:%s, ServerID:%d, EventSize:%d, EventEndPos:%d, Flag:0x%x",
		header.Type(),
		time.Unix(header.Timestamp, 0),
		header.Timestamp,
		header.EventSize,
		header.LogPos,
		header.Flag,
	)
}

func DecodeEventHeader(data []byte, size int64) (*EventHeader, error) {
	if l := len(data); int64(l) < size {
		return nil, fmt.Errorf("invalid event header size %d, should be %d", l, size)
	}

	var pos int
	eventHeader := &EventHeader{Data: data}

	// timestamp
	eventHeader.Timestamp = int64(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	// event_type
	eventHeader.EventType = data[pos]
	pos++

	// serverId
	eventHeader.ServerID = int64(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	// event_size
	eventHeader.EventSize = int64(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	// version > 2
	if size > 13 {
		// log_pos
		eventHeader.LogPos = int64(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4
		// flags
		eventHeader.Flag = binary.LittleEndian.Uint16(data[pos:])
	}

	return eventHeader, nil
}
