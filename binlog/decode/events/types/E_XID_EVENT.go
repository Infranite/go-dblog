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

// XIDEvent is the definition of XID_EVENT
// https://dev.mysql.com/doc/internals/en/xid-event.html
// Transaction ID for 2PC, written whenever a COMMIT is expected.
type XIDEvent struct {
	BaseEventBody
	XID uint64
}

func init() {
	Register(new(XIDEvent))
}

// GetEventType return base env type
func (e *XIDEvent) GetEventType() []uint8 {
	return []uint8{common.XIDEvent}
}

func (e *XIDEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	return &XIDEvent{
		XID: binary.LittleEndian.Uint64(opt.Data),
	}, nil
}
