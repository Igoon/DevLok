//go:build windows || darwin

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/energye/systray"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"devlok/internal/config"
	dbpkg "devlok/internal/db"
	"devlok/internal/devlokerr"
	"devlok/internal/editor"
	"devlok/internal/phpcfg"
	"devlok/internal/platform"
	"devlok/internal/stack"
	"devlok/internal/update"
	"devlok/internal/usercfg"
)

//go:embed assets/running.ico
var runningIcon []byte

//go:embed assets/stopped.ico
var stoppedIcon []byte

var (
	trayReady atomic.Bool
	quitting  atomic.Bool
	trayBusy  atomic.Bool
)

var (
	trayStackMu    sync.Mutex
	trayStackItem  *systray.MenuItem
	trayStackLabel string
	trayMySQLMu    sync.Mutex
	trayMySQLItem  *systray.MenuItem
	trayMySQLLabel string
	phpExtMenuMu   sync.Mutex
	phpExtMenu     = map[string]*systray.MenuItem{}
	phpQuickMu     sync.Mutex
	phpQuickMenu   = map[string]map[string]*systray.MenuItem{}
	trayStatusMu   sync.Mutex
	trayStatusItem *systray.MenuItem
	trayStatusText string
	trayUpdateMu   sync.Mutex
	trayUpdateItem *systray.MenuItem
)

type trayService struct {
	Name  string
	Label string
}

var trayServices = []trayService{
	{Name: "nginx", Label: "Nginx"},
	{Name: "mysql", Label: "MySQL"},
	{Name: "node", Label: "Node.js"},
	{Name: "php", Label: "PHP"},
}

var (
	serviceMenuMu    sync.Mutex
	serviceMenuItems = map[string]map[string]*systray.MenuItem{}
	serviceMenuOrder = map[string][]string{}
	siteMenuMu       sync.Mutex
	siteMenuItems    = map[string]*systray.MenuItem{}
)

func (a *App) beforeClose(ctx context.Context) bool {
	if quitting.Load() {
		return false
	}
	if !trayReady.Load() {
		return false
	}
	wruntime.WindowHide(ctx)
	return true
}

func (a *App) startTray() {
	go systray.Run(a.onTrayReady, a.onTrayExit)
}

func (a *App) stopTray() {
	systray.Quit()
}

func (a *App) onTrayExit() {
	trayReady.Store(false)
	trayLog("exit")
}

func (a *App) onTrayReady() {
	trayReady.Store(true)
	trayLog("ready")
	systray.SetIcon(stoppedIcon)
	systray.SetTitle("DevLok")
	systray.SetTooltip("DevLok - stopped")
	systray.SetOnClick(func(menu systray.IMenu) {
		wruntime.WindowShow(a.ctx)
	})
	systray.SetOnDClick(func(menu systray.IMenu) {
		wruntime.WindowShow(a.ctx)
	})
	systray.SetOnRClick(func(menu systray.IMenu) {
		if menu != nil {
			_ = menu.ShowMenu()
		}
	})

	mStatus := systray.AddMenuItem("DevLok - stopped", "Status DevLok")
	mStatus.Disable()
	trayStatusMu.Lock()
	trayStatusItem = mStatus
	trayStatusText = "DevLok - stopped"
	trayStatusMu.Unlock()
	mUpdates := systray.AddMenuItem("Updates", "Update runtime tersedia")
	mUpdates.Hide()
	trayUpdateMu.Lock()
	trayUpdateItem = mUpdates
	trayUpdateMu.Unlock()
	mUpdates.Click(func() {
		go a.refreshTrayUpdates(true)
	})
	mShow := systray.AddMenuItem("Show DevLok", "Buka jendela DevLok")
	mSite := systray.AddMenuItem("Open site", "Buka site yang aktif")
	a.buildSiteMenu()
	a.buildServiceMenus()
	a.syncPHPExtensions()
	a.syncPHPQuick()
	a.syncTrayStatus()
	go a.refreshTrayUpdates(false)
	systray.AddSeparator()
	label := "Start All"
	if anyServicesRunning() {
		label = "Stop All"
	}
	mStack := systray.AddMenuItem(label, "Jalankan atau hentikan semua service")
	trayStackMu.Lock()
	trayStackItem = mStack
	trayStackLabel = label
	trayStackMu.Unlock()
	mRestart := systray.AddMenuItem("Restart Stack", "Hentikan lalu jalankan ulang stack")
	systray.AddSeparator()
	tools := systray.AddMenuItem("Tools", "Aksi tambahan")
	tools.AddSubMenuItem("Open project folder", "Buka folder project aktif").Click(func() {
		go a.trayOpenProjectFolder()
	})
	tools.AddSubMenuItem("Open terminal", "Buka terminal di folder project").Click(func() {
		go a.trayOpenProjectTerminal()
	})
	tools.AddSubMenuItem("Open editor", "Buka project di editor").Click(func() {
		go a.trayOpenProjectEditor()
	})
	tools.AddSubMenuItem("Copy site URL", "Salin URL site aktif").Click(func() {
		go a.trayCopySiteURL()
	})
	tools.AddSubMenuItem("Open config folder", "Buka folder override DevLok").Click(func() {
		go a.trayOpenConfigFolder()
	})
	tools.AddSubMenuItem("Open log folder", "Buka folder log DevLok").Click(func() {
		go a.trayOpenLogFolder()
	})
	tools.AddSubMenuItem("Check for updates", "Periksa update aplikasi dan runtime").Click(func() {
		go a.refreshTrayUpdates(true)
	})
	tools.AddSubMenuItem("Download app update", "Unduh dan verifikasi update aplikasi").Click(func() {
		go a.trayDownloadAppUpdate()
	})
	tools.AddSubMenuItem("Apply app update", "Jalankan installer update aplikasi").Click(func() {
		go a.trayApplyAppUpdate()
	})
	tools.AddSubMenuItem("Rollback app update", "Pulihkan binary aplikasi sebelumnya").Click(func() {
		go a.trayRollbackAppUpdate()
	})
	tools.AddSubMenuItem("About DevLok", "Tentang aplikasi").Click(func() {
		go a.trayAbout()
	})
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Keluar dari DevLok; stack tetap berjalan")

	mShow.Click(func() {
		wruntime.WindowShow(a.ctx)
	})
	mSite.Click(func() {
		st, err := stack.LoadState("")
		if err != nil || st.URL == "" {
			a.trayInfo("Open site", "Stack belum berjalan. Gunakan Start All.")
			return
		}
		_ = platform.Current().OpenBrowser(st.URL)
	})
	mStack.Click(func() {
		go func() {
			if !trayBusy.CompareAndSwap(false, true) {
				return
			}
			defer trayBusy.Store(false)
			if anyServicesRunning() {
				if err := stack.Down(""); err != nil {
					trayLog("stop all: " + err.Error())
				}
			} else {
				a.trayStartAll()
			}
			a.syncTrayStackMenu()
		}()
	})
	mRestart.Click(func() {
		go a.trayRestartStack()
	})
	mQuit.Click(func() {
		a.quitApp()
	})

	go a.trayStatusLoop()
}

