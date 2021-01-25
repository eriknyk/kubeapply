package pullreq

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/segmentio/kubeapply/data"
	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
	"github.com/segmentio/kubeapply/pkg/config"
)

const (
	// Special string used to indicate where we we can split very long comments.
	kubeapplySplit = "<!-- KUBEAPPLY_SPLIT -->"
)

var templates *template.Template

func init() {
	var err error
	templates, err = loadTemplates()
	if err != nil {
		panic(err)
	}
}

// ApplyCommentData stores data for templating out a "kubeapply apply" comment result.
type ApplyCommentData struct {
	ClusterApplies    []ClusterApply
	PullRequestClient PullRequestClient
	Env               string
}

// ClusterApply contains the results of applying in a single cluster.
type ClusterApply struct {
	ClusterConfig *config.ClusterConfig
	Results       []apply.Result
}

// NumUpdates returns the number of updates that were made as part of the apply.
func (c ClusterApply) NumUpdates() int {
	updates := 0

	for _, result := range c.Results {
		if result.IsCreated() || result.IsUpdated() {
			updates++
		}
	}

	return updates
}

// FormatApplyComment generates the body of an apply comment result.
func FormatApplyComment(commentData ApplyCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"apply_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

// DiffCommentData stores data for templating out a "kubeapply diff" comment result.
type DiffCommentData struct {
	ClusterDiffs      []ClusterDiff
	PullRequestClient PullRequestClient
	Env               string
}

// ClusterDiff contains the results of a diff in a single cluster.
type ClusterDiff struct {
	ClusterConfig *config.ClusterConfig
	Results       []diff.Result
}

// FormatDiffComment generates the body of a diff comment result.
func FormatDiffComment(commentData DiffCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"diff_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

// ErrorCommentData represents data for templating out a kubeapply error comment result.
type ErrorCommentData struct {
	Error error
	Env   string
	Notes []string
}

// FormatErrorComment generates the body of an error comment result.
func FormatErrorComment(commentData ErrorCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"error_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

// HelpCommentData stores data for templating out a "kubeapply help" comment result.
type HelpCommentData struct {
	ClusterConfigs []*config.ClusterConfig
	Env            string
}

// FormatHelpComment generates the body of a help comment result.
func FormatHelpComment(commentData HelpCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"help_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return string(out.Bytes()), nil
}

// StatusCommentData stores data for templating out a "kubeapply status" comment result.
type StatusCommentData struct {
	ClusterStatuses   []ClusterStatus
	PullRequestClient PullRequestClient
	Env               string
}

// ClusterStatus contains the results of getting the status of a cluster.
type ClusterStatus struct {
	ClusterConfig *config.ClusterConfig
	HealthSummary string
}

// FormatStatusComment generates the body of a status comment result.
func FormatStatusComment(commentData StatusCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"status_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

func loadTemplates() (*template.Template, error) {
	templates := template.New("base")

	tempDir, err := ioutil.TempDir("", "templates")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	data.RestoreAssets(tempDir, "pkg/pullreq/templates")
	return templates.ParseGlob(
		filepath.Join(tempDir, "pkg/pullreq/templates/*.gotpl"),
	)
}

func commentChunks(body string, maxLen int) []string {
	chunks := []string{}

	if len(body) > maxLen {
		start := 0
		end := maxLen

		var splitStr string

		if strings.Contains(body, kubeapplySplit) {
			splitStr = kubeapplySplit
		} else {
			// Just split on newlines
			splitStr = "\n"
		}

		for start < end {
			// Try to split at first splitter string after end
			splitIndex := strings.Index(body[end:], splitStr)
			if splitIndex > 0 {
				end += splitIndex
			}

			chunk := body[start:end]
			chunks = append(chunks, chunk)

			if splitIndex > 0 {
				// Swallow splitter string
				start = end + len(splitStr)
			} else {
				start = end
			}
			end = min(start+maxLen, len(body))
		}
	} else {
		chunks = append(chunks, body)
	}

	return chunks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
