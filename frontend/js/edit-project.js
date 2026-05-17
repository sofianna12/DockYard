requireAuth()

const params    = new URLSearchParams(window.location.search)
const projectId = params.get('id')
if (!projectId) window.location.href = 'dashboard.html'

function getProjectType() {
  return document.querySelector('input[name="project-type"]:checked')?.value || 'web'
}

function onProjectTypeChange() {
  const type = getProjectType()
  const show = (type === 'web' || type === 'web_terminal') ? '' : 'none'
  document.getElementById('port-group').style.display   = show
  document.getElementById('scheme-group').style.display = show
}

function toggleAccordion(id) {
  document.getElementById('acc-' + id).classList.toggle('open')
}

function loadEnvFile(input) {
  const file = input.files[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    document.getElementById('env-vars').value = e.target.result.trim()
  }
  reader.readAsText(file)
}

function goBack() {
  window.location.href = `project-details.html?id=${projectId}`
}

async function loadProject() {
  try {
    const p = await api.getProject(projectId)

    document.getElementById('title').value        = p.title || ''
    document.getElementById('description').value  = p.description || ''
    document.getElementById('repository').value   = p.repository || ''
    document.getElementById('docker-image').value = p.docker_image || ''
    document.getElementById('env-vars').value     = p.env_vars || ''
    document.getElementById('mounts').value       = p.mounts || ''
    document.getElementById('container-port').value = p.container_port || ''
    document.getElementById('registry-user').value  = p.registry_user || ''
    document.getElementById('scheme').value         = p.scheme || 'http'

    let type = 'web'
    if (p.terminal_mode)  type = 'terminal'
    else if (p.web_terminal) type = 'web_terminal'
    const radio = document.querySelector(`input[name="project-type"][value="${type}"]`)
    if (radio) radio.checked = true
    onProjectTypeChange()

    const autoStopSelect = document.getElementById('auto-stop')
    const autoStopVal    = String(p.auto_stop_min)
    let found = false
    for (const opt of autoStopSelect.options) {
      if (opt.value === autoStopVal) { opt.selected = true; found = true; break }
    }
    if (!found) autoStopSelect.value = '60'

    if (p.registry_user) document.getElementById('acc-auth').classList.add('open')
    if (p.env_vars)      document.getElementById('acc-env').classList.add('open')
    if (p.mounts)        document.getElementById('acc-mounts').classList.add('open')

    loadFiles()
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
  }
}

async function submitForm() {
  const btn     = document.getElementById('submit-btn')
  const alertEl = document.getElementById('alert')
  const type    = getProjectType()

  const data = {
    title:             document.getElementById('title').value.trim(),
    description:       document.getElementById('description').value.trim(),
    repository:        document.getElementById('repository').value.trim(),
    docker_image:      document.getElementById('docker-image').value.trim(),
    registry_user:     document.getElementById('registry-user').value.trim(),
    registry_password: document.getElementById('registry-password').value,
    env_vars:          document.getElementById('env-vars').value.trim(),
    mounts:            document.getElementById('mounts').value.trim(),
    terminal_mode:     type === 'terminal',
    web_terminal:      type === 'web_terminal',
    container_port:    parseInt(document.getElementById('container-port').value) || 0,
    auto_stop_min:     parseInt(document.getElementById('auto-stop').value),
    scheme:            document.getElementById('scheme').value,
  }

  alertEl.style.display = 'none'

  if (!data.title || !data.docker_image) {
    alertEl.textContent   = 'Title and Docker Image are required'
    alertEl.style.display = 'block'
    return
  }

  btn.disabled  = true
  btn.innerHTML = '<span class="spinner"></span> Saving...'

  try {
    await api.updateProject(projectId, data)
    window.location.href = `project-details.html?id=${projectId}`
  } catch (err) {
    alertEl.textContent   = err.message
    alertEl.style.display = 'block'
    btn.disabled          = false
    btn.textContent       = 'Save Changes'
  }
}

async function loadFiles() {
  const list = document.getElementById('files-list')
  try {
    const files = await api.listFiles(projectId)
    if (!files || files.length === 0) {
      list.innerHTML = '<p style="color:#64748b;font-size:0.82rem">No files uploaded yet.</p>'
      return
    }
    document.getElementById('acc-files').classList.add('open')
    list.innerHTML = files.map(f => `
      <div style="display:flex;align-items:center;justify-content:space-between;padding:0.45rem 0;border-bottom:1px solid #2d3748">
        <div>
          <code style="font-size:0.82rem;color:#7dd3fc">${escapeHtml(f.filename)}</code>
          <span style="color:#64748b;font-size:0.75rem;margin-left:0.5rem">(${formatBytes(f.size_bytes)})</span>
        </div>
        <button class="btn btn-danger btn-sm" onclick="deleteFile('${escapeHtml(f.filename)}')">✕</button>
      </div>
    `).join('')
  } catch (err) {
    list.innerHTML = `<p style="color:#64748b;font-size:0.82rem">${escapeHtml(err.message)}</p>`
  }
}

async function uploadFile(input) {
  const file = input.files[0]
  if (!file) return
  const status  = document.getElementById('upload-status')
  const alertEl = document.getElementById('files-alert')
  alertEl.style.display = 'none'
  status.textContent    = 'Uploading...'
  input.disabled        = true
  const formData = new FormData()
  formData.append('file', file)
  try {
    await api.uploadFile(projectId, formData)
    status.textContent = ''
    input.value        = ''
    loadFiles()
  } catch (err) {
    alertEl.textContent   = err.message
    alertEl.style.display = 'block'
    status.textContent    = ''
  } finally {
    input.disabled = false
  }
}

async function deleteFile(filename) {
  if (!confirm(`Delete "${filename}"?`)) return
  const alertEl = document.getElementById('files-alert')
  alertEl.style.display = 'none'
  try {
    await api.deleteFile(projectId, filename)
    loadFiles()
  } catch (err) {
    alertEl.textContent   = err.message
    alertEl.style.display = 'block'
  }
}

function formatBytes(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function escapeHtml(str) {
  const div = document.createElement('div')
  div.appendChild(document.createTextNode(String(str)))
  return div.innerHTML
}

loadProject()