func (a *App) trayStatusLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	tick := 0
	runningIconShown := false
	for {
		select {
		case <-a.ctx.Done():
			systray.Quit()
			return
		case <-ticker.C:
			tick++
			if consumeShowRequest() {
				wruntime.WindowShow(a.ctx)
			}
			if consumeQuitRequest() {
				a.quitApp()
				return
			}
			a.syncTrayStackMenu()
			a.syncMySQLMenu()
			a.syncTrayStatus()
			running := mainServicesRunning()
			if running != runningIconShown {
				if running {
					systray.SetIcon(runningIcon)
				} else {
					systray.SetIcon(stoppedIcon)
				}
				runningIconShown = running
			}
			if tick%300 == 0 {
				go a.refreshTrayUpdates(false)
			}
			if tick%5 == 0 {
				a.syncServiceMenus()
				a.syncPHPExtensions()
				a.syncPHPQuick()
				a.syncSiteMenu()
				if st, err := stack.LoadState(""); err == nil && st.Domain != "" {
					status := "partial"
					if running {
						status = "running"
					}
					systray.SetTooltip("DevLok - " + st.Domain + " " + status)
				} else {
					systray.SetTooltip("DevLok - stopped")
				}
			}
		}
	}
}

func (a *App) buildServiceMenus() {
	serviceMenuMu.Lock()
	defer serviceMenuMu.Unlock()
	for _, svc := range trayServices {
		versions, err := a.ServiceVersions(svc.Name)
		if err != nil || len(versions) == 0 {
			continue
		}
		switch svc.Name {
		case "mysql":
			a.buildMySQLMenu(versions)
		case "php":
			a.buildPHPMenu(versions)
		default:
			parent := systray.AddMenuItem(svc.Label, "Pilih versi "+svc.Label)
			a.addVersionItemsLocked(parent, svc.Name, versions)
			if svc.Name == "nginx" {
				parent.AddSubMenuItem("Site config (custom)", "Konfigurasi Nginx tambahan untuk site aktif").Click(func() {
					go a.trayOpenNginxCustom()
				})
				parent.AddSubMenuItem("error.log", "Buka log error Nginx").Click(func() {
					go a.trayOpenNginxLog("error")
				})
				parent.AddSubMenuItem("access.log", "Buka log akses Nginx").Click(func() {
					go a.trayOpenNginxLog("access")
				})
			}
		}
	}
	a.syncServiceMenusLocked()
}

func (a *App) addVersionItemsLocked(parent *systray.MenuItem, service string, versions []string) {
	items := map[string]*systray.MenuItem{}
	for _, v := range versions {
		version := v
		label := serviceLabel(service) + " " + version
		item := parent.AddSubMenuItemCheckbox(label, "Gunakan "+label, false)
		items[version] = item
		item.Click(func() {
			go a.traySetServiceVersion(service, version)
		})
	}
	serviceMenuItems[service] = items
	serviceMenuOrder[service] = versions
}

func serviceLabel(service string) string {
	for _, svc := range trayServices {
		if svc.Name == service {
			return svc.Label
		}
	}
	return service
}

func (a *App) buildMySQLMenu(versions []string) {
	parent := systray.AddMenuItem("MySQL", "Kelola MySQL")
	parent.AddSubMenuItem("phpMyAdmin", "Buka phpMyAdmin").Click(func() {
		go a.trayOpenPHPMyAdmin()
	})
	label := "Start MySQL"
	if mysqlServiceRunning() {
		label = "Stop MySQL"
	}
	toggle := parent.AddSubMenuItem(label, "Jalankan atau hentikan MySQL")
	trayMySQLMu.Lock()
	trayMySQLItem = toggle
	trayMySQLLabel = label
	trayMySQLMu.Unlock()
	toggle.Click(func() {
		go a.trayToggleMySQL()
	})

	versionMenu := parent.AddSubMenuItem("Version", "Pilih versi MySQL")
	a.addVersionItemsLocked(versionMenu, "mysql", versions)

	parent.AddSubMenuItem("Create database", "Buat database project aktif").Click(func() {
		go a.trayCreateDatabase()
	})
	rootPw := parent.AddSubMenuItem("Root password", "Kelola password root MySQL")
	rootPw.AddSubMenuItem("Get root password", "Salin password root MySQL").Click(func() {
		go a.trayGetRootPassword()
	})
	rootPw.AddSubMenuItem("Change root password", "Ganti password root MySQL").Click(func() {
		go a.trayChangeRootPassword()
	})
	parent.AddSubMenuItem("Backup all databases", "Backup semua database").Click(func() {
		go a.trayBackupAllDatabases()
	})
	parent.AddSubMenuItem("my.ini (custom)", "Buka konfigurasi MySQL tambahan").Click(func() {
		go a.trayOpenMySQLFile("my.ini")
	})
	parent.AddSubMenuItem("mysqld.log", "Buka log MySQL").Click(func() {
		go a.trayOpenMySQLFile("mysqld.log")
	})
	parent.AddSubMenuItem("dir: data", "Buka folder data MySQL").Click(func() {
		go a.trayOpenMySQLFolder("data")
	})
	parent.AddSubMenuItem("dir: backup", "Buka folder backup MySQL").Click(func() {
		go a.trayOpenMySQLFolder("backup")
	})
}

