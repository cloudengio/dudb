// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cloudeng.io/cmdutil/structdoc"
	"cloudeng.io/file/diskusage"
	"gopkg.in/yaml.v3"
)

type Prefix struct {
	Prefix                   string   `yaml:"prefix" cmd:"the prefix to be analyzed"`
	Database                 string   `yaml:"database" cmd:"the location of the database to use for this prefix"`
	Separator                string   `yaml:"separator" cmd:"filename separator to use, defaults to /"`
	ConcurrentScans          int      `yaml:"concurrent_scans" cmd:"maximum number of concurrent scan operations"`
	ConcurrentStats          int      `yaml:"concurrent_stats" cmd:"maximum number of concurrent stat operations"`
	ConcurrentStatsThreshold int      `yaml:"concurrent_stats_threshold" cmd:"minimum number of files before stats are performed concurrently"`
	SetMaxThreads            int      `yaml:"set_max_threads" cmd:"if non-zero used for debug.SetMaxThreads"`
	ScanSize                 int      `yaml:"scan_size" cmd:"maximum number of items to fetch from the filesystem in a single operation"`
	Exclusions               []string `yaml:"exclusions" cmd:"prefixes and files matching these regular expressions will be ignored when building a dataase"`
	CountHardlinkAsFiles     bool     `yaml:"count_hardlinks_as_files" cmd:"if true, hardlinks will be counted as separate files"`

	Layout layout `yaml:"layout" cmd:"the filesystem layout to use for calculating raw bytes used"`

	regexps    []*regexp.Regexp
	calculator diskusage.Calculator
}

type layout struct {
	Calculator string    `yaml:"calculator" cmd:"the type of disk usage calculator to use"`
	Parameters yaml.Node `yaml:"parameters" cmd:"the layout parameters to use for this calculator"`
}

func (p *Prefix) Calculator() diskusage.Calculator {
	return p.calculator
}

type T struct {
	Prefixes []Prefix `yaml:"prefixes" cmd:"the prefixes to be analyzed"`
}

// ForPrefix returns the prefix configuration that corresponds to path. The
// prefix is the longest matching prefix in the configuration and the returned
// string is the path relative to that prefix. The boolean return value is true
// if a match is found.
func (t T) ForPrefix(path string) (Prefix, bool) {
	var longest Prefix
	for _, p := range t.Prefixes {
		if strings.HasPrefix(path, p.Prefix) && len(p.Prefix) > len(longest.Prefix) {
			longest = p
		}
	}
	if longest.Prefix == "" {
		return Prefix{}, false
	}
	return longest, true
}

