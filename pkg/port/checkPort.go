package port

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func checkLocalPort(port int) bool {
	// Run netstat -ano
	cmd := exec.Command("netstat", "-ano")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	re := regexp.MustCompile(fmt.Sprintf(`:%d\s+.*LISTENING`, port))
	for scanner.Scan() {
		if re.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}