func (a *App) buildPHPMenu(versions []string) {
	parent := systray.AddMenuItem("PHP", "Kelola PHP")
	versionMenu := parent.AddSubMenuItem("Version", "Pilih versi PHP")
	a.addVersionItemsLocked(versionMenu, "php", versions)

	quick := parent.AddSubMenuItem("Quick settings", "Preset PHP dan Nginx; ditulis ke file custom")
	memory := quick.AddSubMenuItem("Memory limit", "Batas memori PHP")
	for _, value := range []string{"256M", "512M", "1024M"} {
		v := value
		item := memory.AddSubMenuItemCheckbox(v, "memory_limit = "+v, false)
		a.addPHPQuickItem("memory", v, item)
		item.Click(func() {
			go a.traySetPHPQuick("memory", v)
		})
	}
	upload := quick.AddSubMenuItem("Upload limit", "Batas upload PHP")
	for _, value := range []string{"64M", "128M", "256M"} {
		v := value
		item := upload.AddSubMenuItemCheckbox(v, "upload_max_filesize/post_max_size = "+v, false)
		a.addPHPQuickItem("upload", v, item)
		item.Click(func() {
			go a.traySetPHPQuick("upload", v)
		})
	}
	timeout := quick.AddSubMenuItem("Max execution", "Batas waktu eksekusi PHP")
	for _, value := range []string{"120", "300", "600"} {
		v := value
		item := timeout.AddSubMenuItemCheckbox(v+"s", "max_execution_time = "+v, false)
		a.addPHPQuickItem("timeout", v, item)
		item.Click(func() {
			go a.traySetPHPQuick("timeout", v)
		})
	}
	quick.AddSubMenuItem("Restart PHP", "Restart stack untuk menerapkan konfigurasi").Click(func() {
		go a.trayRestartStack()
	})

	ext := parent.AddSubMenuItem("Extensions", "Ekstensi PHP")
	for _, name := range phpcfg.DefaultExtensions {
		extension := name
		item := ext.AddSubMenuItemCheckbox(extension, "Aktifkan atau nonaktifkan "+extension, false)
		phpExtMenuMu.Lock()
		phpExtMenu[extension] = item
		phpExtMenuMu.Unlock()
		item.Click(func() {
			go a.trayTogglePHPExtension(extension)
		})
	}
	ext.AddSubMenuItem("Buka folder ext", "Folder ekstensi PHP").Click(func() {
		go a.trayOpenPHPFile("ext")
	})

	parent.AddSubMenuItem("php.ini (custom)", "Buka konfigurasi PHP tambahan").Click(func() {
		go a.trayOpenPHPFile("ini")
	})
	parent.AddSubMenuItem("dir: ext", "Buka folder ekstensi PHP").Click(func() {
		go a.trayOpenPHPFile("ext")
	})
	parent.AddSubMenuItem("php_errors.log", "Buka log error PHP").Click(func() {
		go a.trayOpenPHPFile("log")
	})
}

const (
	phpQuickStart = "; >>> DevLok quick settings >>>"
	phpQuickEnd   = "; <<< DevLok quick settings <<<"
)

func (a *App) addPHPQuickItem(kind, value string, item *systray.MenuItem) {
	phpQuickMu.Lock()
	defer phpQuickMu.Unlock()
	if phpQuickMenu[kind] == nil {
		phpQuickMenu[kind] = map[string]*systray.MenuItem{}
	}
	phpQuickMenu[kind][value] = item
}

func (a *App) phpQuickValues() (string, map[string]string, error) {
	exact, err := a.trayPHPExactVersion()
	if err != nil {
		return "", nil, err
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", nil, err
	}
	path := usercfg.Path(state, "php", exact+".ini")
	data, err := usercfg.Read(path)
	if err != nil {
		return "", nil, err
	}
	values := map[string]string{"memory": "256M", "upload": "64M", "timeout": "120"}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if i := strings.IndexAny(value, ";#"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		switch strings.TrimSpace(key) {
		case "memory_limit":
			values["memory"] = value
		case "upload_max_filesize":
			values["upload"] = value
		case "max_execution_time":
			values["timeout"] = value
		}
	}
	return exact, values, nil
}
func (a *App) traySetPHPQuick(kind, value string) {
	exact, values, err := a.phpQuickValues()
	if err != nil {
		a.trayError("Quick settings", err)
		return
	}
	values[kind] = value
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Quick settings", err)
		return
	}
	path := usercfg.Path(state, "php", exact+".ini")
	header := "# DevLok PHP override\n"
	if err := usercfg.Ensure(path, header); err != nil {
		a.trayError("Quick settings", err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.trayError("Quick settings", err)
		return
	}
	body := stripPHPQuickBlock(string(data))
	block := fmt.Sprintf("%s\nmemory_limit = %s\nupload_max_filesize = %s\npost_max_size = %s\nmax_execution_time = %s\nmax_input_time = %s\n%s\n",
		phpQuickStart, values["memory"], values["upload"], values["upload"], values["timeout"], values["timeout"], phpQuickEnd)
	out := strings.TrimRight(body, "\r\n")
	if out != "" {
		out += "\n\n"
	}
	out += block
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		a.trayError("Quick settings", err)
		return
	}
	if m, err := loadManifest(a.trayProjectPath()); err == nil {
		if err := a.writeNginxQuick(m.Name, values["upload"], values["timeout"]); err != nil {
			a.trayError("Quick settings", err)
			return
		}
	}
	a.syncPHPQuick()
	if _, err := stack.LoadState(""); err == nil {
		if a.trayRestartStack() {
			a.trayInfo("Quick settings", "PHP "+exact+" dan Nginx diperbarui. Stack direstart.")
		}
		return
	}
	a.trayInfo("Quick settings", "PHP "+exact+" diperbarui. Berlaku saat stack dijalankan.")
}

