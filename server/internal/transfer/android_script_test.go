package transfer

import (
	"os/exec"
	"strings"
	"testing"
)

// TestGeneratedScriptParses asks PowerShell whether every script the backend
// can generate is syntactically valid, using PowerShell's own parser.
//
// It exists because a live call once returned PowerShell's "Missing closing '}'"
// for a script that looked fine in the source: the builder had joined a `try`
// block to a `catch` with one brace too few. A string assertion would not have
// caught it; the interpreter that runs the script does. This covers a real
// Android device being absent, because it parses without executing.
func TestGeneratedScriptParses(t *testing.T) {
	b := NewAndroidBackend()
	b.deviceName = "Some Phone"

	// Each entry mirrors a call site, including the templates built inline
	// inside List, Delete and Put.
	listScript := b.connectPreamble() + b.folderWalkScript([]string{"Movies"}, false) + `foreach ($item in $folder.Items()) {
    $isFolder = $false
    if ($item.IsFolder) { $isFolder = $true }
    $size = 0
    if (-not $isFolder) {
        $size = $item.ExtendedProperty('System.Size')
        if (-not $size) { $size = $item.Size }
        if (-not $size) { $size = 0 }
    }
    $kind = 'F'
    if ($isFolder) { $kind = 'D' }
    Write-Output ('ENTRY:' + $kind + '|' + $size + '|' + $item.Name)
}
`

	deleteScript := b.connectPreamble() + b.folderWalkScript([]string{"Movies"}, false) + `$target = $null
foreach ($item in $folder.Items()) {
    if ($item.Name -eq 'a.mkv') { $target = $item; break }
}
if (-not $target) { Write-Output 'MISSING'; exit 0 }
try { $target.InvokeVerb('delete') } catch { Write-Output 'MISSING'; exit 0 }
Write-Output 'OK'
`

	putScript := b.connectPreamble() + b.folderWalkScript([]string{"Movies", "Season 1"}, true) + `foreach ($item in $folder.Items()) {
    if ($item.Name -eq 'a.mkv') {
        try { $item.InvokeVerb('delete') } catch {}
        Start-Sleep -Milliseconds 300
        break
    }
}
$folder.CopyHere('C:\a.mkv', 16)
Write-Output 'OK'
`

	scripts := map[string]string{
	"walk-no-create": b.connectPreamble() + b.folderWalkScript([]string{"Movies"}, false),
	"walk-create":    b.connectPreamble() + b.folderWalkScript([]string{"Movies", "Season 1"}, true),
	"walk-empty":     b.connectPreamble() + b.folderWalkScript(nil, false),
	"list":           listScript,
	"delete":         deleteScript,
	"put":            putScript,
	"free-space": b.connectPreamble() + `$free = $null
try { $free = $folder.ExtendedProperty('System.FreeSpace') } catch {}
if (-not $free) { Write-Output 'MISSING'; exit 0 }
Write-Output ('FREE:' + $free)
`,
	}

	for name, script := range scripts {
		t.Run(name, func(t *testing.T) {
			// Parse only — never execute — so this needs no device.
			check := "$tokens=$null; $errors=$null; " +
				"[void][System.Management.Automation.Language.Parser]::ParseInput(" +
				"@'\n" + script + "\n'@, [ref]$tokens, [ref]$errors); " +
				"if ($errors -and $errors.Count -gt 0) { $errors | ForEach-Object { Write-Output $_.Message }; exit 1 }"

			out, err := exec.Command(powershellPath(), "-NoProfile", "-NonInteractive", "-Command", check).CombinedOutput()
			if err != nil {
				t.Errorf("generated script %q does not parse: %v\n%s", name, err, string(out))
			}
	})
	}
}

// TestGeneratedScriptsHaveBalancedBraces is the cheap guard that does not need
// PowerShell: every generated script must close exactly the blocks it opens.
// It runs everywhere, including a host without PowerShell.
func TestGeneratedScriptsHaveBalancedBraces(t *testing.T) {
	b := NewAndroidBackend()
	b.deviceName = "Some Phone"

	scripts := map[string]string{
	"walk": b.connectPreamble() + b.folderWalkScript([]string{"Movies"}, true),
	"list": b.connectPreamble() + b.folderWalkScript(nil, false),
	}

	for name, script := range scripts {
		t.Run(name, func(t *testing.T) {
			opens := strings.Count(script, "{")
			closes := strings.Count(script, "}")
			if opens != closes {
				t.Errorf("unbalanced braces: %d opened, %d closed", opens, closes)
			}
			// Single-quoted literals are the only quotes the builder emits, and
			// an odd count means one was left unterminated.
			if strings.Count(script, "'")%2 != 0 {
				t.Errorf("odd number of single quotes: %d", strings.Count(script, "'"))
			}
	})
	}
}
