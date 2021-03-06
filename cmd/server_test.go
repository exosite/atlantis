// Copyright 2017 HootSuite Media Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Modified hereafter by contributors to runatlantis/atlantis.

package cmd_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/runatlantis/atlantis/cmd"
	"github.com/runatlantis/atlantis/server"
	. "github.com/runatlantis/atlantis/testing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// passedConfig is set to whatever config ended up being passed to NewServer.
// Used for testing.
var passedConfig server.UserConfig

type ServerCreatorMock struct{}

func (s *ServerCreatorMock) NewServer(userConfig server.UserConfig, config server.Config) (cmd.ServerStarter, error) {
	passedConfig = userConfig
	return &ServerStarterMock{}, nil
}

type ServerStarterMock struct{}

func (s *ServerStarterMock) Start() error {
	return nil
}

func TestExecute_NoConfigFlag(t *testing.T) {
	t.Log("If there is no config flag specified Execute should return nil.")
	c := setupWithDefaults(map[string]interface{}{
		cmd.ConfigFlag: "",
	})
	err := c.Execute()
	Ok(t, err)
}

func TestExecute_ConfigFileExtension(t *testing.T) {
	t.Log("If the config file doesn't have an extension then error.")
	c := setupWithDefaults(map[string]interface{}{
		cmd.ConfigFlag: "does-not-exist",
	})
	err := c.Execute()
	Equals(t, "invalid config: reading does-not-exist: Unsupported Config Type \"\"", err.Error())
}

func TestExecute_ConfigFileMissing(t *testing.T) {
	t.Log("If the config file doesn't exist then error.")
	c := setupWithDefaults(map[string]interface{}{
		cmd.ConfigFlag: "does-not-exist.yaml",
	})
	err := c.Execute()
	Equals(t, "invalid config: reading does-not-exist.yaml: open does-not-exist.yaml: no such file or directory", err.Error())
}

func TestExecute_ConfigFileExists(t *testing.T) {
	t.Log("If the config file exists then there should be no error.")
	tmpFile := tempFile(t, "")
	defer os.Remove(tmpFile) // nolint: errcheck
	c := setupWithDefaults(map[string]interface{}{
		cmd.ConfigFlag: tmpFile,
	})
	err := c.Execute()
	Ok(t, err)
}

func TestExecute_InvalidConfig(t *testing.T) {
	t.Log("If the config file contains invalid yaml there should be an error.")
	tmpFile := tempFile(t, "invalidyaml")
	defer os.Remove(tmpFile) // nolint: errcheck
	c := setupWithDefaults(map[string]interface{}{
		cmd.ConfigFlag: tmpFile,
	})
	err := c.Execute()
	Assert(t, strings.Contains(err.Error(), "unmarshal errors"), "should be an unmarshal error")
}

func TestExecute_RequireRepoWhitelist(t *testing.T) {
	t.Log("If no repo whitelist set should error.")
	c := setup(map[string]interface{}{
		cmd.GHUserFlag:  "user",
		cmd.GHTokenFlag: "token",
	})
	err := c.Execute()
	Assert(t, err != nil, "should be an error")
	Equals(t, "--repo-whitelist must be set for security purposes", err.Error())
}

// Should error if the repo whitelist contained a scheme.
func TestExecute_RepoWhitelistScheme(t *testing.T) {
	c := setup(map[string]interface{}{
		cmd.GHUserFlag:        "user",
		cmd.GHTokenFlag:       "token",
		cmd.RepoWhitelistFlag: "http://github.com/*",
	})
	err := c.Execute()
	Assert(t, err != nil, "should be an error")
	Equals(t, "--repo-whitelist cannot contain ://, should be hostnames only", err.Error())
}

func TestExecute_ValidateLogLevel(t *testing.T) {
	t.Log("Should validate log level.")
	c := setupWithDefaults(map[string]interface{}{
		cmd.LogLevelFlag: "invalid",
	})
	err := c.Execute()
	Assert(t, err != nil, "should be an error")
	Equals(t, "invalid log level: not one of debug, info, warn, error", err.Error())
}

func TestExecute_ValidateSSLConfig(t *testing.T) {
	expErr := "--ssl-key-file and --ssl-cert-file are both required for ssl"
	cases := []struct {
		description string
		flags       map[string]interface{}
		expectError bool
	}{
		{
			"neither option set",
			make(map[string]interface{}),
			false,
		},
		{
			"just ssl-key-file set",
			map[string]interface{}{
				cmd.SSLKeyFileFlag: "file",
			},
			true,
		},
		{
			"just ssl-cert-file set",
			map[string]interface{}{
				cmd.SSLCertFileFlag: "flag",
			},
			true,
		},
		{
			"both flags set",
			map[string]interface{}{
				cmd.SSLCertFileFlag: "cert",
				cmd.SSLKeyFileFlag:  "key",
			},
			false,
		},
	}
	for _, testCase := range cases {
		t.Log("Should validate ssl config when " + testCase.description)
		c := setupWithDefaults(testCase.flags)
		err := c.Execute()
		if testCase.expectError {
			Assert(t, err != nil, "should be an error")
			Equals(t, expErr, err.Error())
		} else {
			Ok(t, err)
		}
	}
}