func stripPHPQuickBlock(text string) string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == phpQuickStart {
			inBlock = true
			continue
		}
		if trimmed == phpQuickEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

const (
	nginxQuickStart = "# >>> DevLok quick settings >>>"
	nginxQuickEnd   = "# <<< DevLok quick settings <<<"
)

func (a *App) writeNginxQuick(site, upload, timeout string) error {
	state, err := platform.Current().StateDir()
	if err != nil {
		return err
	}
	path := usercfg.Path(state, "nginx", site+".conf")
	header := "# DevLok Nginx override untuk site " + site + "\n"
	if err := usercfg.Ensure(path, header); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var lines []string
	inBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == nginxQuickStart {
			inBlock = true
			continue
		}
		if trimmed == nginxQuickEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			lines = append(lines, line)
		}
	}
	body := strings.TrimRight(strings.Join(lines, "\n"), "\r\n")
	if body != "" {
		body += "\n\n"
	}
	body += fmt.Sprintf("%s\nclient_max_body_size %s;\nfastcgi_read_timeout %ss;\n%s\n",
		nginxQuickStart, strings.ToLower(upload), timeout, nginxQuickEnd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	return nil
}
func (a *App) syncPHPQuick() {
	_, values, err := a.phpQuickValues()
	if err != nil {
		return
	}
	phpQuickMu.Lock()
	defer phpQuickMu.Unlock()
	for kind, items := range phpQuickMenu {
		for value, item := range items {
			if values[kind] == value {
				item.Check()
			} else {
				item.Uncheck()
			}
		}
	}
}

