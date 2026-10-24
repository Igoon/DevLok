<script lang="ts">
  import { onMount, onDestroy } from 'svelte'

  type Status = {
    running: boolean
    health: string
    platform: string
    project: string
    path: string
    domain: string
    url: string
    local_url: string
    db_name: string
    db_user: string
    db_port: number
    php_port: number
    nginx_version: string
    php_version: string
    mysql_version: string
    http_port: number
    https_port: number
    http_listening: boolean
    https_listening: boolean
    database_enabled: boolean
    phpmyadmin_enabled: boolean
    services: any[]
  }

  type Site = {
    name: string
    domain: string
    root: string
    php: string
    path: string
    active: boolean
  }

  let status: Status = { running: false, services: [] } as any
  let sites: Site[] = []
  let runtimes: any[] = []
  let logs = ''
  let logService = 'nginx'
  let version = '0.1.0-dev'
  let selectedPath = ''
  let newName = ''
  let newPhp = '8.4'
  let newWithDB = true
  let dbEnabled = true
  let pmaEnabled = true
  let componentsPath = ''
  let mysqlVersion = '8.4'
  let mysqlVersions: string[] = []
  let busy = ''
  let error = ''
  let notice = ''
  let timer: number | undefined
  let tab = 'dashboard'
  let cfgPath = ''
  let cfgDomain = ''
  let cfgAliases = ''
  let cfgRoot = ''
  let cfgHTTPSEnabled = true
  let cfgDBName = ''
  let cfgHTTP = ''
  let cfgHTTPS = ''
  let cfgMySQLPort = ''
  let cfgPHPPort = ''
  let cfgDB = true
  let cfgPMA = true
  let portChecks: Record<string, any> = {}
  let cfgOptions: Record<string, string[]> = { nginx: [], php: [], mysql: [], node: [] }
  let cfgVersions: Record<string, string> = { nginx: '', php: '', mysql: '', node: '' }
  let cfgSnapshot = ''
  let cfgDirty = false
  let logPaused = false
  let versionLoaded = false
  let importPath = 'C:\laragon'
  let importSites: any[] = []
  let importSelected: Record<string, boolean> = {}
  let importResults: any[] = []
  let importAdopt = true
  let importBusy = false
  let updateInfo: any = null
  let updateBusy = false
  let updateNotice = ''
  let updateProgress: any = null
  let lastUpdateRetry: (() => void) | null = null
  let runtimeCache: any = { bytes: 0, files: 0 }
  let updatePrefs: any = {
    auto_update_app: false,
    auto_update_runtime: false,
    auto_update_runtime_series: false,
    interval_minutes: 360
  }
  let prefsBusy = false
  let prefsNotice = ''
  let refreshToken = 0
  let pmaURL = ''

  $: patchUpdates = updateInfo?.runtimes?.filter((u: any) => u.kind === 'patch') || []

  $: cfgDirty =
    cfgPath !== '' &&
    JSON.stringify({
      domain: cfgDomain,
      aliases: cfgAliases,
      root: cfgRoot,
      httpsEnabled: cfgHTTPSEnabled,
      dbName: cfgDBName,
      http: cfgHTTP,
      https: cfgHTTPS,
      mysql: cfgMySQLPort,
      php: cfgPHPPort,
      db: cfgDB,
      pma: cfgPMA,
      versions: cfgVersions
    }) !== cfgSnapshot

  function parsePort(value: string) {
    const n = Number(value)
    return Number.isFinite(n) && n > 0 ? n : 0
  }

  function toggleNode(on: boolean) {
    if (on) {
      cfgVersions.node = cfgOptions.node[0] || ''
    } else {
      cfgVersions.node = ''
    }
  }

  function matchVersion(options: string[], current: string) {
    if (options.length === 0) return current || ''
    if (!current) return options[0]
    if (options.includes(current)) return current
    return options.find((v) => v.startsWith(current + '.')) || options[0]
  }

  let confirmReq: { title: string; message: string; resolve: (v: boolean) => void } | null = null

  function ask(message: string, title = 'Konfirmasi'): Promise<boolean> {
    return new Promise((resolve) => {
      confirmReq = { title, message, resolve }
    })
  }

  function answerConfirm(value: boolean) {
    if (!confirmReq) return
    const resolve = confirmReq.resolve
    confirmReq = null
    resolve(value)
  }

  async function checkPorts() {
    const list = [cfgHTTP, cfgHTTPS, cfgMySQLPort, cfgPHPPort].map(parsePort).filter((p) => p > 0)
    if (list.length === 0) {
      portChecks = {}
      return
    }
    try {
      const checks = await api().CheckPorts(list)
      portChecks = Object.fromEntries(checks.map((c: any) => [String(c.port), c]))
    } catch {
      portChecks = {}
    }
  }

  function portHint(value: string, autoText: string) {
    const port = parsePort(value)
    if (!port) return autoText
    const check = portChecks[String(port)]
    if (!check) return ''
    if (check.available) return 'free'
    return 'in use' + (check.process ? ': ' + check.process : '')
  }
  async function loadConfig(path: string) {
    if (!path) return
    try {
      const settled = await Promise.allSettled([
        api().ProjectConfig(path),
        api().ServiceVersions('nginx'),
        api().ServiceVersions('php'),
        api().ServiceVersions('mysql'),
        api().ServiceVersions('node'),
        api().RuntimeList(),
        api().RuntimeCache()
      ])
      const val = (i: number) => (settled[i].status === 'fulfilled' ? (settled[i] as any).value : undefined)
      const c = val(0)
      if (!c) throw new Error('Konfigurasi project tidak dapat dimuat.')
      const nginx: string[] = val(1) || []
      const php: string[] = val(2) || []
      const mysql: string[] = val(3) || []
      const node: string[] = val(4) || []
      const rl = val(5) || []
      const cache = val(6) || { bytes: 0, files: 0 }
      cfgPath = path
      cfgDomain = c.domain || ''
      cfgAliases = (c.aliases || []).join(', ')
      cfgRoot = c.root || ''
      cfgHTTPSEnabled = !!c.https
      cfgDBName = c.db_name || ''
      cfgHTTP = c.http_port ? String(c.http_port) : ''
      cfgHTTPS = c.https_port ? String(c.https_port) : ''
      cfgMySQLPort = c.mysql_port ? String(c.mysql_port) : ''
      cfgPHPPort = c.php_port ? String(c.php_port) : ''
      cfgDB = !!c.database
      cfgPMA = !!c.phpmyadmin
      runtimes = rl
      runtimeCache = cache
      cfgOptions = { nginx, php, mysql, node }
      cfgVersions = {
        nginx: matchVersion(nginx, c.versions?.nginx),
        php: matchVersion(php, c.versions?.php),
        mysql: matchVersion(mysql, c.versions?.mysql),
        node: matchVersion(node, c.versions?.node)
      }
      componentsPath = path
      dbEnabled = !!c.database
      pmaEnabled = !!c.phpmyadmin
      mysqlVersion = cfgVersions.mysql
      cfgSnapshot = JSON.stringify({
        domain: cfgDomain,
        aliases: cfgAliases,
        root: cfgRoot,
        httpsEnabled: cfgHTTPSEnabled,
        dbName: cfgDBName,
        http: cfgHTTP,
        https: cfgHTTPS,
        mysql: cfgMySQLPort,
        php: cfgPHPPort,
        db: cfgDB,
        pma: cfgPMA,
        versions: cfgVersions
      })
      await checkPorts()
    } catch (e) {
      error = message(e)
    }
  }

  function openConfig() {
    tab = 'config'
    const path = selectedPath || status.path || sites[0]?.path || ''
    if (path) loadConfig(path)
  }

  function saveConfig() {
    if (!cfgPath) {
      error = 'Pilih site terlebih dahulu.'
      return
    }
    run('Menyimpan konfigurasi...', async () => {
      await api().SaveProjectConfig(cfgPath, {
        domain: cfgDomain,
        root: cfgRoot,
        https: cfgHTTPSEnabled,
        db_name: cfgDBName,
        aliases: cfgAliases.split(',').map((s) => s.trim()).filter(Boolean),
        http_port: parsePort(cfgHTTP),
        https_port: parsePort(cfgHTTPS),
        mysql_port: parsePort(cfgMySQLPort),
        php_port: parsePort(cfgPHPPort),
        versions: cfgVersions,
        database: cfgDB,
        phpmyadmin: cfgPMA
      })
      if (status.running && status.path === cfgPath) {
        await api().Down()
        await api().Up(cfgPath)
      }
      await loadConfig(cfgPath)
      notice = 'Konfigurasi disimpan' + (status.running && status.path === cfgPath ? ' dan stack direstart.' : '.')
    })
  }

  function api() {
    const app = window.go?.main?.App
    if (!app) throw new Error('Binding GUI tidak tersedia. Jalankan lewat devlok-gui.exe.')
    return app
  }

  function message(e: unknown) {
    return e instanceof Error ? e.message : String(e)
  }

  async function refresh(silent = true) {
    const token = ++refreshToken
    try {
      const app = api()
      const [s, si] = await Promise.all([app.Status(), app.Sites()])
      if (token !== refreshToken) return
      status = s
      sites = si
      try { pmaURL = await app.PHPMyAdminURL() } catch { pmaURL = '' }
      if (!versionLoaded) {
        version = await app.Version()
        versionLoaded = true
      }
      if (!selectedPath) {
        if (s.running) selectedPath = s.path
        else if (si.length > 0) selectedPath = si[0].path
      }
      if (!componentsPath) {
        const p = s.running ? s.path : selectedPath
        if (p) await loadComponents(p)
      }
      if (tab === 'dashboard' && s.running && !logPaused) {
        await loadLogs()
      }
      if (!error || !silent) error = ''
    } catch (e) {
      if (!silent) error = message(e)
    }
  }

  async function loadComponents(path: string) {
    if (!path) return
    try {
      const settled = await Promise.allSettled([api().Components(path), api().MySQLVersions()])
      const val = (i: number) => (settled[i].status === 'fulfilled' ? (settled[i] as any).value : undefined)
      const c = val(0)
      if (!c) throw new Error('Komponen project tidak dapat dimuat.')
      const versions: string[] = val(1) || []
      dbEnabled = !!c.database
      pmaEnabled = !!c.phpmyadmin
      mysqlVersions = versions
      mysqlVersion = c.mysql_version || mysqlVersions[0] || '8.4'
      if (mysqlVersion && !mysqlVersions.includes(mysqlVersion)) {
        const match = mysqlVersions.find((v) => v.startsWith(mysqlVersion + '.'))
        if (match) {
          mysqlVersion = match
        } else {
          mysqlVersions = [mysqlVersion, ...mysqlVersions]
        }
      }
      componentsPath = path
    } catch (e) {
      error = message(e)
    }
  }

  function applyComponents() {
    if (!componentsPath) {
      error = 'Pilih site terlebih dahulu.'
      return
    }
    if (!dbEnabled) pmaEnabled = false
    run('Menyimpan komponen...', async () => {
      if (mysqlVersion) {
        await api().SetMySQLVersion(componentsPath, mysqlVersion)
      }
      await api().SetComponents(componentsPath, dbEnabled, pmaEnabled)
      if (status.running && status.path === componentsPath) {
        await api().Down()
        await api().Up(componentsPath)
      }
    })
  }

  async function scanLaragon() {
    importBusy = true
    error = ''
    importResults = []
    try {
      const sites = await api().ScanLaragon(importPath)
      importSites = sites
      importSelected = Object.fromEntries(sites.map((s: any) => [s.name, true]))
    } catch (e) {
      error = message(e)
    } finally {
      importBusy = false
    }
  }

  function toggleImportAll() {
    const all = importSites.every((s) => importSelected[s.name])
    importSelected = Object.fromEntries(importSites.map((s) => [s.name, !all]))
  }

  async function runImport() {
    const names = importSites.filter((s) => importSelected[s.name]).map((s) => s.name)
    if (names.length === 0) {
      error = 'Pilih minimal satu site.'
      return
    }
    importBusy = true
    error = ''
    try {
      importResults = await api().ImportLaragon(importPath, names, importAdopt)
      importSites = importSites.map((s) => {
        const r = importResults.find((x) => x.site.name === s.name)
        return r ? { ...s, action: r.action } : s
      })
      await refresh(false)
    } catch (e) {
      error = message(e)
    } finally {
      importBusy = false
    }
  }
  function toggleDatabase() {
    dbEnabled = !dbEnabled
    applyComponents()
  }

  async function run(label: string, fn: () => Promise<unknown>) {
    if (busy) return
    busy = label
    error = ''
    notice = ''
    try {
      await fn()
      await refresh(false)
    } catch (e) {
      error = message(e)
    } finally {
      busy = ''
    }
  }

  function up(path = '') {
    path = path || selectedPath || status.path || sites[0]?.path || ''
    if (!path) {
      error = 'Pilih site terlebih dahulu.'
      return
    }
    run('Menjalankan stack...', async () => {
      selectedPath = path
      await loadComponents(path)
      const result = await api().Up(path)
      notice = (result?.warnings || []).join(' | ')
      for (let i = 0; i < 8; i++) {
        await new Promise((resolve) => setTimeout(resolve, 400))
        await refresh(true)
        if (status.running) break
      }
      await loadLogs()
    })
  }

  function down() {
    run('Menghentikan stack...', async () => {
      await api().Down()
    })
  }

  function createSite() {
    if (!newName.trim()) {
      error = 'Nama site wajib diisi.'
      return
    }
    run('Membuat site...', async () => {
      const result = await api().CreateSite(newName, newPhp, newWithDB)
      notice = (result?.warnings || []).join(' | ')
      const created = result?.name || newName
      newName = ''
      await refresh(false)
      const found = sites.find((s) => s.name === created)
      if (found) {
        selectedPath = found.path
        await loadComponents(found.path)
      }
      await loadLogs()
    })
  }

  async function loadUpdates() {
    updateBusy = true
    error = ''
    updateNotice = ''
    try {
      const [info, prefs] = await Promise.all([api().UpdateInfo(), api().UpdatePreferences()])
      updateInfo = info
      updatePrefs = prefs
    } catch (e) {
      error = message(e)
    } finally {
      updateBusy = false
    }
  }

  async function saveUpdatePrefs() {
    prefsBusy = true
    error = ''
    prefsNotice = ''
    try {
      await api().SaveUpdatePreferences(updatePrefs)
      prefsNotice = 'Preferensi update disimpan.'
    } catch (e) {
      error = message(e)
    } finally {
      prefsBusy = false
    }
  }

  async function downloadAppUpdate() {
    updateBusy = true
    error = ''
    updateNotice = ''
    lastUpdateRetry = () => downloadAppUpdate()
    try {
      const path = await api().DownloadAppUpdate()
      updateNotice = 'Installer terverifikasi: ' + path
      lastUpdateRetry = null
    } catch (e) {
      error = message(e)
    } finally {
      updateBusy = false
      clearProgressSoon()
    }
  }

  async function applyAppUpdate() {
    if (!(await ask('Jalankan installer update? DevLok akan ditutup dan stack dihentikan.'))) return
    updateBusy = true
    error = ''
    updateNotice = ''
    lastUpdateRetry = () => applyAppUpdate()
    try {
      await api().ApplyAppUpdate()
      updateNotice = 'Installer dijadwalkan. DevLok akan ditutup.'
      lastUpdateRetry = null
    } catch (e) {
      error = message(e)
    } finally {
      updateBusy = false
      clearProgressSoon()
    }
  }

  function cancelUpdate() {
    void api().CancelUpdate()
  }

  async function refreshRuntimes() {
    const [rl, cache] = await Promise.all([api().RuntimeList(), api().RuntimeCache()])
    runtimes = rl
    runtimeCache = cache
  }

  async function cleanupUnused() {
    if (!(await ask('Hapus versi runtime lama yang tidak dipakai project?'))) return
    busy = 'Membersihkan runtime'
    error = ''
    notice = ''
    try {
      notice = await api().CleanupUnusedRuntimes()
      await refreshRuntimes()
    } catch (e) {
      error = message(e)
    } finally {
      busy = ''
    }
  }

  async function cleanupCache() {
    if (!(await ask('Hapus semua arsip runtime di cache? Runtime yang sudah terpasang tetap aman, update berikutnya akan mengunduh ulang.'))) return
    busy = 'Membersihkan cache'
    error = ''
    notice = ''
    try {
      notice = await api().CleanupRuntimeCache()
      await refreshRuntimes()
    } catch (e) {
      error = message(e)
    } finally {
      busy = ''
    }
  }

  async function rollbackAppUpdate() {
    if (!(await ask('Rollback ke binary DevLok sebelumnya? DevLok akan ditutup.'))) return
    updateBusy = true
    error = ''
    updateNotice = ''
    try {
      await api().RollbackAppUpdate()
      updateNotice = 'Rollback dijadwalkan. DevLok akan ditutup.'
    } catch (e) {
      error = message(e)
      updateBusy = false
    }
  }

  function formatBytes(n: number) {
    if (!n || n <= 0) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB']
    let value = n
    let unit = 0
    while (value >= 1024 && unit < units.length - 1) {
      value /= 1024
      unit++
    }
    return value.toFixed(unit === 0 ? 0 : 1) + ' ' + units[unit]
  }

  function progressLabel(p: any) {
    const target = p.name ? p.name + (p.version ? ' ' + p.version : '') : 'DevLok ' + (p.version || '')
    switch (p.phase) {
      case 'start': return 'Menyiapkan ' + target
      case 'download': return 'Mengunduh ' + target
      case 'verify': return 'Memverifikasi checksum ' + target
      case 'extract': return 'Mengekstrak ' + target
      case 'configure': return 'Memperbarui project ke ' + target
      case 'restart': return 'Restart stack'
      case 'installer': return 'Menjalankan installer ' + target
      case 'done': return 'Selesai: ' + target
      case 'error': return 'Gagal: ' + target
      default: return target
    }
  }

  function clearProgressSoon() {
    window.setTimeout(() => {
      if (updateProgress?.phase !== 'error') updateProgress = null
    }, 1500)
  }

  async function updateRuntime(u: any) {
    let extra = u.name === 'mysql' ? '\n\nData MySQL tidak dimigrasikan otomatis. Backup database lama sebelum berpindah versi mayor.' : ''
    if (u.kind === 'new') {
      extra += '\n\nIni seri versi baru (' + u.series + '), bukan patch. Periksa kompatibilitas project sebelum melanjutkan.'
    }
    if (!(await ask('Pasang ' + u.name + ' ' + u.latest + ' dan ubah project aktif?' + extra))) return
    updateBusy = true
    error = ''
    updateNotice = ''
    lastUpdateRetry = () => updateRuntime(u)
    try {
      updateNotice = await api().InstallRuntimeUpdate(u.name, u.latest)
      await loadUpdates()
      await refreshRuntimes()
      lastUpdateRetry = null
    } catch (e) {
      error = message(e)
    } finally {
      updateBusy = false
      clearProgressSoon()
    }
  }

  async function updateAllPatches() {
    if (!(await ask('Pasang semua patch runtime dan perbarui project aktif?'))) return
    updateBusy = true
    error = ''
    updateNotice = ''
    lastUpdateRetry = () => updateAllPatches()
    try {
      updateNotice = await api().UpdateAllPatches()
      await loadUpdates()
      await refreshRuntimes()
      lastUpdateRetry = null
    } catch (e) {
      error = message(e)
    } finally {
      updateBusy = false
      clearProgressSoon()
    }
  }

  async function removeRuntime(r: any) {
    if (!(await ask('Hapus runtime ' + r.name + ' ' + r.version + '?\n\nFile runtime akan dihapus dari disk.'))) return
    busy = 'Hapus runtime'
    error = ''
    notice = ''
    try {
      notice = await api().RemoveRuntime(r.name, r.version)
      await refreshRuntimes()
    } catch (e) {
      error = message(e)
    } finally {
      busy = ''
    }
  }

  function copyRuntimeCommand(u: any) {
    const cmd = status.platform === 'darwin' ? 'brew upgrade ' + u.name : 'dl runtime add ' + u.name + '@' + u.latest
    navigator.clipboard?.writeText(cmd)
    updateNotice = 'Perintah disalin.'
  }

  async function loadLogs() {
    try {
      logs = await api().Logs(logService, 250)
    } catch (e) {
      error = message(e)
    }
  }

  function openDatabase() {
    if (!pmaURL) {
      error = 'URL phpMyAdmin belum tersedia. Jalankan Start All terlebih dahulu.'
      return
    }
    openURL(pmaURL)
  }

  function openURL(url: string) {
    if (!url) return
    run('Membuka browser...', () => api().OpenURL(url))
  }

  function openFolder(path: string) {
    if (!path) return
    run('Membuka folder...', () => api().OpenFolder(path))
  }

  function openEditor(path: string) {
    if (!path) return
    run('Membuka editor...', () => api().OpenEditor(path))
  }

  function openTerminal(path: string) {
    run('Membuka terminal...', () => api().OpenTerminal(path))
  }

  function serviceState(name: string) {
    const list = status.services || []
    const svc = list.find((s: any) => s.name === name || (name === 'mysql' && s.name.startsWith('mysql-')))
    return svc?.state || 'stopped'
  }

  function activePath() {
    return status.path || selectedPath || sites[0]?.path || ''
  }

  function togglePMA() {
    if (!dbEnabled) return
    pmaEnabled = !pmaEnabled
    applyComponents()
  }
  function stateClass(s: string) {
    return ['running', 'stopped', 'failed', 'restarting'].includes(s) ? s : 'stopped'
  }

  $: activeSite = sites.find((s) => s.path === selectedPath) || null

  onMount(() => {
    refresh(false)
    timer = window.setInterval(() => refresh(true), 3000)
    window.runtime?.EventsOn('update:progress', (p: any) => {
      updateProgress = p
      if (p.phase === 'start') updateBusy = true
      if (p.phase === 'done' || p.phase === 'error') updateBusy = false
      if (p.phase === 'done') {
        window.setTimeout(() => {
          if (updateProgress === p) updateProgress = null
        }, 1500)
      }
    })
  })

  onDestroy(() => {
    if (timer) window.clearInterval(timer)
    window.runtime?.EventsOff('update:progress')
  })
