package gravity

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	sshutils "github.com/gravitational/robotest/lib/ssh"

	"github.com/gravitational/trace"
)

// parseStatus interprets output of `gravity status` as either text or JSON
func parseStatus(status *ClusterStatus) sshutils.OutputParseFn {
	return func(r io.Reader) error {
		br, isJSON := guessJSONStream(r, 1024)

		if isJSON {
			return fromJSON(br, status)
		} else {
			return fromText(br, status)
		}
	}
}

func fromJSON(r io.Reader, status *ClusterStatus) error {
	d := json.NewDecoder(r)
	if err := d.Decode(&status); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func fromText(r *bufio.Reader, status *ClusterStatus) error {
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return trace.Wrap(err)
		}

		vars := rStatusKV.FindStringSubmatch(line)
		if len(vars) == 3 {
			populateStatus(strings.TrimSpace(vars[1]), strings.TrimSpace(vars[2]), status)
			continue
		}

		vars = rStatusNodeIP.FindStringSubmatch(line)
		if len(vars) == 2 {
			status.Nodes = append(status.Nodes, ClusterServer{AdvertiseIP: strings.TrimSpace(vars[1])})
			continue
		}

	}

	return nil
}

func populateStatus(key, value string, status *ClusterStatus) error {
	switch key {
	case "Cluster":
		status.Name = value
	case "Join token":
		status.Token.Token = value
	case "Application":
		status.App.Name = value
	case "Status", "Application Status":
		status.Status = value
	default:
	}
	return nil
}

// from https://github.com/gravitational/gravity/blob/master/lib/utils/parse.go
//
// ParseDDOutput parses the output of "dd" command and returns the reported
// speed in bytes per second.
//
// Example output:
//
// $ dd if=/dev/zero of=/tmp/testfile bs=1G count=1
// 1+0 records in
// 1+0 records out
// 1073741824 bytes (1.1 GB) copied, 4.52455 s, 237 MB/s
func ParseDDOutput(output string) (uint64, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		return 0, trace.BadParameter("expected 3 lines but got %v:\n%v", len(lines), output)
	}

	// 1073741824 bytes (1.1 GB) copied, 4.52455 s, 237 MB/s
	// 1073741824 bytes (1,1 GB, 1,0 GiB) copied, 4,53701 s, 237 MB/s
	testResults := lines[2]
	match := rSpeed.FindStringSubmatch(testResults)
	if len(match) != 2 {
		return 0, trace.BadParameter("failed to match speed value (e.g. 237 MB/s) in %q", testResults)
	}

	// Support comma-formatted floats - depending on selected locale
	speedValue := strings.TrimSpace(strings.Replace(match[1], ",", ".", 1))
	value, err := strconv.ParseFloat(speedValue, 64)
	if err != nil {
		return 0, trace.Wrap(err, "failed to parse speed value as a float: %q", speedValue)
	}

	units := strings.TrimSpace(strings.TrimPrefix(match[0], match[1]))
	switch units {
	case "kB/s":
		return uint64(value * 1000), nil
	case "MB/s":
		return uint64(value * 1000 * 1000), nil
	case "GB/s":
		return uint64(value * 1000 * 1000 * 1000), nil
	default:
		return 0, trace.BadParameter("expected units (one of kB/s, MB/s, GB/s) but got %q", units)
	}
}

func guessJSONStream(r io.Reader, size int) (*bufio.Reader, bool) {
	buffer := bufio.NewReaderSize(r, size)
	b, _ := buffer.Peek(size)
	return buffer, hasJSONPrefix(b)
}

var jsonPrefix = []byte("{")

// hasJSONPrefix returns true if the provided buffer appears to start with
// a JSON open brace.
func hasJSONPrefix(buf []byte) bool {
	trim := bytes.TrimLeftFunc(buf, unicode.IsSpace)
	return bytes.HasPrefix(trim, jsonPrefix)
}

// rStatusKV matches key/value pairs - i.e. "Status: active"
var rStatusKV = regexp.MustCompile(`^(?P<key>[\w\s]+)\:\s*(?P<val>[\w\d\_\-\.]+),*.*`)

// rStatusNodeIP matches a node's IP address
var rStatusNodeIP = regexp.MustCompile(`^[\s\w\-\d\*]+\((?P<ip>[\d\.]+)\).*`)

// rSpeed matches the value of a speed gauge from the output of `dd`
var rSpeed = regexp.MustCompile(`(\d+(?:[.,]\d+)?) \w+/s$`)
