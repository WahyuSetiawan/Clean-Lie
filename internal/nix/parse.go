package nix

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var sizeRe = regexp.MustCompile(`(?i)([0-9]+(?:[.,][0-9]+)?)\s*([KMGTP]?i?B?)`)

// parseSize mem-parse "4.0G", "3.9 GiB", "123.4 MiB" menjadi byte.
func parseSize(s string) (int64, bool) {
	m := sizeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil {
		return 0, false
	}
	unit := strings.ToUpper(strings.TrimSuffix(m[2], "B"))
	unit = strings.TrimSuffix(unit, "I")
	exp := strings.Index("KMGTP", unit)
	if unit == "" {
		return int64(v), true
	}
	if exp < 0 {
		return 0, false
	}
	mult := float64(1)
	for i := 0; i <= exp; i++ {
		mult *= 1024
	}
	return int64(v * mult), true
}

// parsePathInfoS mem-parse output `nix path-info -S`:
//
//	/nix/store/xxx  12345
func parsePathInfoS(out string) map[string]int64 {
	res := map[string]int64{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "/") {
			continue
		}
		n, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
		if err != nil {
			continue
		}
		res[fields[0]] = n
	}
	return res
}

// parseNarSizes mem-parse `nix path-info --json` (object map atau array).
func parseNarSizes(out string) []int64 {
	var obj map[string]struct {
		NarSize int64 `json:"narSize"`
	}
	if err := json.Unmarshal([]byte(out), &obj); err == nil {
		res := make([]int64, 0, len(obj))
		for _, v := range obj {
			res = append(res, v.NarSize)
		}
		return res
	}
	var arr []struct {
		NarSize int64 `json:"narSize"`
	}
	if err := json.Unmarshal([]byte(out), &arr); err == nil {
		res := make([]int64, 0, len(arr))
		for _, v := range arr {
			res = append(res, v.NarSize)
		}
		return res
	}
	return nil
}

func parseDeadPaths(out string) []string {
	var res []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, StorePath+"/") {
			res = append(res, line)
		}
	}
	return res
}

var deletedRe = regexp.MustCompile(`(\d+)\s+store paths?\s+(?:(?:would be|were|have been)\s+)?deleted`)

func parseGCDeletedCount(out string) int {
	m := deletedRe.FindStringSubmatch(out)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}