</script>

<div class="app">
  <header class="topbar">
    <div class="brand">DevLok <span>local dev</span></div>
    <nav class="tabs">
      <button class:active={tab === 'dashboard'} on:click={() => { tab = 'dashboard'; refresh(true) }}>Dashboard</button>
      <button class:active={tab === 'sites'} on:click={() => (tab = 'sites')}>Sites</button>
      <button class:active={tab === 'logs'} on:click={() => (tab = 'logs')}>Logs</button>
      <button class:active={tab === 'config'} on:click={openConfig}>Konfigurasi</button>
      <button class:active={tab === 'update'} on:click={() => { tab = 'update'; loadUpdates() }}>Update</button>
      <button class:active={tab === 'import'} on:click={() => { tab = 'import'; if (importSites.length === 0) scanLaragon() }}>Import</button>
    </nav>
    <div class="top-right">
      <span class="badge {status.health || (status.running ? 'running' : 'stopped')}">
        <span class="dot"></span>
        {status.health || (status.running ? 'running' : 'stopped')}
      </span>
      <button class="ghost" on:click={() => refresh(false)} disabled={!!busy}>Refresh</button>
    </div>
  </header>

  {#if error}
    <div class="error-box">{error}</div>
  {/if}
  {#if notice}
    <div class="notice-box">{notice}</div>
  {/if}

  {#if confirmReq}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="modal-overlay" on:click={() => answerConfirm(false)}>
      <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
      <div class="modal" role="dialog" aria-modal="true" tabindex="-1" on:click|stopPropagation={() => {}}>
        <h3>{confirmReq.title}</h3>
        <p class="muted modal-message">{confirmReq.message}</p>
        <div class="form-row modal-actions">
          <button on:click={() => answerConfirm(false)}>Batal</button>
          <button class="primary" on:click={() => answerConfirm(true)}>Ya</button>
        </div>
      </div>
    </div>
  {/if}

  {#if tab === 'dashboard'}
  <main class="layout">
    <section class="panel hero">
      {#if status.running}
        <div class="hero-top">
          <div>
            <div class="project-name">{status.project}</div>
            <div class="domain">{status.domain}</div>
          </div>
        </div>

        <div class="url-row">
          <code class="url">{status.url}</code>
          <code class="url">{status.local_url}</code>
        </div>

        <div class="metrics">
          <div class="metric">
            <div class="label">HTTP</div>
            <div class="value" class:ok={status.http_listening} class:warn={!status.http_listening}>
              {status.http_port} {status.http_listening ? 'listening' : 'free'}
            </div>
          </div>
          <div class="metric">
            <div class="label">HTTPS</div>
            <div class="value" class:ok={status.https_listening} class:warn={!status.https_listening}>
              {status.https_port} {status.https_listening ? 'listening' : 'free'}
            </div>
          </div>
          <div class="metric">
            <div class="label">Database</div>
            <div class="value">{status.database_enabled ? (status.db_name || '-') : 'off'}</div>
          </div>
          <div class="metric">
            <div class="label">DB user</div>
            <div class="value">{status.db_user || '-'} : {status.db_port || '-'}</div>
          </div>
        </div>


      {:else}
        <div class="hero-top">
          <div>
            <div class="project-name">Stack belum berjalan</div>
            <div class="domain">Pilih site lalu jalankan.</div>
          </div>
        </div>
        <div class="form-row">
          <select bind:value={selectedPath} on:change={() => loadComponents(selectedPath)}>
            <option value="">Pilih site...</option>
            {#each sites as site}
              <option value={site.path}>{site.name} - {site.domain}</option>
            {/each}
          </select>
          {#if activeSite}
            <button on:click={() => openFolder(activeSite.path)} disabled={!!busy}>Folder</button>
            <button on:click={() => openEditor(activeSite.path)} disabled={!!busy}>Editor</button>
          {/if}
        </div>
      {/if}

      <div class="toolbar">
        {#if status.running}
          <button class="danger tool" on:click={down} disabled={!!busy}>
            <svg viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" fill="currentColor" stroke="none"/></svg>
            Stop
          </button>
        {:else}
          <button class="primary tool" on:click={() => up()} disabled={!!busy}>
            <svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z" fill="currentColor" stroke="none"/></svg>
            Start All
          </button>
        {/if}
        <button class="tool" on:click={() => openURL(status.url)} disabled={!status.running || !!busy}>
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 3 2.5 15 0 18M12 3c-2.5 3-2.5 15 0 18"/></svg>
          Web
        </button>
        <button class="tool" on:click={openDatabase} disabled={!status.running || !dbEnabled || !pmaEnabled || !!busy}>
          <svg viewBox="0 0 24 24"><ellipse cx="12" cy="5" rx="7" ry="3"/><path d="M5 5v6c0 1.7 3.1 3 7 3s7-1.3 7-3V5M5 11v6c0 1.7 3.1 3 7 3s7-1.3 7-3v-6"/></svg>
          Database
        </button>
        <button class="tool" on:click={() => openTerminal(activePath())} disabled={!activePath() || !!busy}>
          <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 9l3 3-3 3M13 15h4"/></svg>
          Terminal
        </button>
        <button class="tool" on:click={() => openEditor(activePath())} disabled={!activePath() || !!busy}>
          <svg viewBox="0 0 24 24"><path d="M4 20h4L18 10l-4-4L4 16z"/><path d="M13.5 6.5l4 4"/></svg>
          Editor
        </button>
        <button class="tool" on:click={() => openFolder(activePath())} disabled={!activePath() || !!busy}>
          <svg viewBox="0 0 24 24"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
          Root
        </button>
      </div>

      <div class="service-list">
        <div class="service-row">
          <div class="service-main">
            <span class="service-dot {serviceState('nginx')}"></span>
            <strong>Nginx</strong>
            <span class="svc-version">{status.nginx_version}</span>
            <span class="muted">{status.running ? `HTTP ${status.http_port} / HTTPS ${status.https_port}` : 'auto 80/443'}</span>
          </div>
          <span class="pill {serviceState('nginx')}">{serviceState('nginx')}</span>
          <button type="button" class="switch" class:on={serviceState('nginx') === 'running'} locked role="switch" aria-checked={serviceState('nginx') === 'running'} aria-label="Nginx aktif" title="Wajib untuk site"></button>
        </div>

        <div class="service-row">
          <div class="service-main">
            <span class="service-dot {serviceState('php')}"></span>
            <strong>PHP</strong>
            <span class="svc-version">{status.php_version}</span>
            <span class="muted">FastCGI{status.running && status.php_port ? ' : ' + status.php_port : ''}</span>
          </div>
          <span class="pill {serviceState('php')}">{serviceState('php')}</span>
          <button type="button" class="switch" class:on={serviceState('php') === 'running'} locked role="switch" aria-checked={serviceState('php') === 'running'} aria-label="PHP aktif" title="Wajib untuk site"></button>
        </div>

        <div class="service-row">
          <div class="service-main">
            <span class="service-dot {dbEnabled ? serviceState('mysql') : 'stopped'}"></span>
            <strong>MySQL</strong>
            <span class="svc-version">{status.mysql_version}</span>
            <span class="muted">{status.running && dbEnabled ? `port ${status.db_port}` : 'auto 3306'}</span>
          </div>
          <span class="pill {dbEnabled ? serviceState('mysql') : 'stopped'}">{dbEnabled ? serviceState('mysql') : 'off'}</span>
          <button type="button" class="switch" class:on={dbEnabled && serviceState('mysql') === 'running'} on:click={toggleDatabase} disabled={!status.running || !componentsPath || !!busy} role="switch" aria-checked={dbEnabled && serviceState('mysql') === 'running'} aria-label="MySQL"></button>
        </div>

        <div class="service-row">
          <div class="service-main">
            <span class="service-dot {status.running && dbEnabled && pmaEnabled ? 'running' : 'stopped'}"></span>
            <strong>phpMyAdmin</strong>
            <span class="muted">Web UI</span>
          </div>
          <span class="pill {status.running && dbEnabled && pmaEnabled ? 'running' : 'stopped'}">{status.running && dbEnabled && pmaEnabled ? 'on' : 'off'}</span>
          <button type="button" class="switch" class:on={status.running && dbEnabled && pmaEnabled} on:click={togglePMA} disabled={!status.running || !componentsPath || !dbEnabled || !!busy} role="switch" aria-checked={status.running && dbEnabled && pmaEnabled} aria-label="phpMyAdmin"></button>
        </div>
      </div>
    </section>
  </main>

  {:else if tab === 'sites'}
  <main class="layout config-layout">
    <section class="panel config-panel">
      <h2>Site workspace</h2>
      <div class="stack">
        {#if sites.length === 0}
          <p class="muted">Belum ada site. Buat di bawah.</p>
        {/if}
        {#each sites as site}
          <div class="site" class:active={site.active || site.path === selectedPath}>
            <div class="site-head">
              <strong>{site.name}</strong>
              {#if site.active}<span class="pill running">active</span>{/if}
            </div>
            <div class="domain">{site.domain} - PHP {site.php}</div>
            <div class="path">{site.path}</div>
          </div>
        {/each}
      </div>

      <h2 style="margin-top:16px">Site baru</h2>
      <div class="form-row">
        <input placeholder="nama site" bind:value={newName} on:keydown={(e) => e.key === 'Enter' && createSite()} />
        <select bind:value={newPhp}>
          <option value="8.1">PHP 8.1</option>
          <option value="8.2">PHP 8.2</option>
          <option value="8.3">PHP 8.3</option>
          <option value="8.4">PHP 8.4</option>
          <option value="8.5">PHP 8.5</option>
        </select>
        <label class="check"><input type="checkbox" bind:checked={newWithDB} /> Database</label>
        <button on:click={createSite} disabled={!!busy}>Buat</button>
      </div>
    </section>
  </main>
  {:else if tab === 'logs'}
  <main class="layout config-layout">
    <section class="panel config-panel">
      <h2>Log</h2>
      <div class="form-row" style="margin-bottom:10px">
        <select bind:value={logService} on:change={loadLogs}>
          <option value="nginx">nginx</option>
          <option value="php">php</option>
          <option value="mysql">mysql</option>
          <option value="app">app</option>
        </select>
        <button on:click={loadLogs} disabled={!!busy}>Muat ulang</button>
        <button on:click={() => (logPaused = !logPaused)}>{logPaused ? 'Follow' : 'Pause'}</button>
      </div>
      <pre class="logs">{logs || 'Belum ada log.'}</pre>
    </section>
  </main>
  {:else if tab === 'config'}
  <main class="layout config-layout">
    <section class="panel config-panel">
      <h2>Konfigurasi project</h2>
      <div class="form-row">
        <select bind:value={cfgPath} on:change={() => loadConfig(cfgPath)}>
          <option value="">Pilih site...</option>
          {#each sites as site}
            <option value={site.path}>{site.name} - {site.domain}</option>
          {/each}
          {#if cfgPath && !sites.some((s) => s.path === cfgPath)}
            <option value={cfgPath}>{cfgPath}</option>
          {/if}
        </select>
        <button on:click={() => loadConfig(cfgPath)} disabled={!cfgPath || !!busy}>Muat</button>
      </div>

      {#if cfgPath}
        <div class="config-grid">
          <label>Virtual host
            <input bind:value={cfgDomain} placeholder="app.local" />
          </label>
          <label>Alias
            <input bind:value={cfgAliases} placeholder="www.app.local, api.app.local" />
          </label>
          <label>Document root
            <input bind:value={cfgRoot} placeholder="public" />
          </label>
          <label>Nama database
            <input bind:value={cfgDBName} placeholder="coba" />
          </label>
        </div>
        <label class="check">
          <input type="checkbox" bind:checked={cfgHTTPSEnabled} />
          HTTPS
        </label>
        <p class="muted">
          Port kosong = otomatis. Default: HTTP 80/8080, HTTPS 443/8443, MySQL 3306, PHP acak.
        </p>

        <h2>Service dan port</h2>
        <div class="service-config">
          <div class="service-config-row">
            <label class="check">
              <input type="checkbox" checked disabled />
              Nginx
            </label>
            {#if cfgOptions.nginx.length > 1}
              <select bind:value={cfgVersions.nginx}>
                {#each cfgOptions.nginx as v}
                  <option value={v}>{v}</option>
                {/each}
              </select>
            {:else}
              <span class="svc-version">{cfgOptions.nginx[0] || cfgVersions.nginx || '-'}</span>
            {/if}
            <div class="ports">
              <input type="text" inputmode="numeric" placeholder="80 (auto)" bind:value={cfgHTTP} on:change={checkPorts} on:blur={checkPorts} />
              <input type="text" inputmode="numeric" placeholder="443 (auto)" bind:value={cfgHTTPS} on:change={checkPorts} on:blur={checkPorts} />
            </div>
            <span class="port-note">{portHint(cfgHTTP, 'auto 80/8080')}<br />{portHint(cfgHTTPS, 'auto 443/8443')}</span>
          </div>

          <div class="service-config-row">
            <label class="check">
              <input type="checkbox" checked disabled />
              PHP
            </label>
            <select bind:value={cfgVersions.php}>
              {#each cfgOptions.php as v}
                <option value={v}>{v}</option>
              {/each}
            </select>
            <div class="ports">
              <input type="text" inputmode="numeric" placeholder="auto" bind:value={cfgPHPPort} on:change={checkPorts} on:blur={checkPorts} />
            </div>
            <span class="port-note">{portHint(cfgPHPPort, 'auto')}<br />FastCGI</span>
          </div>

          <div class="service-config-row">
            <label class="check">
              <input type="checkbox" bind:checked={cfgDB} />
              MySQL
            </label>
            <select bind:value={cfgVersions.mysql} disabled={!cfgDB}>
              {#each cfgOptions.mysql as v}
                <option value={v}>{v}</option>
              {/each}
            </select>
            <div class="ports">
              <input type="text" inputmode="numeric" placeholder="3306 (auto)" bind:value={cfgMySQLPort} on:change={checkPorts} on:blur={checkPorts} disabled={!cfgDB} />
            </div>
            <span class="port-note">{portHint(cfgMySQLPort, 'auto 3306')}<br />TCP</span>
          </div>

          <div class="service-config-row">
            <label class="check">
              <input type="checkbox" bind:checked={cfgPMA} disabled={!cfgDB} />
              phpMyAdmin
            </label>
            <span class="muted">Mengikuti HTTP/HTTPS</span>
          </div>

          <div class="service-config-row">
            <label class="check">
              <input
                type="checkbox"
                checked={!!cfgVersions.node}
                on:change={(e) => toggleNode(e.currentTarget.checked)}
              />
              Node.js
            </label>
            <select bind:value={cfgVersions.node} disabled={!cfgVersions.node}>
              <option value="">Tidak dipakai</option>
              {#each cfgOptions.node as v}
                <option value={v}>{v}</option>
              {/each}
            </select>
            <span class="muted">Runtime untuk run/shell</span>
          </div>
        </div>

        <h2>Runtime terpasang</h2>
        {#if runtimes.length > 0}
          <table>
            <thead><tr><th>Nama</th><th>Versi</th><th>Ukuran</th><th>Status</th><th>Path</th><th>Aksi</th></tr></thead>
            <tbody>
              {#each runtimes as r}
                <tr>
                  <td>{r.name}</td>
                  <td>{r.version}</td>
                  <td>{formatBytes(r.bytes)}</td>
                  <td>{r.in_use ? 'dipakai' : 'tidak dipakai'}</td>
                  <td class="muted">{r.path}</td>
                  <td>
                    <button
                      on:click={() => removeRuntime(r)}
                      disabled={!!busy || r.in_use}
                      title={r.in_use ? 'Dipakai: ' + (r.used_by || []).join(', ') : ''}
                    >Hapus</button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
          <div class="form-row" style="margin-top:8px">
            <button on:click={cleanupUnused} disabled={!!busy || updateBusy}>Hapus versi lama</button>
            <button on:click={cleanupCache} disabled={!!busy || updateBusy}>Bersihkan cache ({formatBytes(runtimeCache.bytes)})</button>
          </div>
        {:else}
          <p class="muted">Tidak ada runtime terpasang.</p>
        {/if}

        <div class="form-row" style="margin-top:12px">
          <button class="primary" on:click={saveConfig} disabled={!!busy || !cfgDirty}>Simpan</button>
          {#if cfgDirty}<span class="muted">Perubahan belum disimpan</span>{/if}
        </div>
        <p class="muted">Port 0 = otomatis. Menyimpan saat stack berjalan akan restart stack.</p>
      {/if}
    </section>
  </main>
  {:else if tab === 'update'}
  <main class="layout config-layout">
    <section class="panel config-panel">
      <h2>Update DevLok</h2>
      <div class="form-row">
        <span class="muted">Versi terpasang: <strong>{version}</strong></span>
        <button on:click={loadUpdates} disabled={updateBusy}>{updateBusy ? 'Memeriksa…' : 'Periksa update'}</button>
      </div>

      <h2>Pembaruan otomatis</h2>
      <div class="prefs-box">
        <label class="check">
          <input type="checkbox" bind:checked={updatePrefs.auto_update_app} />
          Update aplikasi otomatis
        </label>
        <label class="check">
          <input type="checkbox" bind:checked={updatePrefs.auto_update_runtime} />
          Update runtime/service otomatis
        </label>
        <label class="check">
          <input
            type="checkbox"
            bind:checked={updatePrefs.auto_update_runtime_series}
            disabled={!updatePrefs.auto_update_runtime}
          />
          Ikut upgrade seri baru (mis. PHP 8.4 ke 8.5)
        </label>
        <div class="form-row">
          <label class="muted">
            Interval
            <select bind:value={updatePrefs.interval_minutes}>
              <option value={15}>15 menit</option>
              <option value={60}>1 jam</option>
              <option value={360}>6 jam</option>
              <option value={1440}>24 jam</option>
            </select>
          </label>
          <button on:click={saveUpdatePrefs} disabled={prefsBusy}>Simpan preferensi</button>
          {#if prefsNotice}<span class="muted">{prefsNotice}</span>{/if}
        </div>
        <p class="muted">
          Update aplikasi otomatis menutup DevLok dan menghentikan stack sebelum installer dijalankan.
          Runtime hanya menerima patch seri terpasang kecuali opsi seri baru diaktifkan.
        </p>
      </div>

      {#if updateNotice}
        <div class="notice-box">{updateNotice}</div>
      {/if}

      {#if updateProgress}
        <div class="progress-box" class:failed={updateProgress.phase === 'error'}>
          <div class="progress-head">
            <span>{progressLabel(updateProgress)}</span>
            {#if updateProgress.phase === 'download' && updateProgress.total > 0}
              <strong>{updateProgress.percent}%</strong>
            {:else if updateProgress.phase === 'done'}
              <strong>100%</strong>
            {/if}
          </div>
          <div
            class="progress-track"
            class:indeterminate={updateProgress.phase !== 'download' && updateProgress.phase !== 'done' && updateProgress.phase !== 'error'}
          >
            <div class="progress-fill" style="width:{updateProgress.phase === 'done' ? 100 : updateProgress.percent}%"></div>
          </div>
          {#if updateProgress.phase === 'download' && updateProgress.total > 0}
            <div class="muted">{formatBytes(updateProgress.downloaded)} / {formatBytes(updateProgress.total)}</div>
          {:else if updateProgress.message}
            <div class="muted">{updateProgress.message}</div>
          {/if}
          <div class="form-row" style="margin-top:8px">
            {#if updateProgress.phase === 'error' && lastUpdateRetry}
              <button on:click={() => lastUpdateRetry?.()} disabled={updateBusy}>Coba lagi</button>
            {/if}
            {#if updateBusy && updateProgress.phase !== 'done' && updateProgress.phase !== 'error' && updateProgress.phase !== 'installer'}
              <button class="danger" on:click={cancelUpdate}>Batal</button>
            {/if}
          </div>
        </div>
      {/if}

      {#if updateInfo}
        {#if updateInfo.remote_error}
          <p class="muted">Pemeriksaan aplikasi: {updateInfo.remote_error}</p>
        {/if}

        {#if updateInfo.app_update}
          <div class="site">
            <div class="site-head">
              <strong>DevLok {updateInfo.app_update.latest}</strong>
              {#if updateInfo.app_update.severity}
                <span class="pill running">{updateInfo.app_update.severity}</span>
              {/if}
            </div>
            <div class="muted">Terpasang {updateInfo.app_update.current}</div>
            {#if updateInfo.app_update.notes}
              <p class="muted" style="white-space:pre-wrap">{updateInfo.app_update.notes}</p>
            {/if}
            <div class="form-row">
              <button on:click={downloadAppUpdate} disabled={updateBusy}>Unduh installer</button>
              <button class="primary" on:click={applyAppUpdate} disabled={updateBusy}>Unduh & jalankan</button>
            </div>
          </div>
        {:else}
          <p class="muted">Aplikasi sudah memakai versi terbaru.</p>
        {/if}

        <h2>Update runtime</h2>
        {#if patchUpdates.length > 0}
          <div class="form-row" style="margin-bottom:10px">
            <button class="primary" on:click={updateAllPatches} disabled={updateBusy}>Update semua patch ({patchUpdates.length})</button>
          </div>
        {/if}
        {#if !updateInfo.runtimes || updateInfo.runtimes.length === 0}
          {#if status.platform === 'darwin'}
            <p class="muted">Runtime dikelola Homebrew. Perbarui dengan brew upgrade nginx php mysql.</p>
          {:else}
            <p class="muted">Semua runtime terpasang sudah versi terbaru.</p>
          {/if}
        {:else}
          <div class="stack">
            {#each updateInfo.runtimes as u}
              <div class="site">
                <div class="site-head">
                  <strong>{u.name}</strong>
                  {#if u.kind === 'new'}
                    <span class="pill">seri baru {u.series}</span>
                  {:else}
                    <span class="pill running">patch {u.series}</span>
                  {/if}
                </div>
                <div class="muted">{u.installed} &rarr; {u.latest}</div>
                <div class="form-row">
                  <button class="primary" on:click={() => updateRuntime(u)} disabled={updateBusy}>Update ke {u.latest}</button>
                  <button on:click={() => copyRuntimeCommand(u)}>Salin perintah</button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      {:else}
        <p class="muted">Klik Periksa update untuk melihat versi terbaru.</p>
      {/if}

      <h2>Rollback</h2>
      <p class="muted">Pulihkan binary dari backup installer sebelumnya. Hanya tersedia setelah upgrade melalui installer.</p>
      <button on:click={rollbackAppUpdate} disabled={updateBusy}>Rollback app update</button>
    </section>
  </main>
  {:else}
  <main class="layout config-layout">
    <section class="panel config-panel">
      <h2>Import Laragon</h2>
      <div class="form-row">
        <input bind:value={importPath} placeholder="C:\laragon" />
        <button on:click={scanLaragon} disabled={importBusy}>Scan</button>
        <label class="check">
          <input type="checkbox" bind:checked={importAdopt} />
          Daftarkan ke GUI
        </label>
      </div>
      <p class="muted">Scan tidak mengubah file. Import menulis .devlok.yaml di folder project.</p>

      {#if importSites.length > 0}
        <table>
          <thead>
            <tr><th></th><th>Nama</th><th>Domain</th><th>Root</th><th>Sumber</th><th>Status</th></tr>
          </thead>
          <tbody>
            {#each importSites as site}
              <tr>
                <td><input type="checkbox" bind:checked={importSelected[site.name]} /></td>
                <td>{site.name}</td>
                <td>{site.domain}</td>
                <td>{site.root}</td>
                <td class="muted">{site.source}</td>
                <td>{site.action || 'ready'}</td>
              </tr>
            {/each}
          </tbody>
        </table>
        <div class="form-row" style="margin-top:10px">
          <button on:click={toggleImportAll} disabled={importBusy}>Pilih semua</button>
          <button class="primary" on:click={runImport} disabled={importBusy}>Import terpilih</button>
        </div>
      {:else}
        <p class="muted">Belum ada hasil scan.</p>
      {/if}

      {#if importResults.length > 0}
        <h2>Hasil import</h2>
        <ul class="import-results">
          {#each importResults as r}
            <li>{r.site.name}: {r.action}{#if r.message} - {r.message}{/if}</li>
          {/each}
        </ul>
      {/if}
    </section>
  </main>
  {/if}

  <footer class="footer">
    <span>DevLok {version}</span>
    <span class="powered">AI-powered by DeepSeek-v4-Flash</span>
    <span>{busy || 'siap'}</span>
  </footer>
</div>
