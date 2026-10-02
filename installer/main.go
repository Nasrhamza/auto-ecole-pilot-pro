//go:build windows

package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	appName       = "AutoÉcole Pilot Pro v1"
	appExeName    = "AutoEcole-Pilot-Pro-v1.exe"
	uninstallName = "Désinstaller AutoEcole Pilot Pro v1.exe"
	registryKey   = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\AutoEcolePilotProV1`
)

//go:embed payload/AutoEcole-Pilot-Pro-v1.exe payload/MicrosoftEdgeWebview2Setup.exe
var payloadFS embed.FS

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	messageBoxProc = user32.NewProc("MessageBoxW")
	moveFileProc   = kernel32.NewProc("MoveFileExW")
)

func main() {
	exe, _ := os.Executable()
	if strings.EqualFold(filepath.Base(exe), uninstallName) {
		uninstall(exe)
		return
	}
	install(exe)
}

func install(setupPath string) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		showError("Le dossier LOCALAPPDATA est introuvable.")
		return
	}
	installDir := filepath.Join(localAppData, "Programs", "AutoEcolePilotProV1")
	target := filepath.Join(installDir, appExeName)
	uninstaller := filepath.Join(installDir, uninstallName)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		showError("Impossible de créer le dossier d'installation : " + err.Error())
		return
	}
	payload, err := payloadFS.ReadFile("payload/" + appExeName)
	if err != nil {
		showError("Le fichier du logiciel est absent du Setup : " + err.Error())
		return
	}
	staged := target + ".new"
	if err := os.WriteFile(staged, payload, 0755); err != nil {
		showError("Impossible de préparer le logiciel : " + err.Error())
		return
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(staged)
		showError("Fermez AutoÉcole Pilot Pro v1 puis relancez le Setup.")
		return
	}
	if err := os.Rename(staged, target); err != nil {
		_ = os.Remove(staged)
		showError("Impossible d'installer le logiciel : " + err.Error())
		return
	}
	setupBytes, err := os.ReadFile(setupPath)
	if err != nil || os.WriteFile(uninstaller, setupBytes, 0755) != nil {
		showError("Logiciel installé, mais la désinstallation n'a pas pu être préparée.")
		return
	}
	_ = os.WriteFile(filepath.Join(installDir, "version.txt"), []byte("1.0\r\n"), 0644)
	if os.Getenv("AUTOECOLE_INSTALLER_QA") == "1" {
		return
	}
	if !webView2RuntimeInstalled() {
		if err := installWebView2Runtime(); err != nil {
			showError("Microsoft WebView2 n'a pas pu être installé. Vérifiez la connexion Internet puis relancez le Setup : " + err.Error())
			return
		}
	}
	if err := createShortcuts(target, installDir); err != nil {
		showError("Logiciel installé, mais les raccourcis n'ont pas pu être créés : " + err.Error())
		return
	}
	registerUninstaller(target, installDir, uninstaller, len(payload))
	if messageBox("Installation terminée", appName+" est installé.\n\nVoulez-vous ouvrir le logiciel maintenant ?", 0x00000004|0x00000040) == 6 {
		_ = exec.Command(target).Start()
	}
}

func webView2RuntimeInstalled() bool {
	keys := []string{
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
		`HKLM\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
		`HKCU\Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
	}
	for _, key := range keys {
		out, err := exec.Command("reg.exe", "QUERY", key, "/v", "pv").Output()
		if err == nil && strings.Contains(string(out), "REG_SZ") && !strings.Contains(string(out), "0.0.0.0") {
			return true
		}
	}
	return false
}

func installWebView2Runtime() error {
	payload, err := payloadFS.ReadFile("payload/MicrosoftEdgeWebview2Setup.exe")
	if err != nil {
		return err
	}
	bootstrapper, err := os.CreateTemp("", "autoecole-webview2-*.exe")
	if err != nil {
		return err
	}
	bootstrapperPath := bootstrapper.Name()
	defer os.Remove(bootstrapperPath)
	if _, err := bootstrapper.Write(payload); err != nil {
		_ = bootstrapper.Close()
		return err
	}
	if err := bootstrapper.Close(); err != nil {
		return err
	}
	cmd := exec.Command(bootstrapperPath, "/silent", "/install")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return err
	}
	if !webView2RuntimeInstalled() {
		return fmt.Errorf("runtime introuvable après installation")
	}
	return nil
}

func uninstall(currentExe string) {
	if messageBox("Désinstaller", "Voulez-vous désinstaller "+appName+" ?\n\nVos données de gestion seront conservées.", 0x00000004|0x00000030) != 6 {
		return
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	installDir := filepath.Join(localAppData, "Programs", "AutoEcolePilotProV1")
	expectedExe := filepath.Join(installDir, uninstallName)
	actualExe, _ := filepath.Abs(currentExe)
	expectedExe, _ = filepath.Abs(expectedExe)
	if localAppData == "" || !strings.EqualFold(actualExe, expectedExe) {
		showError("Chemin de désinstallation invalide.")
		return
	}
	_ = exec.Command("taskkill.exe", "/IM", appExeName, "/F").Run()
	_ = os.Remove(filepath.Join(installDir, appExeName))
	_ = os.Remove(filepath.Join(installDir, "version.txt"))
	_ = removeShortcuts()
	_ = exec.Command("reg.exe", "DELETE", registryKey, "/f").Run()
	scheduleDeleteOnReboot(currentExe)
	messageBox("Désinstallation terminée", "Le logiciel a été retiré.\nVos données locales ont été conservées pour éviter toute perte.", 0x00000040)
}

func createShortcuts(target, workingDir string) error {
	script := fmt.Sprintf(`