func TestExecute_ValidateVCSConfig(t *testing.T) {
	expErr := "--gh-user/--gh-token or --gitlab-user/--gitlab-token or --bitbucket-user/--bitbucket-token must be set"
	cases := []struct {
		description string
		flags       map[string]interface{}
		expectError bool
	}{
		{
			"no config set",
			make(map[string]interface{}),
			true,
		},
		{
			"just github token set",
			map[string]interface{}{
				cmd.GHTokenFlag: "token",
			},
			true,
		},
		{
			"just gitlab token set",
			map[string]interface{}{
				cmd.GitlabTokenFlag: "token",
			},
			true,
		},
		{
			"just bitbucket token set",
			map[string]interface{}{
				cmd.BitbucketTokenFlag: "token",
			},
			true,
		},
		{
			"just github user set",
			map[string]interface{}{
				cmd.GHUserFlag: "user",
			},
			true,
		},
		{
			"just gitlab user set",
			map[string]interface{}{
				cmd.GitlabUserFlag: "user",
			},
			true,
		},
		{
			"just bitbucket user set",
			map[string]interface{}{
				cmd.BitbucketUserFlag: "user",
			},
			true,
		},
		{
			"github user and gitlab token set",
			map[string]interface{}{
				cmd.GHUserFlag:      "user",
				cmd.GitlabTokenFlag: "token",
			},
			true,
		},
		{
			"gitlab user and github token set",
			map[string]interface{}{
				cmd.GitlabUserFlag: "user",
				cmd.GHTokenFlag:    "token",
			},
			true,
		},
		{
			"github user and bitbucket token set",
			map[string]interface{}{
				cmd.GHUserFlag:         "user",
				cmd.BitbucketTokenFlag: "token",
			},
			true,
		},
		{
			"github user and github token set and should be successful",
			map[string]interface{}{
				cmd.GHUserFlag:  "user",
				cmd.GHTokenFlag: "token",
			},
			false,
		},
		{
			"gitlab user and gitlab token set and should be successful",
			map[string]interface{}{
				cmd.GitlabUserFlag:  "user",
				cmd.GitlabTokenFlag: "token",
			},
			false,
		},
		{
			"bitbucket user and bitbucket token set and should be successful",
			map[string]interface{}{
				cmd.BitbucketUserFlag:  "user",
				cmd.BitbucketTokenFlag: "token",
			},
			false,
		},
		{
			"all set should be successful",
			map[string]interface{}{
				cmd.GHUserFlag:         "user",
				cmd.GHTokenFlag:        "token",
				cmd.GitlabUserFlag:     "user",
				cmd.GitlabTokenFlag:    "token",
				cmd.BitbucketUserFlag:  "user",
				cmd.BitbucketTokenFlag: "token",
			},
			false,
		},
	}
	for _, testCase := range cases {
		t.Log("Should validate vcs config when " + testCase.description)
		testCase.flags[cmd.RepoWhitelistFlag] = "*"

		c := setup(testCase.flags)
		err := c.Execute()
		if testCase.expectError {
			Assert(t, err != nil, "should be an error")
			Equals(t, expErr, err.Error())
		} else {
			Ok(t, err)
		}
	}
}

