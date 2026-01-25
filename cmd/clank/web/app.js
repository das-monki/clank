// State
let projects = [];
let tasks = [];
let currentProjectFilter = '';

// DOM Elements
const projectFilter = document.getElementById('project-filter');
const taskModal = document.getElementById('task-modal');
const projectModal = document.getElementById('project-modal');
const taskForm = document.getElementById('task-form');
const projectForm = document.getElementById('project-form');

// API Helpers
async function api(method, path, body = null) {
    const opts = {
        method,
        headers: { 'Content-Type': 'application/json' },
    };
    if (body) opts.body = JSON.stringify(body);
    const res = await fetch('/api' + path, opts);
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'API error');
    }
    if (res.status === 204) return null;
    return res.json();
}

// Projects
async function loadProjects() {
    projects = await api('GET', '/projects');
    renderProjectFilter();
    renderTaskProjectSelect();
}

function renderProjectFilter() {
    const current = projectFilter.value;
    projectFilter.innerHTML = '<option value="">All Projects</option>';
    projects.forEach(p => {
        const opt = document.createElement('option');
        opt.value = p.id;
        opt.textContent = p.name;
        projectFilter.appendChild(opt);
    });
    projectFilter.value = current;
}

function renderTaskProjectSelect() {
    const select = document.getElementById('task-project');
    select.innerHTML = '';
    projects.forEach(p => {
        const opt = document.createElement('option');
        opt.value = p.id;
        opt.textContent = p.name;
        select.appendChild(opt);
    });
}

// Tasks
async function loadTasks() {
    let path = '/tasks';
    if (currentProjectFilter) {
        path += '?project_id=' + currentProjectFilter;
    }
    tasks = await api('GET', path);
    renderKanban();
}

function renderKanban() {
    const columns = ['backlog', 'todo', 'in_progress', 'done'];
    columns.forEach(status => {
        const container = document.getElementById(status);
        container.innerHTML = '';

        const columnTasks = tasks
            .filter(t => t.status === status && !t.parent_id)
            .sort((a, b) => a.position - b.position);

        columnTasks.forEach(task => {
            container.appendChild(createTaskCard(task));
        });
    });
}

