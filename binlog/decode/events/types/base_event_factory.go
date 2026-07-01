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
	"fmt"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
)

// EventRegistry holds event body decoders by event type.
type EventRegistry struct {
	decoders map[uint8]EventBody
	names    map[uint8]string
}

// EventPlugin registers binlog event extensions for a compatible dialect.
type EventPlugin interface {
	Name() string
	Match(*FmtDescEvent) bool
	Register(*EventRegistry)
}

var defaultEventRegistry = NewEventRegistry()

// NewEventRegistry return an empty event registry
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		decoders: make(map[uint8]EventBody),
		names:    make(map[uint8]string),
	}
}

// DefaultEventRegistry return a copy of default event registry
func DefaultEventRegistry() *EventRegistry {
	return defaultEventRegistry.Clone()
}

// Clone return a copy of the event registry
func (r *EventRegistry) Clone() *EventRegistry {
	clone := NewEventRegistry()
	for eventType, decoder := range r.decoders {
		clone.decoders[eventType] = decoder
	}
	for eventType, name := range r.names {
		clone.names[eventType] = name
	}
	return clone
}

// Register add an event body decoder to the registry
func (r *EventRegistry) Register(decoder EventBody) {
	for _, eventType := range decoder.GetEventType() {
		if _, has := r.decoders[eventType]; has {
			panic(fmt.Errorf("EventType {%x} has already been registered", eventType))
		}
		r.decoders[eventType] = decoder
		if name, ok := common.EventType2Str[eventType]; ok {
			r.names[eventType] = name
		}
	}
}

// RegisterName add a display name for an event type
func (r *EventRegistry) RegisterName(eventType uint8, name string) {
	r.names[eventType] = name
}

// GetEventBodyDecoder return decoder for event type
func (r *EventRegistry) GetEventBodyDecoder(eventType uint8) EventBody {
	if decoder, ok := r.decoders[eventType]; ok {
		return decoder
	}
	return &MetadataEvent{EventType: eventType}
}

// KnowsEventType return bool of if event type is known
func (r *EventRegistry) KnowsEventType(eventType uint8) bool {
	if _, ok := r.decoders[eventType]; ok {
		return true
	}
	if _, ok := r.names[eventType]; ok {
		return true
	}
	_, ok := common.EventType2Str[eventType]
	return ok
}

// EventTypeName return event type name
func (r *EventRegistry) EventTypeName(eventType uint8) string {
	if name, ok := r.names[eventType]; ok {
		return name
	}
	return common.EventTypeName(eventType)
}

// Register add an event body decoder to default registry
func Register(decoder EventBody) {
	defaultEventRegistry.Register(decoder)
}

// GetEventBodyDecoder return decoder from default registry
func GetEventBodyDecoder(eventType uint8) EventBody {
	return defaultEventRegistry.GetEventBodyDecoder(eventType)
}