// Exclude returns true if path should be excluded/ignored.
func (p *Prefix) Exclude(path string) bool {
	for _, re := range p.regexps {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

var (
	DefaultConcurrentStats          = 0
	DefaultConcurrentStatsThreshold = 0
	DefaultConcurrentScans          = 0
	DefaultScanSize                 = 0
)

// ParseConfig will parse a yaml config from the supplied byte slice.
func ParseConfig(buf []byte) (T, error) {
	var cfg T
	if err := yaml.Unmarshal(buf, &cfg.Prefixes); err != nil {
		return T{}, err
	}

	raw := []map[string]any{}
	if err := yaml.Unmarshal(buf, &raw); err != nil {
		return T{}, err
	}

	for i, p := range cfg.Prefixes {
		cfg.Prefixes[i].Prefix = os.ExpandEnv(p.Prefix)
		cfg.Prefixes[i].Database = os.ExpandEnv(p.Database)
		for _, e := range p.Exclusions {
			re, err := regexp.Compile(e)
			if err != nil {
				return T{}, err
			}
			cfg.Prefixes[i].regexps = append(cfg.Prefixes[i].regexps, re)
		}
		calc, err := parseLayout(&cfg.Prefixes[i].Layout)
		if err != nil {
			return T{}, err
		}
		cfg.Prefixes[i].calculator = calc
		if len(p.Separator) == 0 {
			cfg.Prefixes[i].Separator = string(filepath.Separator)
		}
		if _, ok := raw[i]["concurrent_stats_threshold"]; !ok {
			cfg.Prefixes[i].ConcurrentStatsThreshold = DefaultConcurrentStatsThreshold
		}
		if _, ok := raw[i]["concurrent_stats"]; !ok {
			cfg.Prefixes[i].ConcurrentStats = DefaultConcurrentStats
		}
		if _, ok := raw[i]["concurrent_scans"]; !ok {
			cfg.Prefixes[i].ConcurrentScans = DefaultConcurrentScans
		}
		if _, ok := raw[i]["scan_size"]; !ok {
			cfg.Prefixes[i].ScanSize = DefaultScanSize
		}
	}
	return cfg, nil
}

// ReadConfig will read a yaml config from the specified file.
func ReadConfig(filename string) (T, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return T{}, fmt.Errorf("failed to read config file: %v: %v", filename, err)
	}
	return ParseConfig(buf)
}

type RAID0 struct {
	StripeSize int64 `yaml:"stripe_size" cmd:"the size of the raid0 stripes"`
	NumStripes int   `yaml:"num_stripes" cmd:"the number of stripes used"`
}

type Block struct {
	BlockSize int64 `yaml:"size" cmd:"block size used by this filesystem"`
}

type layoutConfig struct {
	newLayout      func(n yaml.Node) (diskusage.Calculator, error)
	describeLayout func() string
}

func bytesCalc(yaml.Node) (diskusage.Calculator, error) {
	return diskusage.NewIdentity(), nil
}

func bytesDesc() string {
	return "bytes: assumes that the size of each file is the number of bytes used\n"
}

func blockCalc(n yaml.Node) (diskusage.Calculator, error) {
	var b Block
	if err := n.Decode(&b); err != nil {
		return nil, fmt.Errorf("failed parsing block layout parameters: %v", err)
	}
	return diskusage.NewBlock(b.BlockSize), nil
}

func blockDesc() string {
	desc, _ := structdoc.Describe(&Block{}, "cmd", "block calculator parameters\n")
	out := &strings.Builder{}
	out.WriteString("block: the size of each file is a multiple of the block size\n")
	out.WriteString(structdoc.FormatFields(2, 4, desc.Fields))
	return out.String()
}

func raid0Calc(n yaml.Node) (diskusage.Calculator, error) {
	var r RAID0
	if err := n.Decode(&r); err != nil {
		return nil, fmt.Errorf("failed parsing RAID0 layout parameters: %v", err)
	}
	return diskusage.NewRAID0(r.StripeSize, r.NumStripes), nil
}

func raid0Desc() string {
	desc, _ := structdoc.Describe(&RAID0{}, "cmd", "raid0 calculator parameters\n")
	out := &strings.Builder{}
	out.WriteString("raid0: the size of each file depends on the RAID0 parameters in use\n")
	out.WriteString(structdoc.FormatFields(2, 4, desc.Fields))
	return out.String()
}

var supportedLayouts = map[string]layoutConfig{
	"bytes": {bytesCalc, bytesDesc},
	"block": {blockCalc, blockDesc},
	"raid0": {raid0Calc, raid0Desc},
}

func parseLayout(l *layout) (diskusage.Calculator, error) {
	if len(l.Calculator) == 0 {
		l.Calculator = "bytes"
	}
	supported, ok := supportedLayouts[strings.ToLower(l.Calculator)]
	if !ok {
		return nil, fmt.Errorf("unsupported disk usage calculator: %v", l.Calculator)
	}
	return supported.newLayout(l.Parameters)
}

// Documentation will return a description of the format of the
// yaml configuration file.
func Documentation() string {
	out := &strings.Builder{}
	desc, err := structdoc.Describe(&T{}, "cmd", "YAML configuration file options\n")
	if err != nil {
		panic(err)
	}
	out.WriteString(structdoc.FormatFields(0, 2, desc.Fields))

	out.WriteString("\nSupported layouts:\n\n")
	for _, v := range supportedLayouts {
		out.WriteString(v.describeLayout())
		out.WriteRune('\n')
	}
	return out.String()
}
