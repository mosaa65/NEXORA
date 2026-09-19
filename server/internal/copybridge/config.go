package copybridge

import (
	"os"
	"path/filepath"
	"strings"
)

// Config contains the standalone local copy-bridge configuration. The bridge is
// intentionally localhost-only by default so USB devices remain scoped to the
// current PC instead of being exposed through the central NEXORA server.
type Config struct {
	HTTPAddr            string
	CORSOrigins         []string
	TempDir             string
	AndroidTargetFolder string
	IOSBundleID         string
	// CommandToken, when set, is required in the Authorization header for every
	// mutating command (copy, mkdir, eject, cancel).
	//
	// Binding to loopback keeps the bridge off the network, but it does NOT stop
	// a local process, or a web page served from a permitted LAN origin, from
	// issuing copy/eject commands. A token closes that gap.
	//
	// It is optional on purpose: requiring it by default would break every
	// existing single-PC install on upgrade. When it is empty the bridge behaves
	// exactly as before and logs a warning once at startup.
	CommandToken string
}

func LoadConfig() Config {
	loadDotEnv()
	tempDir := strings.TrimSpace(os.Getenv("NEXORA_COPY_BRIDGE_TEMP_DIR"))
	if tempDir == "" {
		tempDir = filepath.Join(os.TempDir(), "nexora-copybridge")
	}
	return Config{
		HTTPAddr:            envString("NEXORA_COPY_BRIDGE_ADDR", "127.0.0.1:32145"),
		CORSOrigins:         splitCSV(os.Getenv("NEXORA_COPY_BRIDGE_CORS_ORIGIN")),
		TempDir:             tempDir,
		AndroidTargetFolder: envString("NEXORA_ANDROID_TARGET", "Download"),
		IOSBundleID:         envString("NEXORA_IOS_BUNDLE_ID", "org.videolan.vlc-ios"),
		CommandToken:        strings.TrimSpace(os.Getenv("NEXORA_COPY_BRIDGE_TOKEN")),
	}
}

// splitCSV parses a comma-separated env value into a trimmed, non-empty list.
// A missing/empty value yields nil, which selects the default CORS policy.
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv() {
	candidates := []string{".env", "server/.env", "../.env", "../../.env"}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`+"\r")
			if _, exists := os.LookupEnv(key); !exists || strings.TrimSpace(os.Getenv(key)) == "" {
				_ = os.Setenv(key, value)
			}
		}
		break
	}
}
