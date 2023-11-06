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

package util

import (
	"fmt"

	"github.com/jmrodri/gh2jira/internal/config"
	"github.com/spf13/cobra"
)

func ValidateFlags(cmd *cobra.Command) error {
	profilesFile, err := cmd.Flags().GetString("profiles-file")
	if err != nil {
		return err
	}
	profileName, err := cmd.Flags().GetString("profile-name")
	if err != nil {
		return err
	}

	if profilesFile != "" && profileName == "" {
		fmt.Printf("profile-name not specified: using default config\n")
		return nil
	}

	return nil
}

func GetProfiles(cmd *cobra.Command) (*config.Profiles, error) {
	err := ValidateFlags(cmd)
	if err != nil {
		return nil, err
	}

	profilesFile, err := cmd.Flags().GetString("profiles-file")
	if err != nil {
		return nil, err
	}

	if profilesFile == "" {
		return &config.Profiles{}, nil
	}

	return config.ReadProfiles(profilesFile)
}
