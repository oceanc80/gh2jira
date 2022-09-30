// Copyright © 2022 jesus m. rodriguez jmrodri@gmail.com
//
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

package mock

import (
	"encoding/json"
	"errors"
	"net/http"
)

// MustMarshal helper function that wraps json.Marshal
func MustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)

	if err == nil {
		return b
	}

	panic(err)
}

// WriteError helper function to write errors to HTTP handlers
func WriteError(
	w http.ResponseWriter,
	httpStatus int,
	msg string,
) {
	w.WriteHeader(httpStatus)

	w.Write(MustMarshal(errors.New(msg)))
}
