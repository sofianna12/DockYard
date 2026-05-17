requireAuth()

const params    = new URLSearchParams(window.location.search)
const projectId = params.get('id')
if (!projectId) window.location.href = 'dashboard.html'

let project           = null
let pollInterval      = null
let countdownInterval = null
let logsInterval      = null
let term              = null
let termWs            = null
let fitAddon          = null

function editProject() {
  window.location.href = `edit-project.html?id=${projectId}`
}

async function deleteProject() {
  if (!confirm('Delete this project? This cannot be undone.')) return
  try {
    await api.deleteProject(projectId)
    window.location.href = 'dashboard.html'
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
  }
}

function renderProject(p) {
  project = p
  document.getElementById('title').textContent        = p.title
  document.getElementById('description').textContent  = p.description || '—'
  document.getElementById('docker-image').textContent = p.docker_image
  document.getElementById('auto-stop').textContent    =
    p.auto_stop_min > 0 ? p.auto_stop_min + ' minutes' : 'Never'
  document.getElementById('created-at').textContent   =
    new Date(p.created_at).toLocaleString()

  const repoEl = document.getElementById('repository')
  if (p.repository) {
    repoEl.innerHTML = `<a href="${p.repository}" target="_blank" class="open-app-link">${p.repository}</a>`
  } else {
    repoEl.textContent = '—'
  }

  const url = (p.port && p.scheme) ? `${p.scheme}://localhost:${p.port}` : null
  updateStatus(p.status, url, p.last_accessed_at)
}

function updateStatus(status, url, lastAccessed) {
  const badge     = document.getElementById('status-badge')
  const launchBtn = document.getElementById('launch-btn')
  const stopBtn   = document.getElementById('stop-btn')
  const openApp   = document.getElementById('open-app')
  const openLink  = document.getElementById('open-app-link')

  badge.className   = `badge badge-${status}`
  badge.textContent = status

  if (status === 'running') {
    launchBtn.style.display = 'none'
    stopBtn.style.display   = 'inline-block'
    stopBtn.disabled        = false
    stopBtn.textContent     = '■ Stop'
    if (project && project.terminal_mode) {
      document.getElementById('terminal-section').style.display = 'flex'
      document.getElementById('details-container').classList.add('wide')
      openApp.style.display = 'none'
      openTerminal()
    } else if (project && project.web_terminal) {
      document.getElementById('terminal-section').style.display = 'flex'
      document.getElementById('terminal-section').classList.add('split-logs')
      document.getElementById('details-container').classList.add('wide')
      document.getElementById('terminal-logs-panel').style.display = 'flex'
      document.getElementById('bottom-logs-section').style.display = 'none'
      document.querySelector('.details-bottom-card').style.display = 'none'
      if (url) {
        openApp.style.display = 'block'
        openLink.href         = url
        openLink.textContent  = `🔗 Open App (${url})`
      }
      openTerminal()
    } else if (url) {
      document.getElementById('terminal-topbar').style.display    = 'none'
      document.getElementById('terminal-container').style.display  = 'none'
      document.getElementById('terminal-section').style.display    = 'flex'
      document.getElementById('terminal-section').classList.add('split-logs')
      document.getElementById('details-container').classList.add('wide')
      document.getElementById('terminal-logs-panel').style.display = 'flex'
      document.getElementById('bottom-logs-section').style.display = 'none'
      document.querySelector('.details-bottom-card').style.display = 'none'
      openApp.style.display = 'block'
      openLink.href         = url
      openLink.textContent  = `🔗 Open App (${url})`
    }
    startCountdown(lastAccessed, project ? project.auto_stop_min : 60)
    startPolling()
    startLogsAutoRefresh()
  } else if (status === 'pulling') {
    launchBtn.style.display = 'inline-block'
    launchBtn.disabled      = true
    launchBtn.innerHTML     = '<span class="spinner"></span> Pulling...'
    stopBtn.style.display   = 'none'
    openApp.style.display   = 'none'
    stopCountdown()
    startPolling()
    startLogsAutoRefresh()
  } else {
    launchBtn.style.display = 'inline-block'
    launchBtn.disabled      = false
    launchBtn.textContent   = '▶ Run Docker'
    stopBtn.style.display   = 'none'
    openApp.style.display   = 'none'
    document.getElementById('terminal-topbar').style.display    = ''
    document.getElementById('terminal-container').style.display  = ''
    document.getElementById('terminal-section').style.display    = 'none'
    document.getElementById('terminal-section').classList.remove('split-logs')
    document.getElementById('details-container').classList.remove('wide')
    document.getElementById('terminal-logs-panel').style.display = 'none'
    document.getElementById('bottom-logs-section').style.display = ''
    document.querySelector('.details-bottom-card').style.display = ''
    closeTerminal()
    stopPolling()
    stopCountdown()
    stopLogsAutoRefresh()
    refreshLogs()
  }
}

