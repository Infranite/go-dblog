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
	"strings"

	"github.com/liipx/go-mysql-binlog/binlog/common"
)

// RotateEvent is the definition of ROTATE_EVENT
// https://dev.mysql.com/doc/internals/en/rotate-event.html
// The rotate event is added to the binlog as last event to tell the reader what binlog to request next.
type RotateEvent struct {
	BaseEventBody
	Position uint64
	FileName string
}

func init() {
	Register(new(RotateEvent))
}

// GetEventType return base env type
func (e *RotateEvent) GetEventType() []uint8 {
	return []uint8{common.RotateEvent}
}

func (e *RotateEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)

	event := &RotateEvent{}
	var pos int
	if opt.Description.BinlogVersion > 1 {
		event.Position = binary.LittleEndian.Uint64(opt.Data)
		pos += 8
	}

	event.FileName = strings.TrimSpace(string(opt.Data[pos:]))
	return event, nil
}
