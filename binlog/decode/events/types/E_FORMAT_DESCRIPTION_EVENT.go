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

	"github.com/liipx/go-mysql-binlog/binlog/common"
)

// FmtDescEvent is the definition of FORMAT_DESCRIPTION_EVENT
// https://dev.mysql.com/doc/internals/en/format-description-event.html
type FmtDescEvent struct {
	BaseEventBody
	BinlogVersion     int
	MySQLVersion      string
	CreateTime        int64
	EventHeaderLength int64
	EventTypeHeader   []byte

	// cache the result of HasCheckSum()
	HasCheckSum bool
}

func init() {
	Register(new(FmtDescEvent))
}

// GetEventType return base env type
func (e *FmtDescEvent) GetEventType() []uint8 {
	return []uint8{common.FormatDescriptionEvent}
}

func (e *FmtDescEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)

	var pos int
	event := &FmtDescEvent{}

	// binlog-version
	event.BinlogVersion = int(binary.LittleEndian.Uint16(opt.Data))
	pos += 2

	// mysql-server version
	event.MySQLVersion = string(bytes.Trim(opt.Data[pos:pos+50], string(0x00)))
	event.HasCheckSum = common.HasChecksum(event.MySQLVersion)
	pos += 50

	// create timestamp
	event.CreateTime = int64(binary.LittleEndian.Uint32(opt.Data[pos:]))
	pos += 4

	// event header length
	event.EventHeaderLength = int64(opt.Data[pos])
	pos++

	// event type header lengths
	event.EventTypeHeader = opt.Data[pos:]

	opt.Description = event
	return event, nil
}
