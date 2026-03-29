package environment

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

func detectShellEnv() string {
	// WSL detection via /proc/version
	if runtime.GOOS == "linux" {
		if content, err := os.ReadFile("/proc/version"); err == nil {
			if matched, _ := regexp.MatchString("(?i)(microsoft|wsl)", string(content)); matched {
				return "wsl"
			}
		}
	}

	// Git Bash / MSYS / Cygwin
	msystem := os.Getenv("MSYSTEM")
	if matched, _ := regexp.MatchString("^(MINGW|MSYS|CYGWIN)", msystem); matched {
		return "gitbash"
	}

	// Fallback: uname -s on Linux/macOS
	if output, err := exec.Command("uname", "-s").Output(); err == nil {
		uname := strings.TrimSpace(string(output))
		if strings.HasPrefix(uname, "MINGW") || strings.HasPrefix(uname, "MSYS") {
			return "gitbash"
		}
	}

	// Windows native detection
	if runtime.GOOS == "windows" {
		powershell := os.Getenv("PSModulePath")
		if powershell != "" {
			return "powershell"
		}
		return "cmd"
	}

	// Default fallback
	return "unknown"
}