func TestExecute_Defaults(t *testing.T) {
	t.Log("Should set the defaults for all unspecified flags.")
	c := setup(map[string]interface{}{
		cmd.GHUserFlag:         "user",
		cmd.GHTokenFlag:        "token",
		cmd.GitlabUserFlag:     "gitlab-user",
		cmd.GitlabTokenFlag:    "gitlab-token",
		cmd.BitbucketUserFlag:  "bitbucket-user",
		cmd.BitbucketTokenFlag: "bitbucket-token",
		cmd.RepoWhitelistFlag:  "*",
	})
	err := c.Execute()
	Ok(t, err)

	// Get our hostname since that's what gets defaulted to
	hostname, err := os.Hostname()
	Ok(t, err)
	Equals(t, "http://"+hostname+":4141", passedConfig.AtlantisURL)
	Equals(t, false, passedConfig.AllowForkPRs)
	Equals(t, false, passedConfig.AllowRepoConfig)

	// Get our home dir since that's what gets defaulted to
	dataDir, err := homedir.Expand("~/.atlantis")
	Ok(t, err)
	Equals(t, dataDir, passedConfig.DataDir)

	Equals(t, "github.com", passedConfig.GithubHostname)
	Equals(t, "token", passedConfig.GithubToken)
	Equals(t, "user", passedConfig.GithubUser)
	Equals(t, "", passedConfig.GithubWebhookSecret)
	Equals(t, "gitlab.com", passedConfig.GitlabHostname)
	Equals(t, "gitlab-token", passedConfig.GitlabToken)
	Equals(t, "gitlab-user", passedConfig.GitlabUser)
	Equals(t, "", passedConfig.GitlabWebhookSecret)
	Equals(t, "https://api.bitbucket.org", passedConfig.BitbucketBaseURL)
	Equals(t, "bitbucket-token", passedConfig.BitbucketToken)
	Equals(t, "bitbucket-user", passedConfig.BitbucketUser)
	Equals(t, "", passedConfig.BitbucketWebhookSecret)
	Equals(t, "info", passedConfig.LogLevel)
	Equals(t, 4141, passedConfig.Port)
	Equals(t, false, passedConfig.RequireApproval)
	Equals(t, false, passedConfig.RequireMergeable)
	Equals(t, "", passedConfig.SSLCertFile)
	Equals(t, "", passedConfig.SSLKeyFile)
	Equals(t, "", passedConfig.TFEToken)
}

func TestExecute_ExpandHomeInDataDir(t *testing.T) {
	t.Log("If ~ is used as a data-dir path, should expand to absolute home path")
	c := setup(map[string]interface{}{
		cmd.GHUserFlag:        "user",
		cmd.GHTokenFlag:       "token",
		cmd.RepoWhitelistFlag: "*",
		cmd.DataDirFlag:       "~/this/is/a/path",
	})
	err := c.Execute()
	Ok(t, err)

	home, err := homedir.Dir()
	Ok(t, err)
	Equals(t, home+"/this/is/a/path", passedConfig.DataDir)
}

func TestExecute_RelativeDataDir(t *testing.T) {
	t.Log("Should convert relative dir to absolute.")
	c := setupWithDefaults(map[string]interface{}{
		cmd.DataDirFlag: "../",
	})

	// Figure out what ../ should be as an absolute path.
	expectedAbsolutePath, err := filepath.Abs("../")
	Ok(t, err)

	err = c.Execute()
	Ok(t, err)
	Equals(t, expectedAbsolutePath, passedConfig.DataDir)
}

func TestExecute_GithubUser(t *testing.T) {
	t.Log("Should remove the @ from the github username if it's passed.")
	c := setup(map[string]interface{}{
		cmd.GHUserFlag:        "@user",
		cmd.GHTokenFlag:       "token",
		cmd.RepoWhitelistFlag: "*",
	})
	err := c.Execute()
	Ok(t, err)

	Equals(t, "user", passedConfig.GithubUser)
}

func TestExecute_GitlabUser(t *testing.T) {
	t.Log("Should remove the @ from the gitlab username if it's passed.")
	c := setup(map[string]interface{}{
		cmd.GitlabUserFlag:    "@user",
		cmd.GitlabTokenFlag:   "token",
		cmd.RepoWhitelistFlag: "*",
	})
	err := c.Execute()
	Ok(t, err)

	Equals(t, "user", passedConfig.GitlabUser)
}

func TestExecute_BitbucketUser(t *testing.T) {
	t.Log("Should remove the @ from the bitbucket username if it's passed.")
	c := setup(map[string]interface{}{
		cmd.BitbucketUserFlag:  "@user",
		cmd.BitbucketTokenFlag: "token",
		cmd.RepoWhitelistFlag:  "*",
	})
	err := c.Execute()
	Ok(t, err)

	Equals(t, "user", passedConfig.BitbucketUser)
}

