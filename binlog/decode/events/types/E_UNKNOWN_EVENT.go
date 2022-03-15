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

type UnknownEvent struct {
	BaseEventBody
}

func init() {
	Register(new(UnknownEvent))
}

// GetEventType return base env type
func (e *UnknownEvent) GetEventType() []uint8 {
	return []uint8{common.UnknownEvent}
}
