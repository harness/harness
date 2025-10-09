// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors

import (
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type stackTestSuite struct {
	suite.Suite
}

func (c *stackTestSuite) SetupTest() {}

func (c *stackTestSuite) TestFrame() {
	stack := newStack()
	frames := stack.frames()
	c.Equal(len(frames), 4)
	log.Info().Msg(frames.format())
}

func (c *stackTestSuite) TestFormat() {
	stack := newStack()
	frames := stack.frames()
	c.Contains(frames[len(frames)-1].Function, "testing.tRunner")
}

func TestStackTestSuite(t *testing.T) {
	suite.Run(t, &stackTestSuite{})
}
