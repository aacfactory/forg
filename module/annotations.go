package module

import (
	"bufio"
	"bytes"
	"github.com/aacfactory/errors"
	"io"
	"strings"
)

func ParseAnnotations(s string) (annotations Annotations, err error) {
	annotations = make(map[string]string)
	if s == "" || !strings.Contains(s, "@") {
		return
	}
	currentKey := ""
	currentBody := bytes.NewBuffer(make([]byte, 0, 1))
	blockReading := false
	reader := bufio.NewReader(bytes.NewReader([]byte(s)))
	for {
		line, _, readErr := reader.ReadLine()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			err = errors.Warning("forg: parse annotations failed").WithCause(readErr).WithMeta("source", s)
			return
		}
		if line == nil {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '@' {
			if blockReading {
				currentBody.Write(line)
				if len(line) > 0 {
					currentBody.WriteByte('\n')
				}
				continue
			}
			if len(line) == 1 {
				continue
			}
			if currentKey != "" {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()
			}
			idx := bytes.IndexByte(line, ' ')
			if idx < 0 {
				currentKey = string(line[1:])
				continue
			}
			currentKey = string(line[1:idx])
			line = bytes.TrimSpace(line[idx:])
		}
		if len(line) == 0 {
			continue
		}
		if blockReading {
			remains, hasBlockEnd := bytes.CutSuffix(line, []byte{'<', '<', '<'})
			currentBody.Write(remains)
			if hasBlockEnd {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()

				blockReading = false
			} else {
				if len(remains) > 0 {
					currentBody.WriteByte('\n')
				}
			}
			continue
		}
		line, blockReading = bytes.CutPrefix(line, []byte{'>', '>', '>'})
		if blockReading && currentKey != "" {
			remains, hasBlockEnd := bytes.CutSuffix(line, []byte{'<', '<', '<'})
			currentBody.Write(remains)
			if hasBlockEnd {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()

				blockReading = false
			} else {
				if len(remains) > 0 {
					currentBody.WriteByte('\n')
				}
			}
			continue
		} else if currentKey != "" {
			currentBody.Write(line)

			annotations.set(currentKey, currentBody.String())
			currentKey = ""
			currentBody.Reset()
		}

	}
	if currentKey != "" {
		annotations.set(currentKey, currentBody.String())
		currentKey = ""
		currentBody.Reset()
	}
	return
}

type Annotations map[string]string

func (annotations Annotations) Get(key string) (value string, has bool) {
	value, has = annotations[key]
	return
}

func (annotations Annotations) set(key string, value string) {
	value, _ = strings.CutSuffix(value, "\n")
	value = strings.ReplaceAll(value, "'>>>'", ">>>")
	value = strings.ReplaceAll(value, "'<<<'", "<<<")
	annotations[key] = value
}
