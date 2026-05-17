requireAuth()

const params    = new URLSearchParams(window.location.search)
const projectId = params.get('id')
if (!projectId) window.location.href = 'dashboard.html'

function onTerminalToggle() {
  const checked = document.getElementById('terminal-mode').checked
  document.getElementById('port-group').style.display = checked ? 'none' : ''
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

    document.getElementById('title').value         = p.title || ''
    document.getElementById('description').value   = p.description || ''
    document.getElementById('repository').value    = p.repository || ''
    document.getElementById('docker-image').value  = p.docker_image || ''
    document.getElementById('env-vars').value      = p.env_vars || ''
    document.getElementById('mounts').value        = p.mounts || ''
    document.getElementById('terminal-mode').checked  = !!p.terminal_mode
    document.getElementById('port-group').style.display = p.terminal_mode ? 'none' : ''
    document.getElementById('container-port').value = p.container_port || ''

    const autoStopSelect = document.getElementById('auto-stop')
    const autoStopVal    = String(p.auto_stop_min)
    let found = false
    for (const opt of autoStopSelect.options) {
      if (opt.value === autoStopVal) { opt.selected = true; found = true; break }
    }
    if (!found) autoStopSelect.value = '60'
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
  }
}

async function submitForm() {
  const btn     = document.getElementById('submit-btn')
  const alertEl = document.getElementById('alert')

  const data = {
    title:          document.getElementById('title').value.trim(),
    description:    document.getElementById('description').value.trim(),
    repository:     document.getElementById('repository').value.trim(),
    docker_image:   document.getElementById('docker-image').value.trim(),
    env_vars:       document.getElementById('env-vars').value.trim(),
    mounts:         document.getElementById('mounts').value.trim(),
    terminal_mode:  document.getElementById('terminal-mode').checked,
    container_port: parseInt(document.getElementById('container-port').value) || 0,
    auto_stop_min:  parseInt(document.getElementById('auto-stop').value),
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

loadProject()
