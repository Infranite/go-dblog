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
	"github.com/liipx/go-mysql-binlog/binlog/common"
)

// EventContext global meta information for binary log files
// different versions of the binary log will contain different payload,
// when parsing the log, we need to record these global information
type EventContext struct {
	Description *FmtDescEvent
	TableInfo   map[uint64]*TableMapEvent
}

func (c *EventContext) HasCheckSum() bool {
	if c == nil || c.Description == nil {
		return false
	}
	return c.Description.HasCheckSum
}

func (c *EventContext) GetEventHeaderLength() int64 {
	if c == nil || c.Description == nil {
		return common.DefaultEventHeaderSize
	}
	return c.Description.EventHeaderLength
}

// NewEventContext returns a empty context pointer
func NewEventContext() *EventContext {
	return &EventContext{
		TableInfo: map[uint64]*TableMapEvent{},
	}
}
