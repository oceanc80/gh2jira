// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package jira

import (
	"errors"

	gojira "github.com/andygrunwald/go-jira"
)

type ConnectionOption func(*Connection) error

type Connection struct {
	transport *gojira.BearerAuthTransport
	client    *gojira.Client
	token     string
	baseUri   string
}

func WithBaseURI(u string) ConnectionOption {
	return func(c *Connection) error {
		c.baseUri = u
		return nil
	}
}

func WithAuthToken(t string) ConnectionOption {
	return func(c *Connection) error {
		c.token = t
		return nil
	}
}

func (c *Connection) BaseUri() string { return c.baseUri }

func NewConnection(options ...ConnectionOption) (*Connection, error) {
	c := &Connection{}
	for _, o := range options {
		if err := o(c); err != nil {
			return nil, err
		}
	}
	if c.token == "" {
		return nil, errors.New("cannot access jira without a token")
	}
	if c.baseUri == "" {
		return nil, errors.New("no base URI for jira")
	}
	c.transport = &gojira.BearerAuthTransport{Token: c.token}

	return c, nil
}

func (c *Connection) Connect() error {
	if c.transport == nil {
		return errors.New("transport is not set")
	}
	if c.client == nil {
		gc, err := gojira.NewClient(c.transport.Client(), c.baseUri)
		if err != nil {
			return err
		}
		if gc == nil {
			return errors.New("unable to create github client")
		}
		c.client = gc
	}
	return nil
}
