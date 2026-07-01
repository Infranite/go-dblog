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
	"github.com/Infranite/go-dblog/mysql/common"
)

// EventContext global meta information for binary log files
// different versions of the binary log will contain different payload,
// when parsing the log, we need to record these global information
type EventContext struct {
	Description *FmtDescEvent
	TableInfo   map[uint64]*TableMapEvent
	Registry    *EventRegistry
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

// KnowsEventType reports whether the current FORMAT_DESCRIPTION_EVENT declares
// a post-header slot for eventType. This lets the decoder follow newer MySQL
// versions before this package has a first-class decoder for their new events.
func (c *EventContext) KnowsEventType(eventType uint8) bool {
	if c != nil && c.Registry != nil && c.Registry.KnowsEventType(eventType) {
		return true
	}
	return c.EventPostHeaderLength(eventType) >= 0
}

func (c *EventContext) EventPostHeaderLength(eventType uint8) int {
	if c == nil || c.Description == nil || eventType == 0 {
		return -1
	}
	idx := int(eventType) - 1
	if idx >= len(c.Description.EventTypeHeader) {
		return -1
	}
	return int(c.Description.EventTypeHeader[idx])
}

// NewEventContext returns a empty context pointer
func NewEventContext() *EventContext {
	return &EventContext{
		TableInfo: map[uint64]*TableMapEvent{},
		Registry:  DefaultEventRegistry(),
	}
}