function createTaskCard(task) {
    const card = document.createElement('div');
    card.className = 'task-card';
    card.dataset.id = task.id;
    card.dataset.position = task.position;

    const project = projects.find(p => p.id === task.project_id);
    const subtaskCount = tasks.filter(t => t.parent_id === task.id).length;

    card.innerHTML = `
        <div class="title">${escapeHtml(task.title)}</div>
        <div class="meta">
            <span class="project-tag">${escapeHtml(project?.name || 'Unknown')}</span>
            ${subtaskCount > 0 ? `<span class="subtask-count">${subtaskCount} subtasks</span>` : ''}
        </div>
    `;

    card.addEventListener('click', () => openTaskModal(task));
    return card;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Drag and Drop
function initSortable() {
    const columns = document.querySelectorAll('.tasks');
    columns.forEach(column => {
        new Sortable(column, {
            group: 'tasks',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            onEnd: handleDragEnd
        });
    });
}

async function handleDragEnd(evt) {
    const taskId = evt.item.dataset.id;
    const newStatus = evt.to.id;
    const cards = Array.from(evt.to.children);
    const index = cards.indexOf(evt.item);

    // Calculate new position
    let newPosition;
    if (cards.length === 1) {
        newPosition = 1.0;
    } else if (index === 0) {
        const nextPos = parseFloat(cards[1].dataset.position);
        newPosition = nextPos / 2;
    } else if (index === cards.length - 1) {
        const prevPos = parseFloat(cards[index - 1].dataset.position);
        newPosition = prevPos + 1;
    } else {
        const prevPos = parseFloat(cards[index - 1].dataset.position);
        const nextPos = parseFloat(cards[index + 1].dataset.position);
        newPosition = (prevPos + nextPos) / 2;
    }

    evt.item.dataset.position = newPosition;

    try {
        await api('POST', `/tasks/${taskId}/move`, {
            status: newStatus,
            position: newPosition
        });
        // Update local state
        const task = tasks.find(t => t.id === taskId);
        if (task) {
            task.status = newStatus;
            task.position = newPosition;
        }
    } catch (err) {
        console.error('Failed to move task:', err);
        loadTasks(); // Reload on error
    }
}

// Track current task's parent for navigation
let currentTaskParentId = null;

// Task Modal
function openTaskModal(task = null) {
    const title = document.getElementById('modal-title');
    const form = taskForm;
    const deleteBtn = document.getElementById('delete-task-btn');
    const subtasksSection = document.getElementById('subtasks-section');
    const subtaskInput = document.getElementById('subtask-title-input');
    const parentNav = document.getElementById('parent-task-nav');

    // Reset delete button state
    deleteBtn.textContent = 'Delete';
    deleteBtn.classList.remove('confirming');

    if (task) {
        title.textContent = task.parent_id ? 'Edit Subtask' : 'Edit Task';
        document.getElementById('task-id').value = task.id;
        document.getElementById('task-title').value = task.title;
        document.getElementById('task-project').value = task.project_id;
        document.getElementById('task-description').value = task.description || '';
        document.getElementById('task-spec').value = task.spec || '';
        deleteBtn.style.display = 'block';

        // Show/hide parent navigation
        currentTaskParentId = task.parent_id || null;
        if (task.parent_id) {
            parentNav.classList.add('visible');
            subtasksSection.style.display = 'none';
        } else {
            parentNav.classList.remove('visible');
            subtasksSection.style.display = 'block';
            subtaskInput.value = '';
            renderSubtasks(task.id);
        }
    } else {
        title.textContent = 'New Task';
        form.reset();
        document.getElementById('task-id').value = '';
        deleteBtn.style.display = 'none';
        subtasksSection.style.display = 'none';
        parentNav.classList.remove('visible');
        currentTaskParentId = null;
    }

    taskModal.classList.add('active');
}

function renderSubtasks(parentId) {
    const list = document.getElementById('subtasks-list');
    const subtasks = tasks.filter(t => t.parent_id === parentId);

    list.innerHTML = '';
    subtasks.forEach(st => {
        const li = document.createElement('li');
        li.innerHTML = `
            <span class="status">${st.status}</span>
            <span>${escapeHtml(st.title)}</span>
        `;
        li.addEventListener('click', (e) => {
            e.stopPropagation();
            openTaskModal(st);
        });
        list.appendChild(li);
    });
}

// Event Listeners
document.getElementById('new-project-btn').addEventListener('click', () => {
    projectForm.reset();
    projectModal.classList.add('active');
});

document.getElementById('new-task-btn').addEventListener('click', () => {
    openTaskModal(null);
});

projectFilter.addEventListener('change', (e) => {
    currentProjectFilter = e.target.value;
    loadTasks();
});

// Close modals
document.querySelectorAll('.modal .close').forEach(btn => {
    btn.addEventListener('click', () => {
        taskModal.classList.remove('active');
        projectModal.classList.remove('active');
    });
});

// Back to parent task
document.getElementById('back-to-parent-btn').addEventListener('click', () => {
    if (currentTaskParentId) {
        const parentTask = tasks.find(t => t.id === currentTaskParentId);
        if (parentTask) {
            openTaskModal(parentTask);
        }
    }
});

document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.classList.remove('active');
        }
    });
});

// Project form
projectForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('project-name').value;
    const description = document.getElementById('project-description').value;

    try {
        await api('POST', '/projects', { name, description });
        projectModal.classList.remove('active');
        await loadProjects();
    } catch (err) {
        alert('Failed to create project: ' + err.message);
    }
});

// Task form
taskForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = document.getElementById('task-id').value;
    const data = {
        title: document.getElementById('task-title').value,
        project_id: document.getElementById('task-project').value,
        description: document.getElementById('task-description').value,
        spec: document.getElementById('task-spec').value,
    };

    try {
        if (id) {
            await api('PATCH', `/tasks/${id}`, data);
        } else {
            await api('POST', '/tasks', { ...data, status: 'backlog' });
        }
        await loadTasks();

        // If saving a subtask, go back to parent; otherwise close modal
        if (currentTaskParentId) {
            const parentTask = tasks.find(t => t.id === currentTaskParentId);
            if (parentTask) {
                openTaskModal(parentTask);
                return;
            }
        }
        taskModal.classList.remove('active');
    } catch (err) {
        alert('Failed to save task: ' + err.message);
    }
});

// Delete task - requires double-click confirmation
let deleteConfirmPending = false;
const deleteBtn = document.getElementById('delete-task-btn');