func TestExecute_Flags(t *testing.T) {
	t.Log("Should use all flags that are set.")
	c := setup(map[string]interface{}{
		cmd.AtlantisURLFlag:            "url",
		cmd.AllowForkPRsFlag:           true,
		cmd.AllowRepoConfigFlag:        true,
		cmd.BitbucketBaseURLFlag:       "https://bitbucket-base-url.com",
		cmd.BitbucketTokenFlag:         "bitbucket-token",
		cmd.BitbucketUserFlag:          "bitbucket-user",
		cmd.BitbucketWebhookSecretFlag: "bitbucket-secret",
		cmd.DataDirFlag:                "/path",
		cmd.GHHostnameFlag:             "ghhostname",
		cmd.GHTokenFlag:                "token",
		cmd.GHUserFlag:                 "user",
		cmd.GHWebhookSecretFlag:        "secret",
		cmd.GitlabHostnameFlag:         "gitlab-hostname",
		cmd.GitlabTokenFlag:            "gitlab-token",
		cmd.GitlabUserFlag:             "gitlab-user",
		cmd.GitlabWebhookSecretFlag:    "gitlab-secret",
		cmd.LogLevelFlag:               "debug",
		cmd.PortFlag:                   8181,
		cmd.RepoWhitelistFlag:          "github.com/runatlantis/atlantis",
		cmd.RequireApprovalFlag:        true,
		cmd.RequireMergeableFlag:       true,
		cmd.SSLCertFileFlag:            "cert-file",
		cmd.SSLKeyFileFlag:             "key-file",
		cmd.TFETokenFlag:               "my-token",
	})
	err := c.Execute()
	Ok(t, err)

	Equals(t, "url", passedConfig.AtlantisURL)
	Equals(t, true, passedConfig.AllowForkPRs)
	Equals(t, true, passedConfig.AllowRepoConfig)
	Equals(t, "https://bitbucket-base-url.com", passedConfig.BitbucketBaseURL)
	Equals(t, "bitbucket-token", passedConfig.BitbucketToken)
	Equals(t, "bitbucket-user", passedConfig.BitbucketUser)
	Equals(t, "bitbucket-secret", passedConfig.BitbucketWebhookSecret)
	Equals(t, "/path", passedConfig.DataDir)
	Equals(t, "ghhostname", passedConfig.GithubHostname)
	Equals(t, "token", passedConfig.GithubToken)
	Equals(t, "user", passedConfig.GithubUser)
	Equals(t, "secret", passedConfig.GithubWebhookSecret)
	Equals(t, "gitlab-hostname", passedConfig.GitlabHostname)
	Equals(t, "gitlab-token", passedConfig.GitlabToken)
	Equals(t, "gitlab-user", passedConfig.GitlabUser)
	Equals(t, "gitlab-secret", passedConfig.GitlabWebhookSecret)
	Equals(t, "debug", passedConfig.LogLevel)
	Equals(t, 8181, passedConfig.Port)
	Equals(t, "github.com/runatlantis/atlantis", passedConfig.RepoWhitelist)
	Equals(t, true, passedConfig.RequireApproval)
	Equals(t, true, passedConfig.RequireMergeable)
	Equals(t, "cert-file", passedConfig.SSLCertFile)
	Equals(t, "key-file", passedConfig.SSLKeyFile)
	Equals(t, "my-token", passedConfig.TFEToken)
}

func TestExecute_ConfigFile(t *testing.T) {
	t.Log("Should use all the values from the config file.")
	tmpFile := tempFile(t, `---
atlantis-url: "url"
allow-fork-prs: true
allow-repo-config: true
bitbucket-base-url: "https://mydomain.com"
bitbucket-token: "bitbucket-token"
bitbucket-user: "bitbucket-user"
bitbucket-webhook-secret: "bitbucket-secret"
data-dir: "/path"
gh-hostname: "ghhostname"
gh-token: "token"
gh-user: "user"
gh-webhook-secret: "secret"
gitlab-hostname: "gitlab-hostname"
gitlab-token: "gitlab-token"
gitlab-user: "gitlab-user"
gitlab-webhook-secret: "gitlab-secret"
log-level: "debug"
port: 8181
repo-whitelist: "github.com/runatlantis/atlantis"
require-approval: true
require-mergeable: true
ssl-cert-file: cert-file
ssl-key-file: key-file
tfe-token: my-token
`)
	defer os.Remove(tmpFile) // nolint: errcheck
	c := setup(map[string]interface{}{
		cmd.ConfigFlag: tmpFile,
	})

	err := c.Execute()
	Ok(t, err)
	Equals(t, "url", passedConfig.AtlantisURL)
	Equals(t, true, passedConfig.AllowForkPRs)
	Equals(t, true, passedConfig.AllowRepoConfig)
	Equals(t, "https://mydomain.com", passedConfig.BitbucketBaseURL)
	Equals(t, "bitbucket-token", passedConfig.BitbucketToken)
	Equals(t, "bitbucket-user", passedConfig.BitbucketUser)
	Equals(t, "bitbucket-secret", passedConfig.BitbucketWebhookSecret)
	Equals(t, "/path", passedConfig.DataDir)
	Equals(t, "ghhostname", passedConfig.GithubHostname)
	Equals(t, "token", passedConfig.GithubToken)
	Equals(t, "user", passedConfig.GithubUser)
	Equals(t, "secret", passedConfig.GithubWebhookSecret)
	Equals(t, "gitlab-hostname", passedConfig.GitlabHostname)
	Equals(t, "gitlab-token", passedConfig.GitlabToken)
	Equals(t, "gitlab-user", passedConfig.GitlabUser)
	Equals(t, "gitlab-secret", passedConfig.GitlabWebhookSecret)
	Equals(t, "debug", passedConfig.LogLevel)
	Equals(t, 8181, passedConfig.Port)
	Equals(t, "github.com/runatlantis/atlantis", passedConfig.RepoWhitelist)
	Equals(t, true, passedConfig.RequireApproval)
	Equals(t, true, passedConfig.RequireMergeable)
	Equals(t, "cert-file", passedConfig.SSLCertFile)
	Equals(t, "key-file", passedConfig.SSLKeyFile)
	Equals(t, "my-token", passedConfig.TFEToken)
}

