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

type TrainingParticipantStatus string

const (
	TrainingParticipantStatusCompleted   TrainingParticipantStatus = "COMPLETED"
	TrainingParticipantStatusTestPending TrainingParticipantStatus = "TEST_PENDING"
	TrainingParticipantStatusFailed      TrainingParticipantStatus = "FAILED"
	TrainingParticipantStatusNotAttended TrainingParticipantStatus = "NOT_ATTENDED"
)

var (
	_ fmt.Stringer             = TrainingParticipantStatus("")
	_ encoding.TextMarshaler   = TrainingParticipantStatus("")
	_ encoding.TextUnmarshaler = (*TrainingParticipantStatus)(nil)
)

func TrainingParticipantStatuss() []TrainingParticipantStatus {
	return []TrainingParticipantStatus{
		TrainingParticipantStatusCompleted,
		TrainingParticipantStatusTestPending,
		TrainingParticipantStatusFailed,
		TrainingParticipantStatusNotAttended,
	}
}

func (v TrainingParticipantStatus) IsValid() bool {
	for _, value := range TrainingParticipantStatuss() {
		if v == value {
			return true
		}
	}

	return false
}

func (v TrainingParticipantStatus) String() string {
	return string(v)
}

func (v TrainingParticipantStatus) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *TrainingParticipantStatus) UnmarshalText(text []byte) error {
	val := TrainingParticipantStatus(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid TrainingParticipantStatus value: %q", string(text))
	}

	*v = val

	return nil
}
