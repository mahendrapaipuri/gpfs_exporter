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
	"os/exec"
	"strings"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

func TestParsePerf(t *testing.T) {
	execCommand = fakeExecCommand
	mockedExitStatus = 0
	mockedStdout = `
_fs_io_s_ _n_ 10.22.0.106 _nn_ ib-pitzer-rw02.ten _rc_ 0 _t_ 1579358234 _tu_ 53212 _cl_ gpfs.osc.edu _fs_ scratch _d_ 48 _br_ 205607400434 _bw_ 74839282351 _oc_ 2377656 _cc_ 2201576 _rdc_ 59420404 _wc_ 18874626 _dir_ 40971 _iu_ 544768
_fs_io_s_ _n_ 10.22.0.106 _nn_ ib-pitzer-rw02.ten _rc_ 0 _t_ 1579358234 _tu_ 53212 _cl_ gpfs.osc.edu _fs_ project _d_ 96 _br_ 0 _bw_ 0 _oc_ 513 _cc_ 513 _rdc_ 0 _wc_ 0 _dir_ 0 _iu_ 169
`
	defer func() { execCommand = exec.CommandContext }()
	perfs, err := mmpmon_parse(mockedStdout, log.NewNopLogger())
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if len(perfs) != 2 {
		t.Errorf("Expected 2 perfs returned, got %d", len(perfs))
		return
	}
	if val := perfs[0].FS; val != "scratch" {
		t.Errorf("Unexpected FS got %s", val)
	}
	if val := perfs[1].FS; val != "project" {
		t.Errorf("Unexpected FS got %s", val)
	}
	if val := perfs[0].NodeName; val != "ib-pitzer-rw02.ten" {
		t.Errorf("Unexpected NodeName got %s", val)
	}
	if val := perfs[1].NodeName; val != "ib-pitzer-rw02.ten" {
		t.Errorf("Unexpected NodeName got %s", val)
	}
	if val := perfs[0].ReadBytes; val != 205607400434 {
		t.Errorf("Unexpected ReadBytes got %d", val)
	}
}

func TestMmpmonCollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--exporter.use-cache"}); err != nil {
		t.Fatal(err)
	}
	execCommand = fakeExecCommand
	mockedExitStatus = 0
	mockedStdout = `
_fs_io_s_ _n_ 10.22.0.106 _nn_ ib-pitzer-rw02.ten _rc_ 0 _t_ 1579358234 _tu_ 53212 _cl_ gpfs.osc.edu _fs_ scratch _d_ 48 _br_ 205607400434 _bw_ 74839282351 _oc_ 2377656 _cc_ 2201576 _rdc_ 59420404 _wc_ 18874626 _dir_ 40971 _iu_ 544768
_fs_io_s_ _n_ 10.22.0.106 _nn_ ib-pitzer-rw02.ten _rc_ 0 _t_ 1579358234 _tu_ 53212 _cl_ gpfs.osc.edu _fs_ project _d_ 96 _br_ 0 _bw_ 0 _oc_ 513 _cc_ 513 _rdc_ 0 _wc_ 0 _dir_ 0 _iu_ 169
`
	defer func() { execCommand = exec.CommandContext }()
	expected := `
		# HELP gpfs_perf_operations GPFS operationgs reported by mmpmon
		# TYPE gpfs_perf_operations counter
		gpfs_perf_operations{fs="project",nodename="ib-pitzer-rw02.ten",operation="closes"} 513
		gpfs_perf_operations{fs="project",nodename="ib-pitzer-rw02.ten",operation="inode_updates"} 169
		gpfs_perf_operations{fs="project",nodename="ib-pitzer-rw02.ten",operation="opens"} 513
		gpfs_perf_operations{fs="project",nodename="ib-pitzer-rw02.ten",operation="read_dir"} 0
		gpfs_perf_operations{fs="project",nodename="ib-pitzer-rw02.ten",operation="reads"} 0
		gpfs_perf_operations{fs="project",nodename="ib-pitzer-rw02.ten",operation="writes"} 0
		gpfs_perf_operations{fs="scratch",nodename="ib-pitzer-rw02.ten",operation="closes"} 2201576
		gpfs_perf_operations{fs="scratch",nodename="ib-pitzer-rw02.ten",operation="inode_updates"} 544768
		gpfs_perf_operations{fs="scratch",nodename="ib-pitzer-rw02.ten",operation="opens"} 2377656
		gpfs_perf_operations{fs="scratch",nodename="ib-pitzer-rw02.ten",operation="read_dir"} 40971
		gpfs_perf_operations{fs="scratch",nodename="ib-pitzer-rw02.ten",operation="reads"} 59420404
		gpfs_perf_operations{fs="scratch",nodename="ib-pitzer-rw02.ten",operation="writes"} 18874626
		# HELP gpfs_perf_read_bytes GPFS read bytes
		# TYPE gpfs_perf_read_bytes counter
		gpfs_perf_read_bytes{fs="project",nodename="ib-pitzer-rw02.ten"} 0
		gpfs_perf_read_bytes{fs="scratch",nodename="ib-pitzer-rw02.ten"} 2.05607400434e+11
		# HELP gpfs_perf_write_bytes GPFS write bytes
		# TYPE gpfs_perf_write_bytes counter
		gpfs_perf_write_bytes{fs="project",nodename="ib-pitzer-rw02.ten"} 0
		gpfs_perf_write_bytes{fs="scratch",nodename="ib-pitzer-rw02.ten"} 74839282351
	`
	collector := NewMmpmonCollector(log.NewNopLogger())
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 19 {
		t.Errorf("Unexpected collection count %d, expected 19", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected), "gpfs_perf_read_bytes", "gpfs_perf_write_bytes", "gpfs_perf_operations"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}