func (a *App) syncTrayStatus() {
	text := "DevLok - stopped"
	if st, err := stack.LoadState(""); err == nil && st.Domain != "" {
		state := "partial"
		if mainServicesRunning() {
			state = "running"
		}
		text = st.Domain + " - " + state
	}
	trayStatusMu.Lock()
	defer trayStatusMu.Unlock()
	if trayStatusItem == nil || trayStatusText == text {
		return
	}
	trayStatusItem.SetTitle(text)
	trayStatusText = text
}
func (a *App) refreshTrayUpdates(manual bool) {
	mgr, err := a.runtimeManager()
	if err != nil {
		if manual {
			a.trayError("Update", err)
		}
		return
	}
	updates, err := update.Check(mgr)
	if err != nil {
		if manual {
			a.trayError("Update", err)
		} else {
			trayLog("update check: " + err.Error())
		}
		return
	}
	var appUpdate *update.AppUpdate
	remoteErr := ""
	if state, err := platform.Current().StateDir(); err == nil {
		cfg, cfgErr := update.LoadConfig(state)
		if cfgErr != nil {
			remoteErr = cfgErr.Error()
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			appUpdate, err = update.AppUpdateCheck(ctx, cfg)
			cancel()
			if err != nil {
				remoteErr = err.Error()
			}
		}
	}
	if remoteErr != "" {
		trayLog("update app: " + remoteErr)
	}
	count := len(updates)
	if appUpdate != nil {
		count++
	}
	trayUpdateMu.Lock()
	if trayUpdateItem != nil {
		if count == 0 {
			trayUpdateItem.Hide()
		} else {
			trayUpdateItem.SetTitle(fmt.Sprintf("Updates (%d)", count))
			trayUpdateItem.SetTooltip("Update aplikasi atau runtime tersedia")
			trayUpdateItem.Show()
		}
	}
	trayUpdateMu.Unlock()
	if !manual {
		return
	}
	if count == 0 {
		msg := "Semua runtime terpasang sudah memakai versi terbaru dari registry."
		if remoteErr != "" {
			msg += "\n\nUpdate aplikasi gagal diperiksa: " + remoteErr
		}
		a.trayInfo("Update", msg)
		return
	}
	lines := make([]string, 0, count)
	if appUpdate != nil {
		line := fmt.Sprintf("DevLok %s -> %s", appUpdate.Current, appUpdate.Latest)
		if appUpdate.Severity != "" {
			line += " (" + appUpdate.Severity + ")"
		}
		lines = append(lines, line)
	}
	for _, u := range updates {
		line := fmt.Sprintf("%s %s -> %s", u.Name, u.Installed, u.Latest)
		if u.Kind == update.KindNew {
			line += " (seri baru " + u.Series + ")"
		}
		lines = append(lines, line)
	}
	msg := strings.Join(lines, "\n")
	if appUpdate != nil && appUpdate.URL != "" {
		msg += "\n\nUnduh aplikasi:\n" + appUpdate.URL
	}
	msg += "\n\nRuntime: dl runtime add <nama>@<versi>"
	if remoteErr != "" {
		msg += "\n\nPeringatan update aplikasi: " + remoteErr
	}
	a.trayInfo("Update tersedia", msg)
}
func (a *App) checkAppUpdate(state string) (*update.AppUpdate, error) {
	cfg, err := update.LoadConfig(state)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return update.AppUpdateCheck(ctx, cfg)
}
func (a *App) trayDownloadAppUpdate() {
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Update", err)
		return
	}
	appUpdate, err := a.checkAppUpdate(state)
	if err != nil {
		a.trayError("Update", err)
		return
	}
	if appUpdate == nil {
		a.trayInfo("Update", "Tidak ada update aplikasi.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	path, err := update.Download(ctx, appUpdate, filepath.Join(state, "cache", "updates"))
	if err != nil {
		a.trayError("Update", err)
		return
	}
	a.trayInfo("Update", "Update "+appUpdate.Latest+" diunduh dan terverifikasi:\n\n"+path+"\n\nInstaller belum dijalankan.")
	_ = platform.Current().OpenFolder(filepath.Dir(path))
}
func (a *App) trayApplyAppUpdate() {
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Update", err)
		return
	}
	appUpdate, err := a.checkAppUpdate(state)
	if err != nil {
		a.trayError("Update", err)
		return
	}
	if appUpdate == nil {
		a.trayInfo("Update", "Tidak ada update aplikasi.")
		return
	}
	if !a.trayConfirm("Jalankan installer DevLok " + appUpdate.Latest + "?\n\nStack akan dihentikan dan DevLok akan ditutup. Installer dijalankan setelah aplikasi keluar.") {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	path, err := update.Download(ctx, appUpdate, filepath.Join(state, "cache", "updates"))
	if err != nil {
		a.trayError("Update", err)
		return
	}
	if err := stack.Down(""); err != nil {
		a.trayError("Update", err)
		return
	}
	if err := update.LaunchInstallerDetached(path); err != nil {
		a.trayError("Update", err)
		return
	}
	trayLog("apply update " + appUpdate.Latest + ": " + path)
	a.quitApp()
}
func (a *App) trayRollbackAppUpdate() {
	exe, err := os.Executable()
	if err != nil {
		a.trayError("Rollback", err)
		return
	}
	appDir := filepath.Dir(exe)
	backup := update.RollbackBackupDir(appDir)
	if _, err := os.Stat(filepath.Join(backup, "devlok-gui.exe")); err != nil {
		a.trayInfo("Rollback", "Backup aplikasi sebelumnya tidak ditemukan.\n\nRollback hanya tersedia setelah upgrade melalui installer DevLok.")
		return
	}
	if !a.trayConfirm("Rollback ke binary DevLok sebelumnya?\n\nStack akan dihentikan, GUI ditutup, binary dipulihkan, lalu GUI dijalankan kembali.") {
		return
	}
	if err := stack.Down(""); err != nil {
		a.trayError("Rollback", err)
		return
	}
	if err := update.ScheduleRollback(appDir); err != nil {
		a.trayError("Rollback", err)
		return
	}
	trayLog("rollback app from " + backup)
	a.quitApp()
}
func (a *App) traySetServiceVersion(service, version string) {
	project := lastProjectPath()
	if st, err := stack.LoadState(""); err == nil && st.ProjectDir != "" {
		project = st.ProjectDir
	}
	if project == "" {
		wruntime.WindowShow(a.ctx)
		return
	}
	running := false
	if _, err := stack.LoadState(""); err == nil {
		running = true
	}
	current := currentServiceVersion(service)
	same := version == current || (current != "" && strings.HasPrefix(version, current+"."))
	if err := a.SetServiceVersion(project, service, version); err != nil {
		trayLog("set version: " + err.Error())
		return
	}
	if running && !same && service != "node" {
		_ = stack.Down("")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		_, _ = stack.Up(ctx, stack.Options{ProjectDir: project})
	}
	a.syncServiceMenus()
}

func (a *App) syncServiceMenus() {
	serviceMenuMu.Lock()
	defer serviceMenuMu.Unlock()
	a.syncServiceMenusLocked()
}

func (a *App) syncServiceMenusLocked() {
	for service, items := range serviceMenuItems {
		current := currentServiceVersion(service)
		if current == "" && service == "nginx" {
			if order := serviceMenuOrder[service]; len(order) > 0 {
				current = order[0]
			}
		}
		for version, item := range items {
			if version == current || (current != "" && strings.HasPrefix(version, current+".")) {
				item.Check()
			} else {
				item.Uncheck()
			}
		}
	}
}

func currentServiceVersion(service string) string {
	project := lastProjectPath()
	if st, err := stack.LoadState(""); err == nil && st.ProjectDir != "" {
		project = st.ProjectDir
	}
	if project == "" {
		return ""
	}
	m, err := loadManifest(project)
	if err != nil {
		return ""
	}
	switch service {
	case "php":
		if v := m.Web.PHP; v != "" {
			return v
		}
	case "nginx":
		if v := m.Runtime["nginx"]; v != "" {
			return v
		}
	case "node":
		if v := m.Runtime["node"]; v != "" {
			return v
		}
	case "mysql":
		if v := m.Services["mysql"]; v != "" {
			return v
		}
	}
	return ""
}

func (a *App) syncMySQLMenu() {
	label := "Start MySQL"
	if mysqlServiceRunning() {
		label = "Stop MySQL"
	}
	trayMySQLMu.Lock()
	defer trayMySQLMu.Unlock()
	if trayMySQLItem == nil || trayMySQLLabel == label {
		return
	}
	trayMySQLItem.SetTitle(label)
	trayMySQLLabel = label
}

func mysqlServiceRunning() bool {
	services, _ := stack.Services("")
	for _, s := range services {
		if strings.HasPrefix(s.Name, "mysql-") && s.State == "running" {
			return true
		}
	}
	return false
}

func (a *App) trayToggleMySQL() {
	if mysqlServiceRunning() {
		a.trayStopMySQL()
	} else {
		a.trayStartMySQL()
	}
	a.syncMySQLMenu()
}

func (a *App) trayStartMySQL() {
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("MySQL", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := mgr.Ensure(ctx); err != nil {
		a.trayError("MySQL gagal dijalankan", err)
		return
	}
	a.trayInfo("MySQL", fmt.Sprintf("MySQL berjalan di 127.0.0.1:%d", mgr.Port))
}

func (a *App) trayStopMySQL() {
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("MySQL", err)
		return
	}
	if err := mgr.Stop(); err != nil {
		a.trayError("MySQL gagal dihentikan", err)
		return
	}
	a.trayInfo("MySQL", "MySQL dihentikan.")
}

func (a *App) trayProjectPath() string {
	if st, err := stack.LoadState(""); err == nil && st.ProjectDir != "" {
		return st.ProjectDir
	}
	if p := lastProjectPath(); p != "" {
		if _, err := loadManifest(p); err == nil {
			return p
		}
	}
	if sites, err := a.Sites(); err == nil && len(sites) > 0 {
		return sites[0].Path
	}
	return ""
}

func (a *App) trayDBManager() (*dbpkg.Manager, *config.Manifest, string, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, nil, "", err
	}
	project := a.trayProjectPath()
	m, err := loadManifest(project)
	if err != nil {
		return nil, nil, project, err
	}
	version := m.Services["mysql"]
	if version == "" {
		version = "8.4"
	}
	mgr, err := dbpkg.New(dbpkg.Options{StateDir: state, Version: version, Port: m.Ports.MySQL})
	if err != nil {
		return nil, nil, project, err
	}
	return mgr, m, project, nil
}

