// Copyright 2020 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collectors

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbsStdout = `
VERBS RDMA status: started
`
)

func TestVerbs(t *testing.T) {
	execCommand = fakeExecCommand
	mockedExitStatus = 0
	mockedStdout = "foo"
	defer func() { execCommand = exec.CommandContext }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := verbs(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if out != mockedStdout {
		t.Errorf("Unexpected out: %s", out)
	}
}

func TestParseVerbsDisabled(t *testing.T) {
	stdout := `
VERBS RDMA status: disabled
`
	metric, err := verbs_parse(stdout)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if metric.Status != "disabled" {
		t.Errorf("Unexpected value for status, expected disabled, got %s", metric.Status)
	}
}

func TestParseVerbsStarted(t *testing.T) {
	metric, err := verbs_parse(verbsStdout)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if metric.Status != "started" {
		t.Errorf("Unexpected value for status, expected started, got %s", metric.Status)
	}
}

func TestVerbsCollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	verbsExec = func(ctx context.Context) (string, error) {
		return verbsStdout, nil
	}
	expected := `
		# HELP gpfs_verbs_status GPFS verbs status, 1=started 0=not started
		# TYPE gpfs_verbs_status gauge
		gpfs_verbs_status 1
	`
	collector := NewVerbsCollector(log.NewNopLogger(), false)
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 4 {
		t.Errorf("Unexpected collection count %d, expected 4", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected), "gpfs_verbs_status"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestVerbsCollectorError(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	verbsExec = func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("Error")
	}
	expected := `
		# HELP gpfs_exporter_collect_error Indicates if error has occurred during collection
		# TYPE gpfs_exporter_collect_error gauge
		gpfs_exporter_collect_error{collector="verbs"} 1
	`
	collector := NewVerbsCollector(log.NewNopLogger(), false)
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 3 {
		t.Errorf("Unexpected collection count %d, expected 3", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected), "gpfs_exporter_collect_error"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestVerbsCollectorTimeout(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	verbsExec = func(ctx context.Context) (string, error) {
		return "", context.DeadlineExceeded
	}
	expected := `
		# HELP gpfs_exporter_collect_timeout Indicates the collector timed out
		# TYPE gpfs_exporter_collect_timeout gauge
		gpfs_exporter_collect_timeout{collector="verbs"} 1
	`
	collector := NewVerbsCollector(log.NewNopLogger(), false)
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 3 {
		t.Errorf("Unexpected collection count %d, expected 3", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected), "gpfs_exporter_collect_timeout"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestVerbsCollectorCache(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	// build cache
	verbsExec = func(ctx context.Context) (string, error) {
		return verbsStdout, nil
	}
	collector := NewVerbsCollector(log.NewNopLogger(), true)
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 4 {
		t.Errorf("Unexpected collection count %d, expected 4", val)
	}

	verbsExec = func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("Error")
	}
	expected := `
		# HELP gpfs_verbs_status GPFS verbs status, 1=started 0=not started
		# TYPE gpfs_verbs_status gauge
		gpfs_verbs_status 1
	`
	errorMetrics := `
		# HELP gpfs_exporter_collect_error Indicates if error has occurred during collection
		# TYPE gpfs_exporter_collect_error gauge
		gpfs_exporter_collect_error{collector="verbs"} 1
	`
	if val := testutil.CollectAndCount(collector); val != 4 {
		t.Errorf("Unexpected collection count %d, expected 4", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected+errorMetrics), "gpfs_verbs_status", "gpfs_exporter_collect_error"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}

	timeoutMetrics := `
		# HELP gpfs_exporter_collect_timeout Indicates the collector timed out
		# TYPE gpfs_exporter_collect_timeout gauge
		gpfs_exporter_collect_timeout{collector="verbs"} 1
	`
	verbsExec = func(ctx context.Context) (string, error) {
		return "", context.DeadlineExceeded
	}
	if val := testutil.CollectAndCount(collector); val != 4 {
		t.Errorf("Unexpected collection count %d, expected 4", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected+timeoutMetrics), "gpfs_verbs_status", "gpfs_exporter_collect_timeout"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}

	verbsCache = VerbsMetrics{}
	verbsExec = func(ctx context.Context) (string, error) {
		return verbsStdout, nil
	}
	if val := testutil.CollectAndCount(collector); val != 4 {
		t.Errorf("Unexpected collection count %d, expected 4", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected), "gpfs_verbs_status"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
	if val := verbsCache.Status; val != "started" {
		t.Errorf("Unexpected status %s, expected started", val)
	}
}
