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

// EventBody is the interface for binary events,
// binary file code cloud be decode into a common type.
type EventBody interface {
	GetEventType() []uint8
	Decode(opts ...EventOptionFunc) (EventBody, error)
	Encode() []byte
}

// BaseEventBody is base off all events
type BaseEventBody struct {
	data []byte
}

// GetEventType return base env type
func (event *BaseEventBody) GetEventType() []uint8 {
	panic("Not support")
}

// Decode will decode binary to event structs
func (event *BaseEventBody) Decode(opts ...EventOptionFunc) (EventBody, error) {
	event.data = event.InitOption(opts...).Data
	return event, nil
}

// Encode will encode event structs to binary
func (event *BaseEventBody) Encode() []byte {
	return event.data
}

// InitOption will set options by need
func (event BaseEventBody) InitOption(opts ...EventOptionFunc) *EventOption {
	return NewOptionWith(opts...)
}