func (a *App) trayOpenPHPMyAdmin() {
	url := activePHPMyAdminURL()
	if url == "" {
		a.trayInfo("phpMyAdmin", "Jalankan stack terlebih dahulu dengan Start All.")
		return
	}
	_ = platform.Current().OpenBrowser(url)
}

func (a *App) trayCreateDatabase() {
	mgr, m, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("Create database", err)
		return
	}
	if m.DB.Name == "" {
		a.trayError("Create database", fmt.Errorf("nama database kosong di manifest"))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	dc, err := mgr.EnsureDatabase(ctx, m.Name, m.DB.Name)
	if err != nil {
		a.trayError("Create database", err)
		return
	}
	a.trayInfo("Database siap", fmt.Sprintf("Database: %s\nUser: %s\nPort: %d\nPassword: %s", dc.Name, dc.User, mgr.Port, dc.Password))
}

func (a *App) trayGetRootPassword() {
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("Root password", err)
		return
	}
	pw, err := mgr.RootPassword()
	if err != nil {
		a.trayError("Root password", err)
		return
	}
	if pw == "" {
		a.trayInfo("Root password", "Root password belum dibuat. Jalankan MySQL atau Start All.")
		return
	}
	_ = wruntime.ClipboardSetText(a.ctx, pw)
	a.trayInfo("Root password", "Password root disalin ke clipboard:\n\n"+pw)
}