func TestExecute_EnvironmentOverride(t *testing.T) {
	t.Log("Environment variables should override config file flags.")
	tmpFile := tempFile(t, `---
atlantis-url: "url"
allow-fork-prs: true
allow-repo-config: true
bitbucket-base-url: "https://mydomain.com"
bitbucket-token: "bitbucket-token"
bitbucket-user: "bitbucket-user"
bitbucket-webhook-secret: "bitbucket-secret"
data-dir: "/path"
gh-hostname: "ghhostname"
gh-token: "token"
gh-user: "user"
gh-webhook-secret: "secret"
gitlab-hostname: "gitlab-hostname"
gitlab-token: "gitlab-token"
gitlab-user: "gitlab-user"
gitlab-webhook-secret: "gitlab-secret"
log-level: "debug"
port: 8181
repo-whitelist: "github.com/runatlantis/atlantis"
require-approval: true
ssl-cert-file: cert-file
ssl-key-file: key-file
ssl-key-file: my-token
`)
	defer os.Remove(tmpFile) // nolint: errcheck

	// NOTE: We add the ATLANTIS_ prefix below.
	for name, value := range map[string]string{
		"ATLANTIS_URL":             "override-url",
		"ALLOW_FORK_PRS":           "false",
		"ALLOW_REPO_CONFIG":        "false",
		"BITBUCKET_BASE_URL":       "https://override-bitbucket-base-url",
		"BITBUCKET_TOKEN":          "override-bitbucket-token",
		"BITBUCKET_USER":           "override-bitbucket-user",
		"BITBUCKET_WEBHOOK_SECRET": "override-bitbucket-secret",
		"DATA_DIR":                 "/override-path",
		"GH_HOSTNAME":              "override-gh-hostname",
		"GH_TOKEN":                 "override-gh-token",
		"GH_USER":                  "override-gh-user",
		"GH_WEBHOOK_SECRET":        "override-gh-webhook-secret",
		"GITLAB_HOSTNAME":          "override-gitlab-hostname",
		"GITLAB_TOKEN":             "override-gitlab-token",
		"GITLAB_USER":              "override-gitlab-user",
		"GITLAB_WEBHOOK_SECRET":    "override-gitlab-webhook-secret",
		"LOG_LEVEL":                "info",
		"PORT":                     "8282",
		"REPO_WHITELIST":           "override,override",
		"REQUIRE_APPROVAL":         "false",
		"REQUIRE_MERGEABLE":        "false",
		"SSL_CERT_FILE":            "override-cert-file",
		"SSL_KEY_FILE":             "override-key-file",
		"TFE_TOKEN":                "override-my-token",
	} {
		os.Setenv("ATLANTIS_"+name, value) // nolint: errcheck
	}
	c := setup(map[string]interface{}{
		cmd.ConfigFlag: tmpFile,
	})
	err := c.Execute()
	Ok(t, err)
	Equals(t, "override-url", passedConfig.AtlantisURL)
	Equals(t, false, passedConfig.AllowForkPRs)
	Equals(t, false, passedConfig.AllowRepoConfig)
	Equals(t, "https://override-bitbucket-base-url", passedConfig.BitbucketBaseURL)
	Equals(t, "override-bitbucket-token", passedConfig.BitbucketToken)
	Equals(t, "override-bitbucket-user", passedConfig.BitbucketUser)
	Equals(t, "override-bitbucket-secret", passedConfig.BitbucketWebhookSecret)
	Equals(t, "/override-path", passedConfig.DataDir)
	Equals(t, "override-gh-hostname", passedConfig.GithubHostname)
	Equals(t, "override-gh-token", passedConfig.GithubToken)
	Equals(t, "override-gh-user", passedConfig.GithubUser)
	Equals(t, "override-gh-webhook-secret", passedConfig.GithubWebhookSecret)
	Equals(t, "override-gitlab-hostname", passedConfig.GitlabHostname)
	Equals(t, "override-gitlab-token", passedConfig.GitlabToken)
	Equals(t, "override-gitlab-user", passedConfig.GitlabUser)
	Equals(t, "override-gitlab-webhook-secret", passedConfig.GitlabWebhookSecret)
	Equals(t, "info", passedConfig.LogLevel)
	Equals(t, 8282, passedConfig.Port)
	Equals(t, "override,override", passedConfig.RepoWhitelist)
	Equals(t, false, passedConfig.RequireApproval)
	Equals(t, false, passedConfig.RequireMergeable)
	Equals(t, "override-cert-file", passedConfig.SSLCertFile)
	Equals(t, "override-key-file", passedConfig.SSLKeyFile)
	Equals(t, "override-my-token", passedConfig.TFEToken)
}

