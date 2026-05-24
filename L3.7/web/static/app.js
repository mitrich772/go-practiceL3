'use strict';

const API = '';
const PAGE_SIZE = 20;

const state = {
    token: localStorage.getItem('wc_token') || '',
    user: JSON.parse(localStorage.getItem('wc_user') || 'null'),
    filters: { search: '' },
    offset: 0,
    hasMore: false,
};

let items = [];

function authHeaders(extra) {
    const headers = extra || {};
    if (state.token) headers.Authorization = 'Bearer ' + state.token;
    return headers;
}

function fmtDate(iso) {
    if (!iso) return '';
    return new Date(iso).toLocaleString('ru-RU');
}

function flash(el, msg, ok) {
    el.hidden = false;
    el.textContent = msg;
    el.className = 'flash ' + (ok ? 'ok' : 'err');
    setTimeout(() => { el.hidden = true; }, 3500);
}

function escapeHTML(s) {
    return String(s ?? '').replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
}

function itemSnapshot(raw) {
    if (!raw) return '—';
    const it = typeof raw === 'string' ? JSON.parse(raw) : raw;
    return [
        it.name,
        'SKU: ' + it.sku,
        'qty: ' + it.quantity,
        it.location ? 'loc: ' + it.location : '',
    ].filter(Boolean).join(' / ');
}

function canWrite() {
    return state.user && (state.user.role === 'admin' || state.user.role === 'manager');
}

function canDelete() {
    return state.user && state.user.role === 'admin';
}

function canHistory() {
    return state.user && state.user.role === 'admin';
}

function buildQuery(extra) {
    const q = new URLSearchParams();
    if (state.filters.search) q.set('search', state.filters.search);
    if (extra) {
        for (const [k, v] of Object.entries(extra)) q.set(k, v);
    }
    const s = q.toString();
    return s ? '?' + s : '';
}

