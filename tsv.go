// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"strconv"
	"time"

	"cloudeng.io/cmd/idu/internal/reports"
)

type tsvReports struct {
}

func (tr *tsvReports) generateReports(ctx context.Context, rf *reportsFlags, when time.Time, filenames *reportFilenames, data []byte) error {
	var sdb reports.AllStats
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&sdb); err != nil {
		return fmt.Errorf("failed to decode stats: %v", err)
	}
	return writeReportFiles(&sdb, filenames, tr.formatMerged, tr.formatUserGroupMerged, rf.TSV)
}

func (tr *tsvReports) formatMerged(merged map[string]reports.MergedStats) []byte {
	out := &bytes.Buffer{}
	wr := csv.NewWriter(out)
	wr.Comma = '\t'
	wr.Write([]string{"prefix", "bytes", "storage bytes", "files", "directories", "directories", "directory bytes"})
	for k, v := range merged {
		wr.Write([]string{k,
			strconv.FormatInt(v.Bytes, 10),
			strconv.FormatInt(v.Storage, 10),
			strconv.FormatInt(v.Files, 10),
			strconv.FormatInt(v.Prefixes, 10),
			strconv.FormatInt(v.PrefixBytes, 10)})
	}
	wr.Flush()
	return out.Bytes()
}

func (tr *tsvReports) formatUserGroupMerged(merged map[uint32]reports.MergedStats, nameForID func(uint32) string) []byte {
	out := &bytes.Buffer{}
	wr := csv.NewWriter(out)
	wr.Comma = '\t'
	wr.Write([]string{"id", "idname", "bytes", "storage bytes", "files", "directories", "directory bytes"})
	for k, v := range merged {
		wr.Write([]string{
			strconv.FormatUint(uint64(k), 10),
			nameForID(k),
			strconv.FormatInt(v.Bytes, 10),
			strconv.FormatInt(v.Storage, 10),
			strconv.FormatInt(v.Files, 10),
			strconv.FormatInt(v.Prefixes, 10),
			strconv.FormatInt(v.PrefixBytes, 10),
		})
	}
	wr.Flush()
	return out.Bytes()
}