func TestExecute_FlagConfigOverride(t *testing.T) {
	t.Log("Flags should override config file flags.")
	tmpFile := tempFile(t, `---
atlantis-url: "url"
allow-fork-prs: true
allow-repo-config: true
bitbucket-base-url: "https://bitbucket-base-url"
bitbucket-token: "bitbucket-token"
bitbucket-user: "bitbucket-user"
bitbucket-webhook-secret: "bitbucket-secret"
data-dir: "/path"
gh-hostname: "ghhostname"
gh-token: "token"
gh-user: "user"
gh-webhook-secret: "secret"
gitlab-hostname: "gitlab-hostname"
gitlab-token: "gitlab-token"
gitlab-user: "gitlab-user"
gitlab-webhook-secret: "gitlab-secret"
log-level: "debug"
port: 8181
repo-whitelist: "github.com/runatlantis/atlantis"
require-approval: true
require-mergeable: true
ssl-cert-file: cert-file
ssl-key-file: key-file
tfe-token: my-token
`)

	defer os.Remove(tmpFile) // nolint: errcheck
	c := setup(map[string]interface{}{
		cmd.AtlantisURLFlag:            "override-url",
		cmd.AllowForkPRsFlag:           false,
		cmd.AllowRepoConfigFlag:        false,
		cmd.BitbucketBaseURLFlag:       "https://override-bitbucket-base-url",
		cmd.BitbucketTokenFlag:         "override-bitbucket-token",
		cmd.BitbucketUserFlag:          "override-bitbucket-user",
		cmd.BitbucketWebhookSecretFlag: "override-bitbucket-secret",
		cmd.DataDirFlag:                "/override-path",
		cmd.GHHostnameFlag:             "override-gh-hostname",
		cmd.GHTokenFlag:                "override-gh-token",
		cmd.GHUserFlag:                 "override-gh-user",
		cmd.GHWebhookSecretFlag:        "override-gh-webhook-secret",
		cmd.GitlabHostnameFlag:         "override-gitlab-hostname",
		cmd.GitlabTokenFlag:            "override-gitlab-token",
		cmd.GitlabUserFlag:             "override-gitlab-user",
		cmd.GitlabWebhookSecretFlag:    "override-gitlab-webhook-secret",
		cmd.LogLevelFlag:               "info",
		cmd.PortFlag:                   8282,
		cmd.RepoWhitelistFlag:          "override,override",
		cmd.RequireApprovalFlag:        false,
		cmd.RequireMergeableFlag:       false,
		cmd.SSLCertFileFlag:            "override-cert-file",
		cmd.SSLKeyFileFlag:             "override-key-file",
		cmd.TFETokenFlag:               "override-my-token",
	})
	err := c.Execute()
	Ok(t, err)
	Equals(t, "override-url", passedConfig.AtlantisURL)
	Equals(t, false, passedConfig.AllowForkPRs)
	Equals(t, "https://override-bitbucket-base-url", passedConfig.BitbucketBaseURL)
	Equals(t, "override-bitbucket-token", passedConfig.BitbucketToken)
	Equals(t, "override-bitbucket-user", passedConfig.BitbucketUser)
	Equals(t, "override-bitbucket-secret", passedConfig.BitbucketWebhookSecret)
	Equals(t, "/override-path", passedConfig.DataDir)
	Equals(t, "override-gh-hostname", passedConfig.GithubHostname)
	Equals(t, "override-gh-token", passedConfig.GithubToken)
	Equals(t, "override-gh-user", passedConfig.GithubUser)
	Equals(t, "override-gh-webhook-secret", passedConfig.GithubWebhookSecret)
	Equals(t, "override-gitlab-hostname", passedConfig.GitlabHostname)
	Equals(t, "override-gitlab-token", passedConfig.GitlabToken)
	Equals(t, "override-gitlab-user", passedConfig.GitlabUser)
	Equals(t, "override-gitlab-webhook-secret", passedConfig.GitlabWebhookSecret)
	Equals(t, "info", passedConfig.LogLevel)
	Equals(t, 8282, passedConfig.Port)
	Equals(t, "override,override", passedConfig.RepoWhitelist)
	Equals(t, false, passedConfig.RequireApproval)
	Equals(t, false, passedConfig.RequireMergeable)
	Equals(t, "override-cert-file", passedConfig.SSLCertFile)
	Equals(t, "override-key-file", passedConfig.SSLKeyFile)
	Equals(t, "override-my-token", passedConfig.TFEToken)
}