$ws = New-Object -ComObject WScript.Shell
$desktop = [Environment]::GetFolderPath('Desktop')
$programs = [Environment]::GetFolderPath('Programs')
foreach ($path in @((Join-Path $desktop 'AutoÉcole Pilot Pro v1.lnk'),(Join-Path $programs 'AutoÉcole Pilot Pro v1.lnk'))) {
  $shortcut = $ws.CreateShortcut($path)
  $shortcut.TargetPath = '%s'
  $shortcut.WorkingDirectory = '%s'
  $shortcut.IconLocation = '%s,0'
  $shortcut.Description = 'Gestion complète d auto-école'
  $shortcut.Save()
}
`, psQuote(target), psQuote(workingDir), psQuote(target))
	return runPowerShell(script)
}

func removeShortcuts() error {
	script := `
$desktop = [Environment]::GetFolderPath('Desktop')
$programs = [Environment]::GetFolderPath('Programs')
foreach ($path in @((Join-Path $desktop 'AutoÉcole Pilot Pro v1.lnk'),(Join-Path $programs 'AutoÉcole Pilot Pro v1.lnk'))) {
  Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
}
`
	return runPowerShell(script)
}

func registerUninstaller(target, installDir, uninstaller string, payloadSize int) {
	values := [][4]string{
		{"DisplayName", "REG_SZ", appName, ""},
		{"DisplayVersion", "REG_SZ", "1.0", ""},
		{"Publisher", "REG_SZ", "AutoÉcole Pilot Pro", ""},
		{"InstallLocation", "REG_SZ", installDir, ""},
		{"DisplayIcon", "REG_SZ", target + ",0", ""},
		{"UninstallString", "REG_SZ", `"` + uninstaller + `"`, ""},
		{"NoModify", "REG_DWORD", "1", ""},
		{"NoRepair", "REG_DWORD", "1", ""},
		{"EstimatedSize", "REG_DWORD", fmt.Sprintf("%d", (payloadSize+1023)/1024), ""},
	}
	for _, value := range values {
		_ = exec.Command("reg.exe", "ADD", registryKey, "/v", value[0], "/t", value[1], "/d", value[2], "/f").Run()
	}
}

func runPowerShell(script string) error {
	runes := utf16.Encode([]rune(script))
	bytes := make([]byte, len(runes)*2)
	for i, r := range runes {
		bytes[i*2] = byte(r)
		bytes[i*2+1] = byte(r >> 8)
	}
	encoded := base64.StdEncoding.EncodeToString(bytes)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encoded)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func psQuote(s string) string { return strings.ReplaceAll(s, "'", "''") }

func scheduleDeleteOnReboot(path string) {
	p, err := syscall.UTF16PtrFromString(path)
	if err == nil {
		_, _, _ = moveFileProc.Call(uintptr(unsafe.Pointer(p)), 0, 0x00000004)
	}
}

func showError(text string) { messageBox("Erreur d'installation", text, 0x00000010) }

func messageBox(title, text string, flags uintptr) int {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)
	result, _, _ := messageBoxProc.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), flags)
	return int(result)
}
