<script lang="ts">
  import { onMount, onDestroy } from 'svelte'

  type Status = {
    running: boolean
    health: string
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

  const serviceRuntimeNames = new Set(['nginx', 'php', 'mysql', 'node', 'phpmyadmin'])
  $: extraRuntimes = runtimes.filter((r) => !serviceRuntimeNames.has(r.name))

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
    if (!current) return ''
    if (options.includes(current)) return current
    return options.find((v) => v.startsWith(current + '.')) || current
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
      const [c, nginx, php, mysql, node, rl] = await Promise.all([
        api().ProjectConfig(path),
        api().ServiceVersions('nginx'),
        api().ServiceVersions('php'),
        api().ServiceVersions('mysql'),
        api().ServiceVersions('node'),
        api().RuntimeList()
      ])
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
    try {
      const app = api()
      const [s, si] = await Promise.all([app.Status(), app.Sites()])
      status = s
      sites = si
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
      const [c, versions] = await Promise.all([api().Components(path), api().MySQLVersions()])
      dbEnabled = !!c.database
      pmaEnabled = !!c.phpmyadmin
      mysqlVersions = versions || []
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

  async function loadLogs() {
    try {
      logs = await api().Logs(logService, 250)
    } catch (e) {
      error = message(e)
    }
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

  function phpMyAdminURL() {
    if (!status.url) return ''
    try {
      const u = new URL(status.url)
      return `${u.protocol}//phpmyadmin.test${u.port ? ':' + u.port : ''}`
    } catch {
      return ''
    }
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
  $: pmaURL = phpMyAdminURL()

  onMount(() => {
    refresh(false)
    timer = window.setInterval(() => refresh(true), 5000)
  })

  onDestroy(() => {
    if (timer) window.clearInterval(timer)
  })
</script>

<div class="app">
  <header class="topbar">
    <div class="brand">DevLok <span>local dev</span></div>
    <nav class="tabs">
      <button class:active={tab === 'dashboard'} on:click={() => (tab = 'dashboard')}>Dashboard</button>
      <button class:active={tab === 'sites'} on:click={() => (tab = 'sites')}>Sites</button>
      <button class:active={tab === 'logs'} on:click={() => (tab = 'logs')}>Logs</button>
      <button class:active={tab === 'config'} on:click={openConfig}>Konfigurasi</button>
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
        <button class="tool" on:click={() => openURL(pmaURL)} disabled={!status.running || !dbEnabled || !pmaEnabled || !!busy}>
          <svg viewBox="0 0 24 24"><ellipse cx="12" cy="5" rx="7" ry="3"/><path d="M5 5v6c0 1.7 3.1 3 7 3s7-1.3 7-3V5M5 11v6c0 1.7 3.1 3 7 3s7-1.3 7-3v-6"/></svg>
          Database
        </button>
        <button class="tool" on:click={() => openTerminal(activePath())} disabled={!activePath() || !!busy}>
          <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 9l3 3-3 3M13 15h4"/></svg>
          Terminal
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

        <h2>Tool tambahan terpasang</h2>
        {#if extraRuntimes.length > 0}
          <table>
            <thead><tr><th>Nama</th><th>Versi</th><th>Path</th></tr></thead>
            <tbody>
              {#each extraRuntimes as r}
                <tr><td>{r.name}</td><td>{r.version}</td><td class="muted">{r.path}</td></tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="muted">Tidak ada tool tambahan terpasang.</p>
        {/if}

        <div class="form-row" style="margin-top:12px">
          <button class="primary" on:click={saveConfig} disabled={!!busy || !cfgDirty}>Simpan</button>
          {#if cfgDirty}<span class="muted">Perubahan belum disimpan</span>{/if}
        </div>
        <p class="muted">Port 0 = otomatis. Menyimpan saat stack berjalan akan restart stack.</p>
      {/if}
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
    <span class="powered">Powered by DeepSeek-v4-Flash</span>
    <span>{busy || 'siap'}</span>
  </footer>
</div>
