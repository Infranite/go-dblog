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

// MetadataEvent preserves known but undecoded event payload
type MetadataEvent struct {
	BaseEventBody
	EventType        uint8
	PostHeaderLength int
	PostHeader       []byte
	Payload          []byte
}

func (e *MetadataEvent) GetEventType() []uint8 {
	return []uint8{e.EventType}
}

// Decode split event data into post-header and payload by metadata
func (e *MetadataEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	eventType := e.EventType
	if opt.EventType != 0 {
		eventType = opt.EventType
	}
	event := &MetadataEvent{
		BaseEventBody: BaseEventBody{data: opt.Data},
		EventType:     eventType,
	}
	if opt.EventContext == nil {
		event.PostHeaderLength = -1
		event.Payload = opt.Data
		return event, nil
	}
	event.PostHeaderLength = opt.EventPostHeaderLength(eventType)
	if event.PostHeaderLength <= 0 || event.PostHeaderLength > len(opt.Data) {
		event.Payload = opt.Data
		return event, nil
	}
	event.PostHeader = opt.Data[:event.PostHeaderLength]
	event.Payload = opt.Data[event.PostHeaderLength:]
	return event, nil
}