func TestExecute_FlagEnvVarOverride(t *testing.T) {
	t.Log("Flags should override environment variables.")

	envVars := map[string]string{
		"ATLANTIS_URL":             "url",
		"ALLOW_FORK_PRS":           "true",
		"ALLOW_REPO_CONFIG":        "true",
		"BITBUCKET_BASE_URL":       "https://bitbucket-base-url",
		"BITBUCKET_TOKEN":          "bitbucket-token",
		"BITBUCKET_USER":           "bitbucket-user",
		"BITBUCKET_WEBHOOK_SECRET": "bitbucket-secret",
		"DATA_DIR":                 "/path",
		"GH_HOSTNAME":              "gh-hostname",
		"GH_TOKEN":                 "gh-token",
		"GH_USER":                  "gh-user",
		"GH_WEBHOOK_SECRET":        "gh-webhook-secret",
		"GITLAB_HOSTNAME":          "gitlab-hostname",
		"GITLAB_TOKEN":             "gitlab-token",
		"GITLAB_USER":              "gitlab-user",
		"GITLAB_WEBHOOK_SECRET":    "gitlab-webhook-secret",
		"LOG_LEVEL":                "debug",
		"PORT":                     "8181",
		"REPO_WHITELIST":           "*",
		"REQUIRE_APPROVAL":         "true",
		"REQUIRE_MERGEABLE":        "true",
		"SSL_CERT_FILE":            "cert-file",
		"SSL_KEY_FILE":             "key-file",
		"TFE_TOKEN":                "my-token",
	}
	for name, value := range envVars {
		os.Setenv("ATLANTIS_"+name, value) // nolint: errcheck
	}
	defer func() {
		// Unset after this test finishes.
		for name := range envVars {
			os.Unsetenv("ATLANTIS_" + name) // nolint: errcheck
		}
	}()

	c := setup(map[string]interface{}{
		cmd.AtlantisURLFlag:            "override-url",
		cmd.AllowForkPRsFlag:           false,
		cmd.AllowRepoConfigFlag:        false,
		cmd.BitbucketBaseURLFlag:       "https://override-bitbucket-base-url",
		cmd.BitbucketTokenFlag:         "override-bitbucket-token",
		cmd.BitbucketUserFlag:          "override-bitbucket-user",
		cmd.BitbucketWebhookSecretFlag: "override-bitbucket-secret",
		cmd.DataDirFlag:                "/override-path",
		cmd.GHHostnameFlag:             "override-gh-hostname",
		cmd.GHTokenFlag:                "override-gh-token",
		cmd.GHUserFlag:                 "override-gh-user",
		cmd.GHWebhookSecretFlag:        "override-gh-webhook-secret",
		cmd.GitlabHostnameFlag:         "override-gitlab-hostname",
		cmd.GitlabTokenFlag:            "override-gitlab-token",
		cmd.GitlabUserFlag:             "override-gitlab-user",
		cmd.GitlabWebhookSecretFlag:    "override-gitlab-webhook-secret",
		cmd.LogLevelFlag:               "info",
		cmd.PortFlag:                   8282,
		cmd.RepoWhitelistFlag:          "override,override",
		cmd.RequireApprovalFlag:        false,
		cmd.RequireMergeableFlag:       false,
		cmd.SSLCertFileFlag:            "override-cert-file",
		cmd.SSLKeyFileFlag:             "override-key-file",
		cmd.TFETokenFlag:               "override-my-token",
	})
	err := c.Execute()
	Ok(t, err)

	Equals(t, "override-url", passedConfig.AtlantisURL)
	Equals(t, false, passedConfig.AllowForkPRs)
	Equals(t, false, passedConfig.AllowRepoConfig)
	Equals(t, "https://override-bitbucket-base-url", passedConfig.BitbucketBaseURL)
	Equals(t, "override-bitbucket-token", passedConfig.BitbucketToken)
	Equals(t, "override-bitbucket-user", passedConfig.BitbucketUser)
	Equals(t, "override-bitbucket-secret", passedConfig.BitbucketWebhookSecret)
	Equals(t, "/override-path", passedConfig.DataDir)
	Equals(t, "override-gh-hostname", passedConfig.GithubHostname)
	Equals(t, "override-gh-token", passedConfig.GithubToken)
	Equals(t, "override-gh-user", passedConfig.GithubUser)
	Equals(t, "override-gh-webhook-secret", passedConfig.GithubWebhookSecret)
	Equals(t, "override-gitlab-hostname", passedConfig.GitlabHostname)
	Equals(t, "override-gitlab-token", passedConfig.GitlabToken)
	Equals(t, "override-gitlab-user", passedConfig.GitlabUser)
	Equals(t, "override-gitlab-webhook-secret", passedConfig.GitlabWebhookSecret)
	Equals(t, "info", passedConfig.LogLevel)
	Equals(t, 8282, passedConfig.Port)
	Equals(t, "override,override", passedConfig.RepoWhitelist)
	Equals(t, false, passedConfig.RequireApproval)
	Equals(t, false, passedConfig.RequireMergeable)
	Equals(t, "override-cert-file", passedConfig.SSLCertFile)
	Equals(t, "override-key-file", passedConfig.SSLKeyFile)
	Equals(t, "override-my-token", passedConfig.TFEToken)
}

