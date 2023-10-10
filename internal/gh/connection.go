// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gh

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
)

type ConnectionOption func(*Connection) error

type Connection struct {
	transport *http.Client
	client   *github.Client
	token  string
	ctx    context.Context
}

// for unit testing
func WithTransport(t *http.Client) ConnectionOption {
	return func(c *Connection) error {
		c.transport = t
		return nil
	}
}

func WithToken(token string) ConnectionOption {
	return func(c *Connection) error {
		c.token = token
		return nil
	}
}

func WithContext(ctx context.Context) ConnectionOption {
	return func(c *Connection) error {
		c.ctx = ctx
		return nil
	}
}

func WithClient(client *github.Client) ConnectionOption {
	return func(c *Connection) error {
		c.client = client
		return nil
	}
}

func NewConnection(options ...ConnectionOption) (*Connection, error) {
	c := &Connection{}
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Connection) Connect() error {
	if c.transport == nil {
		if c.token == "" {
			return errors.New("cannot create github client without a token")
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.token},
		)
		c.transport = oauth2.NewClient(c.ctx, ts)
		if c.transport == nil {
			return errors.New("transport is not set")
		}
	}
	c.client = github.NewClient(c.transport)
	if c.client == nil {
		return errors.New("client is not set")
	}

	return nil
}

