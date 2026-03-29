//go:build windows

package cmd

// removePathExport is a no-op on Windows.
// The install.ps1 script adds the Go bin path to the user's PATH via the
// Windows registry ([HKCU\Environment]). Reverting that safely requires
// registry access; we instead print a reminder in the uninstall success message.
func removePathExport() {}
