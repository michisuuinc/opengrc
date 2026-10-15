// Copyright (c) 2026 MICHISUU Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package coredata

import (
	"encoding"
	"fmt"
)

// PersonalDataLifecycleStage is a stage of the personal information
// lifecycle that the Japanese PrivacyMark risk assessment walks through
// (取得・入力, 移送・送信, 利用・加工, 保管・バックアップ, 消去・廃棄).
type PersonalDataLifecycleStage string

const (
	PersonalDataLifecycleStageCollection PersonalDataLifecycleStage = "COLLECTION"
	PersonalDataLifecycleStageTransfer   PersonalDataLifecycleStage = "TRANSFER"
	PersonalDataLifecycleStageUse        PersonalDataLifecycleStage = "USE"
	PersonalDataLifecycleStageStorage    PersonalDataLifecycleStage = "STORAGE"
	PersonalDataLifecycleStageDisposal   PersonalDataLifecycleStage = "DISPOSAL"
)

var (
	_ fmt.Stringer             = PersonalDataLifecycleStage("")
	_ encoding.TextMarshaler   = PersonalDataLifecycleStage("")
	_ encoding.TextUnmarshaler = (*PersonalDataLifecycleStage)(nil)
)

// PersonalDataLifecycleStages returns the stages in lifecycle order.
func PersonalDataLifecycleStages() []PersonalDataLifecycleStage {
	return []PersonalDataLifecycleStage{
		PersonalDataLifecycleStageCollection,
		PersonalDataLifecycleStageTransfer,
		PersonalDataLifecycleStageUse,
		PersonalDataLifecycleStageStorage,
		PersonalDataLifecycleStageDisposal,
	}
}

func (v PersonalDataLifecycleStage) IsValid() bool {
	switch v {
	case
		PersonalDataLifecycleStageCollection,
		PersonalDataLifecycleStageTransfer,
		PersonalDataLifecycleStageUse,
		PersonalDataLifecycleStageStorage,
		PersonalDataLifecycleStageDisposal:
		return true
	}

	return false
}

func (v PersonalDataLifecycleStage) String() string {
	return string(v)
}

func (v PersonalDataLifecycleStage) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *PersonalDataLifecycleStage) UnmarshalText(text []byte) error {
	val := PersonalDataLifecycleStage(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid PersonalDataLifecycleStage value: %q", string(text))
	}

	*v = val

	return nil
}