func (a *App) trayChangeRootPassword() {
	if !a.trayConfirm("Ganti root password MySQL? Password lama tidak dapat dikembalikan.") {
		return
	}
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("Change root password", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pw, err := mgr.ChangeRootPassword(ctx)
	if err != nil {
		a.trayError("Change root password", err)
		return
	}
	_ = wruntime.ClipboardSetText(a.ctx, pw)
	a.trayInfo("Root password", "Password root baru disalin ke clipboard:\n\n"+pw)
}

func (a *App) trayBackupAllDatabases() {
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("Backup", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	dir, files, err := mgr.BackupAll(ctx, "")
	if err != nil {
		a.trayError("Backup", err)
		return
	}
	a.trayInfo("Backup selesai", fmt.Sprintf("%d database dibackup ke:\n%s", len(files), dir))
	_ = platform.Current().OpenFolder(dir)
}

func (a *App) trayOpenMySQLFile(kind string) {
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("MySQL", err)
		return
	}
	switch kind {
	case "my.ini":
		header := "# DevLok MySQL override\n# File ini dibaca lewat !include di my.ini.\n# Letakkan setting di bawah [mysqld].\n[mysqld]\n"
		if err := usercfg.Ensure(mgr.CustomConfigPath, header); err != nil {
			a.trayError("MySQL", err)
			return
		}
		a.trayOpenFile(mgr.CustomConfigPath, "Konfigurasi MySQL custom belum tersedia.")
	case "mysqld.log":
		a.trayOpenFile(mgr.LogPath, "Log MySQL belum tersedia.")
	}
}

func (a *App) trayOpenMySQLFolder(kind string) {
	mgr, _, _, err := a.trayDBManager()
	if err != nil {
		a.trayError("MySQL", err)
		return
	}
	path := mgr.DataDir
	if kind == "backup" {
		path = filepath.Join(mgr.StateDir, "backups", "mysql")
	}
	a.trayOpenFolder(path)
}

func (a *App) trayPHPExactVersion() (string, error) {
	m, err := loadManifest(a.trayProjectPath())
	if err != nil {
		return "", err
	}
	version := m.Web.PHP
	if version == "" {
		version = m.Runtime["php"]
	}
	if version == "" {
		version = "8.4"
	}
	runtimes, err := a.RuntimeList()
	if err != nil {
		return "", err
	}
	for _, r := range runtimes {
		if r.Name == "php" && (r.Version == version || strings.HasPrefix(r.Version, version+".")) {
			return r.Version, nil
		}
	}
	for _, r := range runtimes {
		if r.Name == "php" {
			return r.Version, nil
		}
	}
	return "", fmt.Errorf("runtime PHP belum terpasang")
}

func (a *App) trayPHPPaths() (string, string, string, error) {
	exact, err := a.trayPHPExactVersion()
	if err != nil {
		return "", "", "", err
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", "", "", err
	}
	return usercfg.Path(state, "php", exact+".ini"),
		filepath.Join(state, "runtimes", "php", exact, "ext"),
		filepath.Join(state, "logs", "php-error.log"), nil
}

func (a *App) phpExtensionState() (string, map[string]bool, string, error) {
	exact, err := a.trayPHPExactVersion()
	if err != nil {
		return "", nil, "", err
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", nil, "", err
	}
	extDir := filepath.Join(state, "runtimes", "php", exact, "ext")
	enabled := map[string]bool{}
	for _, name := range phpcfg.Load(state, exact) {
		enabled[name] = true
	}
	return exact, enabled, extDir, nil
}

func (a *App) trayTogglePHPExtension(name string) {
	exact, enabled, extDir, err := a.phpExtensionState()
	if err != nil {
		a.trayError("PHP extension", err)
		return
	}
	if _, err := os.Stat(filepath.Join(extDir, "php_"+name+".dll")); err != nil {
		a.trayInfo("PHP extension", "Ekstensi "+name+" tidak tersedia di PHP "+exact+".")
		return
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("PHP extension", err)
		return
	}
	list := make([]string, 0, len(enabled)+1)
	if enabled[name] {
		delete(enabled, name)
	} else {
		enabled[name] = true
	}
	for ext := range enabled {
		list = append(list, ext)
	}
	if err := phpcfg.Save(state, exact, list); err != nil {
		a.trayError("PHP extension", err)
		return
	}
	a.syncPHPExtensions()
	action := "diaktifkan"
	if !enabled[name] {
		action = "dinonaktifkan"
	}
	if _, err := stack.LoadState(""); err == nil {
		if a.trayRestartStack() {
			a.trayInfo("PHP extension", name+" "+action+" untuk PHP "+exact+". Stack direstart.")
		}
		return
	}
	a.trayInfo("PHP extension", name+" "+action+" untuk PHP "+exact+". Berlaku saat stack dijalankan.")
}

func (a *App) syncPHPExtensions() {
	_, enabled, _, err := a.phpExtensionState()
	if err != nil {
		return
	}
	phpExtMenuMu.Lock()
	defer phpExtMenuMu.Unlock()
	for name, item := range phpExtMenu {
		if enabled[name] {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func (a *App) trayOpenPHPFile(kind string) {
	ini, ext, logPath, err := a.trayPHPPaths()
	if err != nil {
		a.trayError("PHP", err)
		return
	}
	switch kind {
	case "ini":
		header := "# DevLok PHP override\n# File ini ditambahkan ke akhir php.ini setiap Start All.\n# Contoh:\n# memory_limit = 512M\n# upload_max_filesize = 128M\n# post_max_size = 128M\n# max_execution_time = 300\n"
		if err := usercfg.Ensure(ini, header); err != nil {
			a.trayError("PHP", err)
			return
		}
		a.trayOpenFile(ini, "Konfigurasi PHP custom belum tersedia.")
	case "ext":
		a.trayOpenFolder(ext)
	case "log":
		a.trayOpenFile(logPath, "Log PHP belum tersedia.")
	}
}

func (a *App) trayOpenNginxCustom() {
	m, err := loadManifest(a.trayProjectPath())
	if err != nil {
		a.trayError("Nginx", err)
		return
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Nginx", err)
		return
	}
	path := usercfg.Path(state, "nginx", m.Name+".conf")
	header := "# DevLok Nginx override untuk site " + m.Name + "\n# Di-include di dalam blok server.\n# Contoh:\n# client_max_body_size 128m;\n# fastcgi_read_timeout 300s;\n"
	if err := usercfg.Ensure(path, header); err != nil {
		a.trayError("Nginx", err)
		return
	}
	a.trayOpenFile(path, "Konfigurasi Nginx custom belum tersedia.")
}

func (a *App) trayOpenNginxLog(kind string) {
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Nginx", err)
		return
	}
	name := "error.log"
	if kind == "access" {
		name = "access.log"
	}
	a.trayOpenFile(filepath.Join(state, "logs", "nginx", name), "Log Nginx belum tersedia.")
}

func (a *App) trayOpenProjectFolder() {
	project := a.trayProjectPath()
	if project == "" {
		a.trayInfo("Project", "Belum ada site aktif.")
		return
	}
	a.trayOpenFolder(project)
}

func (a *App) trayOpenProjectTerminal() {
	project := a.trayProjectPath()
	if project == "" {
		a.trayInfo("Terminal", "Belum ada site aktif.")
		return
	}
	if err := platform.Current().OpenTerminal(project, nil); err != nil {
		a.trayError("Terminal", err)
	}
}

func (a *App) trayOpenProjectEditor() {
	project := a.trayProjectPath()
	if project == "" {
		a.trayInfo("Editor", "Belum ada site aktif.")
		return
	}
	if err := editor.Open(project); err != nil {
		a.trayError("Editor", err)
	}
}

func (a *App) trayCopySiteURL() {
	st, err := stack.LoadState("")
	if err != nil || st.URL == "" {
		a.trayInfo("Copy site URL", "Stack belum berjalan.")
		return
	}
	_ = wruntime.ClipboardSetText(a.ctx, st.URL)
	a.trayInfo("Copy site URL", st.URL+" disalin ke clipboard.")
}

func (a *App) trayOpenConfigFolder() {
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Config", err)
		return
	}
	a.trayOpenFolder(filepath.Join(state, "config"))
}

func (a *App) trayOpenLogFolder() {
	state, err := platform.Current().StateDir()
	if err != nil {
		a.trayError("Log", err)
		return
	}
	a.trayOpenFolder(filepath.Join(state, "logs"))
}

func (a *App) trayAbout() {
	state, _ := platform.Current().StateDir()
	msg := "DevLok " + a.Version() + "\n\nLocal development environment untuk Nginx, PHP, dan MySQL."
	if state != "" {
		msg += "\n\nState: " + state
	}
	a.trayInfo("About DevLok", msg)
}
func (a *App) trayRestartStack() bool {
	project := a.trayProjectPath()
	if project == "" {
		a.trayInfo("Restart Stack", "Pilih atau buat site terlebih dahulu.")
		return false
	}
	if _, err := stack.LoadState(""); err != nil {
		a.trayInfo("Restart Stack", "Stack belum berjalan.")
		return false
	}
	if !trayBusy.CompareAndSwap(false, true) {
		return false
	}
	defer trayBusy.Store(false)
	_ = stack.Down("")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if _, err := stack.Up(ctx, stack.Options{ProjectDir: project}); err != nil {
		a.trayError("Restart Stack", err)
		return false
	}
	return true
}

func (a *App) trayOpenFile(path, missing string) {
	if path == "" {
		a.trayInfo("DevLok", missing)
		return
	}
	if _, err := os.Stat(path); err != nil {
		a.trayInfo("DevLok", missing+"\n\n"+path)
		return
	}
	if err := editor.Open(path); err != nil {
		a.trayError("Buka file", err)
	}
}

func (a *App) trayOpenFolder(path string) {
	if path == "" {
		a.trayError("Buka folder", fmt.Errorf("path kosong"))
		return
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		a.trayError("Buka folder", err)
		return
	}
	if err := platform.Current().OpenFolder(path); err != nil {
		a.trayError("Buka folder", err)
	}
}

func (a *App) trayInfo(title, message string) {
	_, _ = wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
		Type:    wruntime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

func (a *App) trayError(title string, err error) {
	msg := err.Error()
	if e, ok := devlokerr.As(err); ok && e.Hint != "" {
		msg += "\n\n" + e.Hint
	}
	trayLog(title + ": " + err.Error())
	_, _ = wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
		Type:    wruntime.ErrorDialog,
		Title:   title,
		Message: msg,
	})
}

func (a *App) trayConfirm(message string) bool {
	res, err := wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
		Type:          wruntime.QuestionDialog,
		Title:         "DevLok",
		Message:       message,
		Buttons:       []string{"Ya", "Batal"},
		DefaultButton: "Batal",
	})
	return err == nil && res == "Ya"
}

