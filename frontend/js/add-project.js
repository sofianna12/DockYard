requireAuth()

const fileQueue = []

function queueFile(input) {
  const file = input.files[0]
  if (!file) return
  input.value = ''
  const idx = fileQueue.findIndex(f => f.name === file.name)
  if (idx !== -1) fileQueue.splice(idx, 1, file)
  else fileQueue.push(file)
  renderFileQueue()
}

function removeQueuedFile(name) {
  const idx = fileQueue.findIndex(f => f.name === name)
  if (idx !== -1) fileQueue.splice(idx, 1)
  renderFileQueue()
}

function renderFileQueue() {
  const list = document.getElementById('files-queue')
  if (fileQueue.length === 0) {
    list.innerHTML = '<p style="color:#64748b;font-size:0.82rem">No files queued.</p>'
    return
  }
  list.innerHTML = fileQueue.map(f => `
    <div style="display:flex;align-items:center;justify-content:space-between;padding:0.45rem 0;border-bottom:1px solid #2d3748">
      <div>
        <code style="font-size:0.82rem;color:#7dd3fc">${escapeHtml(f.name)}</code>
        <span style="color:#64748b;font-size:0.75rem;margin-left:0.5rem">(${formatBytes(f.size)})</span>
      </div>
      <button class="btn btn-danger btn-sm" onclick="removeQueuedFile('${escapeHtml(f.name)}')">✕</button>
    </div>
  `).join('')
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
    const project = await api.createProject(data)

    if (fileQueue.length > 0) {
      btn.innerHTML = '<span class="spinner"></span> Uploading files...'
      for (const file of fileQueue) {
        const formData = new FormData()
        formData.append('file', file)
        await api.uploadFile(project.id, formData)
      }
    }

    window.location.href = 'dashboard.html'
  } catch (err) {
    alertEl.textContent   = err.message
    alertEl.style.display = 'block'
    btn.disabled          = false
    btn.textContent       = 'Save Project'
  }
}

onProjectTypeChange()
