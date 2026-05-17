requireAuth()

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

async function submitForm() {
  const btn   = document.getElementById('submit-btn')
  const alertEl = document.getElementById('alert')

  const data = {
    title:             document.getElementById('title').value.trim(),
    description:       document.getElementById('description').value.trim(),
    repository:        document.getElementById('repository').value.trim(),
    docker_image:      document.getElementById('docker-image').value.trim(),
    registry_user:     document.getElementById('registry-user').value.trim(),
    registry_password: document.getElementById('registry-password').value,
    env_vars:          document.getElementById('env-vars').value.trim(),
    mounts:            document.getElementById('mounts').value.trim(),
    terminal_mode:     document.getElementById('terminal-mode').checked,
    container_port:    parseInt(document.getElementById('container-port').value) || 0,
    auto_stop_min:     parseInt(document.getElementById('auto-stop').value),
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
    await api.createProject(data)
    window.location.href = 'dashboard.html'
  } catch (err) {
    alertEl.textContent   = err.message
    alertEl.style.display = 'block'
    btn.disabled          = false
    btn.textContent       = 'Save Project'
  }
}