async function apiLogin(payload) {
    const res = await fetch(API + '/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiCreate(payload) {
    const res = await fetch(API + '/items', {
        method: 'POST',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiUpdate(id, payload) {
    const res = await fetch(API + '/items/' + id, {
        method: 'PUT',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiDelete(id) {
    const res = await fetch(API + '/items/' + id, {
        method: 'DELETE',
        headers: authHeaders(),
    });
    if (!res.ok && res.status !== 204) throw new Error(await res.text());
}

async function apiList() {
    const qs = buildQuery({ limit: PAGE_SIZE, offset: state.offset, sort: 'updated_at', order: 'desc' });
    const res = await fetch(API + '/items' + qs, { headers: authHeaders() });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiHistory(id) {
    const res = await fetch(API + '/items/' + id + '/history', { headers: authHeaders() });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

function renderAuth() {
    const label = document.getElementById('current-user');
    if (!state.user) {
        label.textContent = 'не авторизован';
        document.getElementById('create-form').hidden = true;
        return;
    }
    label.textContent = state.user.username + ' / ' + state.user.role;
    document.getElementById('create-form').hidden = !canWrite();
}

function renderItems(rows) {
    const tbody = document.querySelector('#items-table tbody');
    tbody.innerHTML = '';
    if (!state.user) {
        tbody.innerHTML = '<tr><td colspan="8" class="empty">Выберите роль и войдите</td></tr>';
        return;
    }
    if (!rows || rows.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="empty">Товаров нет</td></tr>';
        return;
    }
    for (const it of rows) {
        const actions = [];
        actions.push(`<button class="secondary" data-act="edit" data-id="${it.id}" ${canWrite() ? '' : 'disabled'}>✎</button>`);
        actions.push(`<button class="danger" data-act="delete" data-id="${it.id}" ${canDelete() ? '' : 'disabled'}>✕</button>`);
        if (canHistory()) actions.push(`<button class="secondary" data-act="history" data-id="${it.id}">история</button>`);
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${it.id}</td>
            <td>${escapeHTML(it.name)}</td>
            <td><code>${escapeHTML(it.sku)}</code></td>
            <td class="num">${it.quantity}</td>
            <td>${escapeHTML(it.location)}</td>
            <td>${escapeHTML(it.description)}</td>
            <td>${fmtDate(it.updated_at)}</td>
            <td class="row-actions">${actions.join(' ')}</td>
        `;
        tbody.appendChild(tr);
    }
}

function renderHistory(id, rows) {
    document.getElementById('history-block').hidden = false;
    document.getElementById('history-title').textContent = '#' + id;
    const tbody = document.querySelector('#history-table tbody');
    tbody.innerHTML = '';
    if (!rows || rows.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty">История пуста</td></tr>';
        return;
    }
    for (const h of rows) {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${fmtDate(h.changed_at)}</td>
            <td><span class="tag ${h.action.toLowerCase()}">${h.action}</span></td>
            <td>${escapeHTML(h.actor)}</td>
            <td>${escapeHTML(h.actor_role)}</td>
            <td>${escapeHTML(itemSnapshot(h.old_data))}</td>
            <td>${escapeHTML(itemSnapshot(h.new_data))}</td>
        `;
        tbody.appendChild(tr);
    }
}

function updatePager() {
    const page = Math.floor(state.offset / PAGE_SIZE) + 1;
    document.getElementById('page-info').textContent = 'страница ' + page;
    document.getElementById('prev-page').disabled = state.offset === 0;
    document.getElementById('next-page').disabled = !state.hasMore;
}

async function refresh() {
    renderAuth();
    if (!state.user) {
        renderItems([]);
        return;
    }
    try {
        const listResp = await apiList();
        items = listResp.items || [];
        state.hasMore = !!listResp.has_more;
        renderItems(items);
        updatePager();
    } catch (e) {
        console.error('refresh failed', e);
        if (String(e.message).includes('unauthorized') || String(e.message).includes('invalid token')) {
            logout();
        }
    }
}

function readFiltersFromUI() {
    state.filters.search = document.getElementById('q-search').value.trim();
    state.offset = 0;
}

function readCreateForm(prefix) {
    return {
        name: document.getElementById(prefix + '-name').value.trim(),
        sku: document.getElementById(prefix + '-sku').value.trim(),
        quantity: parseInt(document.getElementById(prefix + '-quantity').value, 10),
        location: document.getElementById(prefix + '-location').value.trim(),
        description: document.getElementById(prefix + '-description').value.trim(),
    };
}

function logout() {
    state.token = '';
    state.user = null;
    localStorage.removeItem('wc_token');
    localStorage.removeItem('wc_user');
    refresh();
}

function bind() {
    document.getElementById('login-form').addEventListener('submit', async (ev) => {
        ev.preventDefault();
        const role = document.getElementById('role-select').value;
        const username = document.getElementById('username').value.trim() || role;
        const resp = await apiLogin({ username, role });
        state.token = resp.token;
        state.user = resp.user;
        localStorage.setItem('wc_token', state.token);
        localStorage.setItem('wc_user', JSON.stringify(state.user));
        await refresh();
    });

    document.getElementById('create-form').addEventListener('submit', async (ev) => {
        ev.preventDefault();
        const flashEl = document.getElementById('form-flash');
        try {
            await apiCreate(readCreateForm('f'));
            flash(flashEl, 'Товар добавлен', true);
            ev.target.reset();
            await refresh();
        } catch (e) {
            flash(flashEl, 'Ошибка: ' + e.message, false);
        }
    });

    document.getElementById('filter-form').addEventListener('submit', (ev) => {
        ev.preventDefault();
        readFiltersFromUI();
        refresh();
    });

    document.getElementById('reset-filters').addEventListener('click', () => {
        document.getElementById('q-search').value = '';
        readFiltersFromUI();
        refresh();
    });

    document.getElementById('prev-page').addEventListener('click', () => {
        if (state.offset === 0) return;
        state.offset = Math.max(0, state.offset - PAGE_SIZE);
        refresh();
    });

    document.getElementById('next-page').addEventListener('click', () => {
        if (!state.hasMore) return;
        state.offset += PAGE_SIZE;
        refresh();
    });

    document.querySelector('#items-table tbody').addEventListener('click', async (ev) => {
        const btn = ev.target.closest('button[data-act]');
        if (!btn) return;
        const id = Number(btn.dataset.id);
        if (btn.dataset.act === 'delete') {
            if (!confirm('Удалить товар #' + id + '?')) return;
            try {
                await apiDelete(id);
                await refresh();
            } catch (e) {
                alert('Не удалось удалить: ' + e.message);
            }
        } else if (btn.dataset.act === 'edit') {
            const it = items.find(x => x.id === id);
            if (!it) return;
            openEditModal(it);
        } else if (btn.dataset.act === 'history') {
            try {
                const resp = await apiHistory(id);
                renderHistory(id, resp.history || []);
            } catch (e) {
                alert('Не удалось загрузить историю: ' + e.message);
            }
        }
    });

    const modal = document.getElementById('edit-modal');
    document.getElementById('edit-close').addEventListener('click', () => modal.hidden = true);
    modal.addEventListener('click', (ev) => {
        if (ev.target === modal) modal.hidden = true;
    });

    document.getElementById('edit-form').addEventListener('submit', async (ev) => {
        ev.preventDefault();
        const id = Number(document.getElementById('edit-form').dataset.id);
        try {
            await apiUpdate(id, readCreateForm('e'));
            modal.hidden = true;
            await refresh();
        } catch (e) {
            alert('Не удалось сохранить: ' + e.message);
        }
    });
}

function openEditModal(it) {
    const form = document.getElementById('edit-form');
    form.dataset.id = String(it.id);
    document.getElementById('e-name').value = it.name;
    document.getElementById('e-sku').value = it.sku;
    document.getElementById('e-quantity').value = it.quantity;
    document.getElementById('e-location').value = it.location || '';
    document.getElementById('e-description').value = it.description || '';
    document.getElementById('edit-modal').hidden = false;
}

bind();
refresh();
