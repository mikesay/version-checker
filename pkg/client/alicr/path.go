package alicr

import (
	"regexp"
	"strings"
)

var (
	alicrPattern = regexp.MustCompile(`^registry\.(.*)\.aliyuncs\.com$`)
)

func (c *Client) IsHost(host string) bool {
	return alicrPattern.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return "acs", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