deleteBtn.addEventListener('click', async () => {
    const id = document.getElementById('task-id').value;
    if (!id) return;

    if (!deleteConfirmPending) {
        // First click - ask for confirmation
        deleteConfirmPending = true;
        deleteBtn.textContent = 'Confirm Delete?';
        deleteBtn.classList.add('confirming');
        // Reset after 3 seconds
        setTimeout(() => {
            if (deleteConfirmPending) {
                deleteConfirmPending = false;
                deleteBtn.textContent = 'Delete';
                deleteBtn.classList.remove('confirming');
            }
        }, 3000);
        return;
    }

    // Second click - actually delete
    deleteConfirmPending = false;
    deleteBtn.textContent = 'Delete';
    deleteBtn.classList.remove('confirming');

    try {
        await api('DELETE', `/tasks/${id}`);
        taskModal.classList.remove('active');
        await loadTasks();
    } catch (err) {
        alert('Failed to delete task: ' + err.message);
    }
});

// Add subtask
document.getElementById('add-subtask-btn').addEventListener('click', async () => {
    const parentId = document.getElementById('task-id').value;
    const projectId = document.getElementById('task-project').value;
    const input = document.getElementById('subtask-title-input');
    const title = input.value.trim();

    if (!title) {
        input.focus();
        return;
    }

    try {
        await api('POST', '/tasks', {
            title,
            project_id: projectId,
            parent_id: parentId,
            status: 'backlog'
        });
        input.value = '';
        await loadTasks();
        renderSubtasks(parentId);
    } catch (err) {
        alert('Failed to create subtask: ' + err.message);
    }
});

// Allow Enter key to add subtask
document.getElementById('subtask-title-input').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        e.preventDefault();
        document.getElementById('add-subtask-btn').click();
    }
});

// Fullscreen editor helper
function setupFullscreenEditor(config) {
    const { expandBtn, collapseBtn, fullscreenEl, modalTextarea } = config;
    const fsTextarea = fullscreenEl.querySelector('.fullscreen-textarea');
    const fsPreview = fullscreenEl.querySelector('.fullscreen-preview');
    const toggleBtns = fullscreenEl.querySelectorAll('.toggle-btn');

    // Expand
    expandBtn.addEventListener('click', () => {
        fsTextarea.value = modalTextarea.value;
        fullscreenEl.classList.add('active');
        fullscreenEl.classList.remove('view-mode');
        toggleBtns.forEach(btn => btn.classList.toggle('active', btn.dataset.mode === 'edit'));
        fsTextarea.focus();
    });

    // Collapse
    collapseBtn.addEventListener('click', () => {
        modalTextarea.value = fsTextarea.value;
        fullscreenEl.classList.remove('active', 'view-mode');
    });

    // Edit/View toggle
    toggleBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const mode = btn.dataset.mode;
            toggleBtns.forEach(b => b.classList.toggle('active', b === btn));

            if (mode === 'view') {
                fsPreview.innerHTML = marked.parse(fsTextarea.value || '*No content*');
                fullscreenEl.classList.add('view-mode');
            } else {
                fullscreenEl.classList.remove('view-mode');
                fsTextarea.focus();
            }
        });
    });

    return { fullscreenEl, fsTextarea, modalTextarea };
}

// Setup description fullscreen editor
const descEditor = setupFullscreenEditor({
    expandBtn: document.getElementById('expand-desc-btn'),
    collapseBtn: document.getElementById('collapse-desc-btn'),
    fullscreenEl: document.getElementById('desc-fullscreen'),
    modalTextarea: document.getElementById('task-description')
});

// Setup spec fullscreen editor
const specEditor = setupFullscreenEditor({
    expandBtn: document.getElementById('expand-spec-btn'),
    collapseBtn: document.getElementById('collapse-spec-btn'),
    fullscreenEl: document.getElementById('spec-fullscreen'),
    modalTextarea: document.getElementById('task-spec')
});

// Escape key to collapse any active fullscreen editor
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        [descEditor, specEditor].forEach(({ fullscreenEl, fsTextarea, modalTextarea }) => {
            if (fullscreenEl.classList.contains('active')) {
                modalTextarea.value = fsTextarea.value;
                fullscreenEl.classList.remove('active', 'view-mode');
            }
        });
    }
});

// Mobile tab navigation
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        const column = btn.dataset.column;

        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        // Update visible column
        document.querySelectorAll('.column').forEach(col => col.classList.remove('active'));
        document.querySelector(`.column[data-status="${column}"]`).classList.add('active');
    });
});

// Initialize
async function init() {
    await loadProjects();
    await loadTasks();
    initSortable();
}

init();
