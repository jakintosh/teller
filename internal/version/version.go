package version

import (
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// The following variables are populated by the linker via -ldflags.
// They intentionally use package scope so callers only interact through Info().
var (
	rawVersion = "dev"
	rawCommit  = ""
	rawDate    = ""
)

// Info captures the build metadata for the binary.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
}

var (
	infoOnce sync.Once
	info     Info
)

// Data exposes the best-effort build metadata for the binary. It prefers values
// injected at link time but will fall back to go build debug information when
// available and finally to development defaults.
func Data() Info {
	infoOnce.Do(func() {
		info = Info{
			Version:   normalize(rawVersion),
			Commit:    normalize(rawCommit),
			BuildDate: normalize(rawDate),
		}

		buildInfo, ok := debug.ReadBuildInfo()
		if ok {
			// Prefer module version from go install / release builds.
			if shouldReplaceVersion(info.Version) && validVersion(buildInfo.Main.Version) {
				info.Version = buildInfo.Main.Version
			}

			if shouldReplace(info.Commit) {
				if rev := setting(buildInfo.Settings, "vcs.revision"); rev != "" {
					info.Commit = rev
					if mod := setting(buildInfo.Settings, "vcs.modified"); mod == "true" {
						info.Commit += "-dirty"
					}
				}
			}

			if shouldReplace(info.BuildDate) {
				if t := setting(buildInfo.Settings, "vcs.time"); t != "" {
					if parsed, err := time.Parse(time.RFC3339, t); err == nil {
						info.BuildDate = parsed.UTC().Format(time.RFC3339)
					} else {
						info.BuildDate = t
					}
				}
			}
		}

		if shouldReplaceVersion(info.Version) {
			info.Version = "dev"
		}

		if shouldReplace(info.Commit) {
			info.Commit = "unknown"
		} else {
			info.Commit = shortenCommit(info.Commit)
		}

		if shouldReplace(info.BuildDate) {
			info.BuildDate = "unknown"
		}
	})

	return info
}

func setting(settings []debug.BuildSetting, key string) string {
	for _, s := range settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

func normalize(input string) string {
	return strings.TrimSpace(input)
}

func shouldReplace(value string) bool {
	return value == ""
}

func shouldReplaceVersion(value string) bool {
	return value == "" || value == "dev" || value == "(devel)"
}

func validVersion(value string) bool {
	if value == "" || value == "(devel)" {
		return false
	}
	return strings.HasPrefix(value, "v")
}

func shortenCommit(commit string) string {
	const shortLen = 12
	if len(commit) <= shortLen {
		return commit
	}
	return commit[:shortLen]
}
