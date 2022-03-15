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
)

var eventFactory = map[uint8]EventBody{}

func Register(decoder EventBody) {
	for _, eventType := range decoder.GetEventType() {
		if _, has := eventFactory[eventType]; has {
			panic(fmt.Errorf("EventType {%x} has already been registered", eventType))
		}
		eventFactory[eventType] = decoder
	}
}

func GetEventBodyDecoder(eventType uint8) EventBody {
	return eventFactory[eventType]
}
