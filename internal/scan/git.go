package scan

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var remoteSectionRE = regexp.MustCompile(`^\[remote\s+"(.+)"\]$`)

func extractGitRemote(workspacePath string) (remoteURL, owner string) {
	gitCfg := filepath.Join(workspacePath, ".git", "config")
	f, err := os.Open(gitCfg)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	currentRemote := ""
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentRemote = ""
			if m := remoteSectionRE.FindStringSubmatch(line); len(m) == 2 {
				currentRemote = m[1]
			}
			continue
		}
		if currentRemote == "origin" && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			url := strings.TrimSpace(parts[1])
			return url, ownerFromRemote(url)
		}
	}
	return "", ""
}

func ownerFromRemote(remote string) string {
	remote = strings.TrimSuffix(remote, ".git")
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) != 2 {
			return ""
		}
		path := parts[1]
		segs := strings.Split(path, "/")
		if len(segs) >= 2 {
			return segs[len(segs)-2]
		}
		return ""
	}
	if strings.HasPrefix(remote, "https://") || strings.HasPrefix(remote, "http://") {
		withoutScheme := remote[strings.Index(remote, "//")+2:]
		segs := strings.Split(withoutScheme, "/")
		if len(segs) >= 3 {
			return segs[len(segs)-2]
		}
	}
	return ""
}