func (a *App) syncTrayStackMenu() {
	label := "Start All"
	if anyServicesRunning() {
		label = "Stop All"
	}
	trayStackMu.Lock()
	defer trayStackMu.Unlock()
	if trayStackItem == nil || trayStackLabel == label {
		return
	}
	trayStackItem.SetTitle(label)
	trayStackItem.SetTooltip("Jalankan atau hentikan semua service")
	trayStackLabel = label
}

func (a *App) trayStartAll() {
	project := lastProjectPath()
	if _, err := loadManifest(project); err != nil {
		project = ""
	}
	if project == "" {
		sites, err := a.Sites()
		if err != nil || len(sites) == 0 {
			wruntime.WindowShow(a.ctx)
			return
		}
		project = sites[0].Path
	}
	saveLastProject(project)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if _, err := stack.Up(ctx, stack.Options{ProjectDir: project}); err != nil {
		trayLog("start all: " + err.Error())
	}
}

func anyServicesRunning() bool {
	services, _ := stack.Services("")
	for _, s := range services {
		if s.State == "running" {
			return true
		}
	}
	return false
}

func mainServicesRunning() bool {
	st, err := stack.LoadState("")
	if err != nil {
		return false
	}
	services, _ := stack.Services("")
	have := map[string]bool{}
	for _, s := range services {
		if s.State != "running" {
			continue
		}
		if strings.HasPrefix(s.Name, "mysql-") {
			have["mysql"] = true
		} else {
			have[s.Name] = true
		}
	}
	if !have["nginx"] || !have["php"] {
		return false
	}
	dbEnabled := true
	if m, err := loadManifest(st.ProjectDir); err == nil {
		dbEnabled = m.ComponentEnabled("mysql")
	}
	if dbEnabled && !have["mysql"] {
		return false
	}
	return true
}

func activePHPMyAdminURL() string {
	st, err := stack.LoadState("")
	if err != nil {
		return ""
	}
	https := strings.HasPrefix(st.URL, "https://")
	scheme := "http"
	port := st.HTTPPort
	if https {
		scheme = "https"
		port = st.HTTPSPort
	}
	if m, err := config.Load(filepath.Join(st.ProjectDir, config.FileName)); err == nil {
		if !m.ComponentEnabled("mysql") || !m.ComponentEnabled("phpmyadmin") {
			return ""
		}
	}
	host := "phpmyadmin.test"
	if (scheme == "https" && port != 443) || (scheme == "http" && port != 80) {
		host += ":" + strconv.Itoa(port)
	}
	return scheme + "://" + host
}

func consumeShowRequest() bool {
	state, err := platform.Current().StateDir()
	if err != nil {
		return false
	}
	path := filepath.Join(state, "state", "show.request")
	if _, err := os.Stat(path); err != nil {
		return false
	}
	_ = os.Remove(path)
	return true
}

func trayLog(msg string) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return
	}
	dir := filepath.Join(state, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "gui.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s tray %s\n", time.Now().Format(time.RFC3339), msg)
}

func (a *App) quitApp() {
	quitting.Store(true)
	go func() {
		systray.Quit()
		wruntime.Quit(a.ctx)
	}()
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
}

func consumeQuitRequest() bool {
	state, err := platform.Current().StateDir()
	if err != nil {
		return false
	}
	path := filepath.Join(state, "state", "quit.request")
	if _, err := os.Stat(path); err != nil {
		return false
	}
	_ = os.Remove(path)
	return true
}

func (a *App) buildSiteMenu() {
	sites, err := a.Sites()
	if err != nil || len(sites) == 0 {
		return
	}
	parent := systray.AddMenuItem("Sites", "Daftar site")
	siteMenuMu.Lock()
	defer siteMenuMu.Unlock()
	for _, site := range sites {
		path := site.Path
		item := parent.AddSubMenuItemCheckbox(site.Name+" - "+site.Domain, path, site.Active)
		siteMenuItems[path] = item
		item.Click(func() {
			go a.trayOpenSite(path)
		})
	}
}

func (a *App) trayOpenSite(path string) {
	if st, err := stack.LoadState(""); err == nil {
		if sameTrayPath(st.ProjectDir, path) {
			wruntime.WindowShow(a.ctx)
			_ = platform.Current().OpenBrowser(st.URL)
			return
		}
		_ = stack.Down("")
	}
	saveLastProject(path)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if _, err := stack.Up(ctx, stack.Options{ProjectDir: path}); err != nil {
		trayLog("open site: " + err.Error())
	}
	a.syncSiteMenu()
}

func (a *App) syncSiteMenu() {
	active := ""
	if st, err := stack.LoadState(""); err == nil {
		active = st.ProjectDir
	}
	siteMenuMu.Lock()
	defer siteMenuMu.Unlock()
	for path, item := range siteMenuItems {
		if sameTrayPath(path, active) {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func sameTrayPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}
