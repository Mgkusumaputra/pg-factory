package environment

import "os"

func detectDevNull(shellEnv string) string {
	switch shellEnv {
	case "cmd", "powershell":
		return "NUL"
	default:
		if _, err := os.Stat("/dev/null"); os.IsNotExist(err) {
			return "NUL" // fallback if somehow /dev/null doesn't exist
		}
		return "/dev/null"
	}
}
