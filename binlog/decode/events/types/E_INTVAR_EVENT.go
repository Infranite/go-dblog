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

// IntvarEvent is the definition of INTVAR_EVENT
// https://dev.mysql.com/doc/internals/en/intvar-event.html
type IntvarEvent struct {
	BaseEventBody
	Type  uint8
	Value uint64
}

func init() {
	Register(new(IntvarEvent))
}

// GetEventType return base env type
func (e *IntvarEvent) GetEventType() []uint8 {
	return []uint8{common.IntvarEvent}
}

func (e *IntvarEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	return &IntvarEvent{
		Type:  opt.Data[0],
		Value: binary.LittleEndian.Uint64(opt.Data[1:]),
	}, nil
}

// TODO: BinIntvarEvent.Type format
