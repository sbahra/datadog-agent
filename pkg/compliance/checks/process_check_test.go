// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.
package checks

import (
	"testing"

	"github.com/DataDog/datadog-agent/pkg/compliance"
	"github.com/DataDog/datadog-agent/pkg/compliance/mocks"
	"github.com/DataDog/datadog-agent/pkg/util/cache"

	"github.com/DataDog/gopsutil/process"

	"github.com/stretchr/testify/assert"
)

type processFixture struct {
	name      string
	check     processCheck
	processes processes
	expKV     compliance.KVMap
	expError  error
	useCache  bool
}

func (f *processFixture) run(t *testing.T) {
	t.Helper()

	reporter := f.check.reporter.(*mocks.Reporter)
	processFetcherFunc = func() (processes, error) {
		return f.processes, nil
	}

	if !f.useCache {
		cache.Cache.Delete(processCacheKey)
	}

	expectedCalls := 0
	if f.expKV != nil {
		reporter.On(
			"Report",
			newTestRuleEvent(
				[]string{"check_kind:process"},
				f.expKV,
			),
		).Once()
		expectedCalls = 1
	}

	err := f.check.Run()
	reporter.AssertNumberOfCalls(t, "Report", expectedCalls)
	assert.Equal(t, f.expError, err)
}

func TestProcessCheck(t *testing.T) {
	tests := []processFixture{
		{
			name: "Simple case",
			check: processCheck{
				baseCheck: newTestBaseCheck(&mocks.Reporter{}, checkKindProcess),
				process: &compliance.Process{
					Name: "proc1",
					Report: compliance.Report{
						{
							Kind:     "flag",
							Property: "--path",
							As:       "path",
						},
					},
				},
			},
			processes: map[int32]*process.FilledProcess{
				42: {
					Name:    "proc1",
					Cmdline: []string{"arg1", "--path=foo"},
				},
			},
			expKV: compliance.KVMap{
				"path": "foo",
			},
		},
		{
			name: "Process not found",
			check: processCheck{
				baseCheck: newTestBaseCheck(&mocks.Reporter{}, checkKindProcess),
				process: &compliance.Process{
					Name: "proc1",
					Report: compliance.Report{
						{
							Kind:     "flag",
							Property: "--path",
							As:       "path",
						},
					},
				},
			},
			processes: map[int32]*process.FilledProcess{
				42: {
					Name:    "proc2",
					Cmdline: []string{"arg1", "--path=foo"},
				},
				43: {
					Name:    "proc3",
					Cmdline: []string{"arg1", "--path=foo"},
				},
			},
			expKV: nil,
		},
		{
			name: "Argument not found",
			check: processCheck{
				baseCheck: newTestBaseCheck(&mocks.Reporter{}, checkKindProcess),
				process: &compliance.Process{
					Name: "proc1",
					Report: compliance.Report{
						{
							Kind:     "flag",
							Property: "--path",
							As:       "path",
						},
					},
				},
			},
			processes: map[int32]*process.FilledProcess{
				42: {
					Name:    "proc1",
					Cmdline: []string{"arg1", "--paths=foo"},
				},
			},
			expKV: nil,
		},
		{
			name: "Override returned value",
			check: processCheck{
				baseCheck: newTestBaseCheck(&mocks.Reporter{}, checkKindProcess),
				process: &compliance.Process{
					Name: "proc1",
					Report: compliance.Report{
						{
							Kind:     "flag",
							Property: "--verbose",
							As:       "verbose",
							Value:    "true",
						},
					},
				},
			},
			processes: map[int32]*process.FilledProcess{
				42: {
					Name:    "proc1",
					Cmdline: []string{"arg1", "--verbose"},
				},
			},
			expKV: compliance.KVMap{
				"verbose": "true",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestProcessCheckCache(t *testing.T) {
	// Run first fixture, populating cache
	firstContent := processFixture{
		check: processCheck{
			baseCheck: newTestBaseCheck(&mocks.Reporter{}, checkKindProcess),
			process: &compliance.Process{
				Name: "proc1",
				Report: compliance.Report{
					{
						Kind:     "flag",
						Property: "--path",
						As:       "path",
					},
				},
			},
		},
		processes: map[int32]*process.FilledProcess{
			42: {
				Name:    "proc1",
				Cmdline: []string{"arg1", "--path=foo"},
			},
		},
		expKV: compliance.KVMap{
			"path": "foo",
		},
	}
	firstContent.run(t)

	// Run second fixture, using cache
	secondFixture := processFixture{
		check: processCheck{
			baseCheck: newTestBaseCheck(&mocks.Reporter{}, checkKindProcess),
			process: &compliance.Process{
				Name: "proc1",
				Report: compliance.Report{
					{
						Kind:     "flag",
						Property: "--path",
						As:       "path",
					},
				},
			},
		},
		expKV: compliance.KVMap{
			"path": "foo",
		},
		useCache: true,
	}
	secondFixture.run(t)
}
