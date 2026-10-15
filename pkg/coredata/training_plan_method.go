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

type TrainingPlanMethod string

const (
	TrainingPlanMethodClassroom TrainingPlanMethod = "CLASSROOM"
	TrainingPlanMethodOnline    TrainingPlanMethod = "ONLINE"
	TrainingPlanMethodELearning TrainingPlanMethod = "E_LEARNING"
	TrainingPlanMethodMaterials TrainingPlanMethod = "MATERIALS"
)

var (
	_ fmt.Stringer             = TrainingPlanMethod("")
	_ encoding.TextMarshaler   = TrainingPlanMethod("")
	_ encoding.TextUnmarshaler = (*TrainingPlanMethod)(nil)
)

func TrainingPlanMethods() []TrainingPlanMethod {
	return []TrainingPlanMethod{
		TrainingPlanMethodClassroom,
		TrainingPlanMethodOnline,
		TrainingPlanMethodELearning,
		TrainingPlanMethodMaterials,
	}
}

func (v TrainingPlanMethod) IsValid() bool {
	for _, value := range TrainingPlanMethods() {
		if v == value {
			return true
		}
	}

	return false
}

func (v TrainingPlanMethod) String() string {
	return string(v)
}

func (v TrainingPlanMethod) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *TrainingPlanMethod) UnmarshalText(text []byte) error {
	val := TrainingPlanMethod(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid TrainingPlanMethod value: %q", string(text))
	}

	*v = val

	return nil
}
