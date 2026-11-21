package selfupdate

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Modes of GET /api/update.
const (
	ModeSelf   = "self"   // the button: download, verify, replace, restart
	ModeManual = "manual" // no button: the state carries commands to copy
	ModeOff    = "off"    // nothing to offer
)

// Install kinds.
const (
	InstallBinary     = "binary"
	InstallDocker     = "docker"
	InstallKubernetes = "kubernetes"
)

// Setting values of update.self_update.
const (
	SettingAuto = "auto" // self when safe: a container, a read-only dir or an open port without auth give manual
	SettingOn   = "on"   // also self on an open port without auth
	SettingOff  = "off"  // no update from the UI
)

const (
	installSh  = "https://raw.githubusercontent.com/buggregator/server/master/scripts/install.sh"
	installPs1 = "https://raw.githubusercontent.com/buggregator/server/master/scripts/install.ps1"
)

// Facts are what the mode is decided from. Detect fills them for the running
// process; tests build them by hand.
type Facts struct {
	Version   string // running version, "dev" for a build without one
	Release   bool   // Version is a release version
	CheckOn   bool   // update_check.enabled
	Setting   string // update.self_update
	AuthOn    bool
	HTTPAddr  string
	GOOS      string
	Exe       string // resolved path of the running binary
	Container string // "", InstallDocker or InstallKubernetes
	Writable  bool   // the binary's dir takes a new file
}

// Mode is the decision: what the UI may offer and why.
type Mode struct {
	Mode          string
	Reason        string
	Install       string
	RestartManual bool
}

// Command is one line the user copies for a manual update.
type Command struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Command string `json:"command"`
}

// Decide picks the mode. Order: off reasons, then where it runs, then safety.
func Decide(f Facts) Mode {
	m := Mode{Install: InstallBinary}
	switch f.Container {
	case InstallDocker, InstallKubernetes:
		m.Install = f.Container
	}
	switch {
	case !f.CheckOn:
		m.Mode, m.Reason = ModeOff, "the update check is off: set update_check.enabled: true (env UPDATE_CHECK_ENABLED=true)"
		return m
	case !f.Release:
		m.Mode, m.Reason = ModeOff, "this build has no release version ("+f.Version+")"
		return m
	case f.Setting == SettingOff:
		m.Mode, m.Reason = ModeOff, "updates from the UI are off: set update.self_update: auto (env UPDATE_SELF=auto)"
		return m
	}
	m.Mode = ModeManual
	switch {
	case m.Install == InstallKubernetes:
		m.Reason = "running in Kubernetes: upgrade the Helm release"
	case m.Install == InstallDocker:
		m.Reason = "running in a container: pull the new image and re-create the container"
	case !f.Writable:
		m.Reason = "the server cannot write to " + filepath.Dir(f.Exe) + ": update it from a terminal"
	case f.Setting != SettingOn && !f.AuthOn && !LoopbackOnly(f.HTTPAddr):
		m.Reason = "the server listens on " + f.HTTPAddr + " without auth: set update.self_update: on (env UPDATE_SELF=on) to allow updates from the UI, or listen on 127.0.0.1"
	default:
		m.Mode, m.Reason = ModeSelf, ""
		m.RestartManual = f.GOOS == "windows"
	}
	return m
}

// Commands are the manual update lines for the install kind; version is the
// release to install ("" = latest).
func Commands(install, goos, exe, version string) []Command {
	tag := version
	if tag == "" {
		tag = "latest"
	}
	switch install {
	case InstallDocker:
		return []Command{
			{ID: "compose", Label: "Docker Compose (image tag latest; a pinned tag: set it to " + tag + " first)", Command: "docker compose pull && docker compose up -d"},
			{ID: "docker", Label: "Docker (then re-create the container)", Command: "docker pull ghcr.io/buggregator/server:" + tag},
		}
	case InstallKubernetes:
		return []Command{
			// The UI shows labels only as tabs (2+ commands): the second one
			// says which Helm the first needs.
			{ID: "helm", Label: "Helm 3.14+", Command: "helm upgrade buggregator oci://ghcr.io/buggregator/helm-chart/buggregator --reset-then-reuse-values --set image.tag=" + tag},
			{ID: "helm-old", Label: "Older Helm (keeps the old chart's defaults)", Command: "helm upgrade buggregator oci://ghcr.io/buggregator/helm-chart/buggregator --reuse-values --set image.tag=" + tag},
		}
	}
	dir := filepath.Dir(exe)
	if goos == "windows" {
		// PowerShell single quotes: a quote is doubled.
		return []Command{{ID: "powershell", Label: "PowerShell", Command: "$env:BUGGREGATOR_INSTALL_DIR='" + strings.ReplaceAll(dir, "'", "''") + "'; irm " + installPs1 + " | iex"}}
	}
	// sh single quotes: a quote ends the string, is escaped, starts it again.
	return []Command{{ID: "shell", Label: "Shell", Command: "curl -fsSL " + installSh + " | BUGGREGATOR_INSTALL_DIR='" + strings.ReplaceAll(dir, "'", `'\''`) + "' sh"}}
}

// loopbackHost reports whether a request's Host names this machine
// (localhost, 127.0.0.1, [::1]); a DNS-rebinding page sends its own name.
func loopbackHost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return LoopbackOnly(host)
}

// LoopbackOnly reports whether a listen address takes local connections only.
// ":8000" and "0.0.0.0:8000" listen on every interface.
func LoopbackOnly(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// container tells where the process runs.
func container() string {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return InstallKubernetes
	}
	for _, f := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(f); err == nil {
			return InstallDocker
		}
	}
	return ""
}

// writable probes the dir with a temp file.
func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".buggregator-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// executable is the resolved path of the running binary.
func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return exe, nil
}

// SameOrigin reports whether a request comes from this server's own page or
// from no page at all (curl, scripts); for other modules' write routes.
func SameOrigin(r *http.Request) bool { return sameOrigin(r) }

// LoopbackHost reports whether a request's Host names this machine.
func LoopbackHost(host string) bool { return loopbackHost(host) }