// If using bitbucket cloud, webhook secrets are not supported.
func TestExecute_BitbucketCloudWithWebhookSecret(t *testing.T) {
	c := setup(map[string]interface{}{
		cmd.BitbucketUserFlag:          "user",
		cmd.BitbucketTokenFlag:         "token",
		cmd.RepoWhitelistFlag:          "*",
		cmd.BitbucketWebhookSecretFlag: "my secret",
	})
	err := c.Execute()
	ErrEquals(t, "--bitbucket-webhook-secret cannot be specified for Bitbucket Cloud because it is not supported by Bitbucket", err)
}

// Base URL must have a scheme.
func TestExecute_BitbucketServerBaseURLScheme(t *testing.T) {
	c := setup(map[string]interface{}{
		cmd.BitbucketUserFlag:    "user",
		cmd.BitbucketTokenFlag:   "token",
		cmd.RepoWhitelistFlag:    "*",
		cmd.BitbucketBaseURLFlag: "mydomain.com",
	})
	ErrEquals(t, "--bitbucket-base-url must have http:// or https://, got \"mydomain.com\"", c.Execute())

	c = setup(map[string]interface{}{
		cmd.BitbucketUserFlag:    "user",
		cmd.BitbucketTokenFlag:   "token",
		cmd.RepoWhitelistFlag:    "*",
		cmd.BitbucketBaseURLFlag: "://mydomain.com",
	})
	ErrEquals(t, "error parsing --bitbucket-webhook-secret flag value \"://mydomain.com\": parse ://mydomain.com: missing protocol scheme", c.Execute())
}

// Port should be retained on base url.
func TestExecute_BitbucketServerBaseURLPort(t *testing.T) {
	c := setup(map[string]interface{}{
		cmd.BitbucketUserFlag:    "user",
		cmd.BitbucketTokenFlag:   "token",
		cmd.RepoWhitelistFlag:    "*",
		cmd.BitbucketBaseURLFlag: "http://mydomain.com:7990",
	})
	Ok(t, c.Execute())
	Equals(t, "http://mydomain.com:7990", passedConfig.BitbucketBaseURL)
}

func setup(flags map[string]interface{}) *cobra.Command {
	vipr := viper.New()
	for k, v := range flags {
		vipr.Set(k, v)
	}
	c := &cmd.ServerCmd{
		ServerCreator: &ServerCreatorMock{},
		Viper:         vipr,
		SilenceOutput: true,
	}
	return c.Init()
}

func setupWithDefaults(flags map[string]interface{}) *cobra.Command {
	vipr := viper.New()
	flags[cmd.GHUserFlag] = "user"
	flags[cmd.GHTokenFlag] = "token"
	flags[cmd.RepoWhitelistFlag] = "*"

	for k, v := range flags {
		vipr.Set(k, v)
	}
	c := &cmd.ServerCmd{
		ServerCreator: &ServerCreatorMock{},
		Viper:         vipr,
		SilenceOutput: true,
	}
	return c.Init()
}

func tempFile(t *testing.T, contents string) string {
	f, err := ioutil.TempFile("", "")
	Ok(t, err)
	newName := f.Name() + ".yaml"
	err = os.Rename(f.Name(), newName)
	Ok(t, err)
	ioutil.WriteFile(newName, []byte(contents), 0644) // nolint: errcheck
	return newName
}