function startPolling() {
  if (pollInterval) return
  pollInterval = setInterval(async () => {
    try {
      const res = await api.getStatus(projectId)
      if (res) updateStatus(res.status, res.url || null, project ? project.last_accessed_at : null)
    } catch (_) {}
  }, 5000)
}

function stopPolling() {
  clearInterval(pollInterval)
  pollInterval = null
}

function startCountdown(lastAccessed, autoStopMin) {
  stopCountdown()
  if (!lastAccessed || autoStopMin <= 0) return

  const el = document.getElementById('countdown')
  el.style.display = 'block'

  function tick() {
    const expires = new Date(new Date(lastAccessed).getTime() + autoStopMin * 60000)
    const diff    = expires - Date.now()
    if (diff <= 0) {
      el.textContent = 'Auto-stopping...'
      return
    }
    const m = Math.floor(diff / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    el.textContent = `⏱ Auto-stop in ${m}m ${s}s`
  }

  tick()
  countdownInterval = setInterval(tick, 1000)
}

function stopCountdown() {
  clearInterval(countdownInterval)
  countdownInterval = null
  const el = document.getElementById('countdown')
  if (el) el.style.display = 'none'
}

async function launchContainer() {
  const btn     = document.getElementById('launch-btn')
  const alertEl = document.getElementById('alert')
  alertEl.style.display = 'none'

  btn.disabled  = true
  btn.innerHTML = '<span class="spinner"></span> Launching...'

  try {
    await api.launchContainer(projectId)
    // Launch is async — show pulling state and let polling detect when running
    updateStatus('pulling', null, null)
  } catch (err) {
    alertEl.textContent   = err.message
    alertEl.style.display = 'block'
    btn.disabled          = false
    btn.textContent       = '▶ Run Docker'
  }
}

async function stopContainer() {
  const btn = document.getElementById('stop-btn')
  btn.disabled  = true
  btn.textContent = 'Stopping...'
  try {
    await api.stopContainer(projectId)
    updateStatus('stopped', null, null)
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
    btn.disabled    = false
    btn.textContent = '■ Stop'
  }
}

async function refreshLogs() {
  const el     = document.getElementById('logs-output')
  const elSide = document.getElementById('logs-output-side')
  try {
    const res  = await api.getLogs(projectId, 200)
    const text = res && res.logs ? res.logs.trim() : ''
    el.textContent = text || 'No logs yet.'
    el.scrollTop   = el.scrollHeight
    if (elSide) {
      elSide.textContent = text || 'No logs yet.'
      elSide.scrollTop   = elSide.scrollHeight
    }
  } catch (err) {
    el.textContent = 'Error fetching logs: ' + err.message
    if (elSide) elSide.textContent = 'Error fetching logs: ' + err.message
  }
}

function startLogsAutoRefresh() {
  if (logsInterval) return
  refreshLogs()
  logsInterval = setInterval(() => {
    if (document.getElementById('logs-auto-refresh').checked) refreshLogs()
  }, 5000)
}

function stopLogsAutoRefresh() {
  clearInterval(logsInterval)
  logsInterval = null
}

function formatFileSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function renderFiles(files) {
  const el = document.getElementById('files-list')
  if (!files || files.length === 0) {
    el.innerHTML = '<span class="files-empty">No files uploaded.</span>'
    return
  }
  el.innerHTML = files.map(f => `
    <div class="file-item">
      <div class="file-item-info">
        <div class="file-item-name">${escapeHtml(f.filename)}</div>
        <div class="file-item-path">${escapeHtml(f.container_path)}</div>
      </div>
      <span class="file-item-size">${formatFileSize(f.size_bytes)}</span>
      <button class="file-delete-btn" onclick="handleFileDelete(${JSON.stringify(f.filename)})" title="Delete">✕</button>
    </div>
  `).join('')
}

async function loadFiles() {
  try {
    const files = await api.listFiles(projectId)
    renderFiles(files)
  } catch (_) {
    renderFiles([])
  }
}

async function handleFileUpload(event) {
  const file = event.target.files[0]
  if (!file) return
  const formData = new FormData()
  formData.append('file', file)
  const btn = document.getElementById('upload-btn')
  btn.disabled    = true
  btn.textContent = 'Uploading...'
  try {
    await api.uploadFile(projectId, formData)
    await loadFiles()
  } catch (err) {
    alert('Upload failed: ' + err.message)
  } finally {
    btn.disabled    = false
    btn.textContent = '+ Upload'
    event.target.value = ''
  }
}

async function handleFileDelete(filename) {
  if (!confirm(`Delete "${filename}"?`)) return
  try {
    await api.deleteFile(projectId, filename)
    await loadFiles()
  } catch (err) {
    alert('Delete failed: ' + err.message)
  }
}

async function init() {
  try {
    const p = await api.getProject(projectId)
    renderProject(p)
    loadFiles()
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
    document.getElementById('title').textContent   = 'Error'
  }
}

let termLastAttempt = 0

function openTerminal() {
  if (termWs && (termWs.readyState === WebSocket.OPEN || termWs.readyState === WebSocket.CONNECTING)) return
  const now = Date.now()
  if (now - termLastAttempt < 5000) return
  termLastAttempt = now

  if (!term) {
    term = new Terminal({ cursorBlink: true, fontSize: 14, theme: { background: '#0d1117', foreground: '#e6edf3' } })
    fitAddon = new FitAddon.FitAddon()
    term.loadAddon(fitAddon)
    term.open(document.getElementById('terminal-container'))
    fitAddon.fit()

    new ResizeObserver(() => {
      if (fitAddon) {
        fitAddon.fit()
        if (termWs && termWs.readyState === WebSocket.OPEN) {
          termWs.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }))
        }
      }
    }).observe(document.getElementById('terminal-container'))
  }

  const token = localStorage.getItem('token')
  const wsProto = location.protocol === 'https:' ? 'wss' : 'ws'
  termWs = new WebSocket(`${wsProto}://${location.hostname}:8081/ws/projects/${projectId}/terminal?token=${token}`)
  termWs.binaryType = 'arraybuffer'

  termWs.onopen = () => {
    const { rows, cols } = term
    termWs.send(JSON.stringify({ type: 'resize', rows, cols }))
  }

  termWs.onmessage = (e) => {
    const data = e.data instanceof ArrayBuffer ? new Uint8Array(e.data) : e.data
    term.write(data)
  }

  termWs.onclose = () => term && term.write('\r\n\x1b[90m[disconnected]\x1b[0m\r\n')

  term.onData(data => {
    if (termWs && termWs.readyState === WebSocket.OPEN) {
      termWs.send(new TextEncoder().encode(data))
    }
  })
}

function closeTerminal() {
  if (termWs) { termWs.close(); termWs = null }
}

function reconnectTerminal() {
  closeTerminal()
  openTerminal()
}

function escapeHtml(str) {
  const div = document.createElement('div')
  div.appendChild(document.createTextNode(String(str)))
  return div.innerHTML
}

window.addEventListener('beforeunload', () => {
  stopPolling()
  stopCountdown()
  stopLogsAutoRefresh()
  closeTerminal()
})

init()
