package module

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aacfactory/errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	deps = "https://deps.dev/_/s/go/p"
)

func LatestVersion(path string) (v string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		err = errors.Warning("forg: get version from deps.dev failed").WithCause(errors.Warning("forg: path is required"))
		return
	}
	path = strings.ReplaceAll(url.PathEscape(path), "/", "%2F")
	resp, getErr := http.Get(fmt.Sprintf("%s/%s", deps, path))
	if getErr != nil {
		if errors.Map(getErr).Contains(http.ErrHandlerTimeout) {
			v, err = LatestVersionFromProxy(path)
			return
		}
		err = errors.Warning("forg: get version from deps.dev failed").
			WithCause(getErr).
			WithMeta("path", path)
		return
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusGatewayTimeout {
			v, err = LatestVersionFromProxy(path)
			return
		}
		err = errors.Warning("forg: get version from deps.dev failed").
			WithCause(errors.Warning("status code is not ok").WithMeta("status", strconv.Itoa(resp.StatusCode))).
			WithMeta("path", path)
		return
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		err = errors.Warning("forg: get version from deps.dev failed").WithCause(readErr).WithMeta("path", path)
		return
	}
	_ = resp.Body.Close()
	result := DepResult{}
	decodeErr := json.Unmarshal(body, &result)
	if decodeErr != nil {
		err = errors.Warning("forg: get version from deps.dev failed").WithCause(decodeErr).WithMeta("path", path)
		return
	}
	v = result.Version.Version
	if v == "" {
		err = errors.Warning("forg: get version from deps.dev failed").WithCause(errors.Warning("forg: version was not found")).WithMeta("path", path)
		return
	}
	return
}

type DepVersion struct {
	Version string `json:"version"`
}

type DepResult struct {
	Version DepVersion `json:"version"`
}

func LatestVersionFromProxy(path string) (v string, err error) {
	goproxy, hasProxy := os.LookupEnv("GOPROXY")
	if !hasProxy || goproxy == "" {
		err = errors.Warning("forg: get version from goproxy failed").WithCause(errors.Warning("goproxy was not set")).WithMeta("path", path)
		return
	}
	proxys := strings.Split(goproxy, ",")
	proxy := ""
	for _, p := range proxys {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			proxy = p
			break
		}
	}
	if proxy == "" {
		err = errors.Warning("forg: get version from goproxy failed").WithCause(errors.Warning("goproxy is invalid")).WithMeta("path", path)
		return
	}
	resp, getErr := http.Get(fmt.Sprintf("%s/%s/@v/list", proxy, path))
	if getErr != nil {
		err = errors.Warning("forg: get version from goproxy failed").WithCause(getErr).WithMeta("path", path)
		return
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		err = errors.Warning("forg: get version from goproxy failed").WithCause(readErr).WithMeta("path", path)
		return
	}
	_ = resp.Body.Close()
	if len(body) == 0 {
		err = errors.Warning("forg: get version from goproxy failed").WithCause(errors.Warning("forg: version was not found")).WithMeta("path", path)
		return
	}
	idx := bytes.LastIndexByte(body, '\n')
	if idx < 0 {
		v = string(body)
		return
	}
	body = body[0:idx]
	idx = bytes.LastIndexByte(body, '\n')
	if idx < 0 {
		v = string(body)
		return
	}
	v = string(body[idx+1:])
	if v == "" {
		err = errors.Warning("forg: get version from goproxy failed").WithCause(errors.Warning("forg: version was not found")).WithMeta("path", path)
		return
	}
	return
}
