requireAuth()

let currentPage = 1
const PAGE_SIZE = 20

async function loadProjects(page = 1) {
  const grid = document.getElementById('projects-grid')
  try {
    const res = await api.getProjects(page, PAGE_SIZE)
    const projects = res.data
    const total = res.total

    if (!projects || projects.length === 0) {
      grid.innerHTML = `
        <div class="empty-state">
          <p>No projects yet.</p>
          <a href="add-project.html" class="btn btn-primary" style="margin-top:1rem">+ Add your first project</a>
        </div>`
      renderPagination(0, page)
      return
    }

    grid.innerHTML = projects.map(p => `
      <div class="project-card" onclick="window.location.href='project-details.html?id=${p.id}'">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:0.5rem">
          <span class="project-card-title">${escapeHtml(p.title)}</span>
          <span class="badge badge-${p.status}">${p.status}</span>
        </div>
        <p class="project-card-description">${escapeHtml(p.description || 'No description')}</p>
        <p class="project-card-image">🐳 ${escapeHtml(p.docker_image)}</p>
      </div>
    `).join('')

    renderPagination(total, page)
  } catch (err) {
    grid.innerHTML = `<div class="alert alert-error">${escapeHtml(err.message)}</div>`
  }
}

function renderPagination(total, page) {
  const totalPages = Math.ceil(total / PAGE_SIZE)
  const container = document.getElementById('pagination')
  if (!container) return
  if (totalPages <= 1) { container.innerHTML = ''; return }
  container.innerHTML = `
    <button onclick="changePage(${page - 1})" ${page <= 1 ? 'disabled' : ''}>← Prev</button>
    <span>Page ${page} / ${totalPages}</span>
    <button onclick="changePage(${page + 1})" ${page >= totalPages ? 'disabled' : ''}>Next →</button>
  `
}

function changePage(page) {
  currentPage = page
  loadProjects(page)
}

function escapeHtml(str) {
  const div = document.createElement('div')
  div.appendChild(document.createTextNode(String(str)))
  return div.innerHTML
}

loadProjects(currentPage)
