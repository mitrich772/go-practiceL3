'use strict';

const API = '';
const PAGE_SIZE = 20;

const state = {
    filters: {
        from: '',
        to: '',
        type: '',
        category: '',
    },
    offset: 0,
    hasMore: false,
};

// ===== utils =====

function fmtMoney(v) {
    return Number(v).toLocaleString('ru-RU', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
    });
}

function fmtDate(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    return d.toLocaleString('ru-RU');
}

function flash(el, msg, ok) {
    el.hidden = false;
    el.textContent = msg;
    el.className = 'flash ' + (ok ? 'ok' : 'err');
    setTimeout(() => { el.hidden = true; }, 3500);
}

function rfc3339FromLocalDatetime(value) {
    if (!value) return '';
    const d = new Date(value);
    if (Number.isNaN(d.getTime())) return '';
    return d.toISOString();
}

function dayStartRFC(value) {
    if (!value) return '';
    return new Date(value + 'T00:00:00').toISOString();
}

function dayEndRFC(value) {
    if (!value) return '';
    return new Date(value + 'T23:59:59').toISOString();
}

function buildQuery(extra) {
    const q = new URLSearchParams();
    if (state.filters.from) q.set('from', dayStartRFC(state.filters.from));
    if (state.filters.to) q.set('to', dayEndRFC(state.filters.to));
    if (state.filters.type) q.set('type', state.filters.type);
    if (state.filters.category) q.set('category', state.filters.category);
    if (extra) {
        for (const [k, v] of Object.entries(extra)) q.set(k, v);
    }
    const s = q.toString();
    return s ? '?' + s : '';
}

// ===== API =====

async function apiCreate(payload) {
    const res = await fetch(API + '/items', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiUpdate(id, payload) {
    const res = await fetch(API + '/items/' + id, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiDelete(id) {
    const res = await fetch(API + '/items/' + id, { method: 'DELETE' });
    if (!res.ok && res.status !== 204) throw new Error(await res.text());
}

async function apiList() {
    const qs = buildQuery({ limit: PAGE_SIZE, offset: state.offset, sort: 'occurred_at', order: 'desc' });
    const res = await fetch(API + '/items' + qs);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

// apiAnalytics: согласно ТЗ передаются только from и to.
async function apiAnalytics() {
    const q = new URLSearchParams();
    if (state.filters.from) q.set('from', dayStartRFC(state.filters.from));
    if (state.filters.to) q.set('to', dayEndRFC(state.filters.to));
    const qs = q.toString() ? '?' + q.toString() : '';
    const res = await fetch(API + '/analytics' + qs);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

// ===== render =====

// renderAnalytics показывает строго 5 метрик из ТЗ:
// count, sum, avg, median, p90.
function renderAnalytics(a) {
    document.getElementById('m-count').textContent = a.count;
    document.getElementById('m-sum').textContent = fmtMoney(a.sum);
    document.getElementById('m-avg').textContent = fmtMoney(a.avg);
    document.getElementById('m-median').textContent = fmtMoney(a.median);
    document.getElementById('m-p90').textContent = fmtMoney(a.p90);
}

function renderItems(items) {
    const tbody = document.querySelector('#items-table tbody');
    tbody.innerHTML = '';
    if (!items || items.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="empty">Записей нет</td></tr>';
        return;
    }
    for (const it of items) {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${it.id}</td>
            <td><span class="tag ${it.type}">${it.type === 'income' ? 'доход' : 'расход'}</span></td>
            <td class="num">${fmtMoney(it.amount)}</td>
            <td>${escapeHTML(it.category)}</td>
            <td>${fmtDate(it.occurred_at)}</td>
            <td>${escapeHTML(it.note || '')}</td>
            <td class="row-actions">
                <button class="secondary" data-act="edit"   data-id="${it.id}">✎</button>
                <button class="danger"    data-act="delete" data-id="${it.id}">✕</button>
            </td>
        `;
        tbody.appendChild(tr);
    }
}

function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
}

function updateCSVLink() {
    const link = document.getElementById('csv-link');
    link.href = API + '/items.csv' + buildQuery();
}

function updatePager() {
    const page = Math.floor(state.offset / PAGE_SIZE) + 1;
    document.getElementById('page-info').textContent = 'страница ' + page;
    document.getElementById('prev-page').disabled = state.offset === 0;
    document.getElementById('next-page').disabled = !state.hasMore;
}

// ===== controllers =====

let items = [];

async function refresh() {
    try {
        const [listResp, anResp] = await Promise.all([apiList(), apiAnalytics()]);
        items = listResp.items || [];
        state.hasMore = !!listResp.has_more;
        renderItems(items);
        renderAnalytics(anResp);
        updatePager();
        updateCSVLink();
    } catch (e) {
        console.error('refresh failed', e);
    }
}

function readFiltersFromUI() {
    state.filters.from = document.getElementById('q-from').value;
    state.filters.to = document.getElementById('q-to').value;
    state.filters.type = document.getElementById('q-type').value;
    state.filters.category = document.getElementById('q-category').value.trim();
    state.offset = 0;
}

function bind() {
    document.getElementById('create-form').addEventListener('submit', async (ev) => {
        ev.preventDefault();
        const payload = {
            type: document.getElementById('f-type').value,
            amount: parseFloat(document.getElementById('f-amount').value),
            category: document.getElementById('f-category').value.trim(),
            note: document.getElementById('f-note').value.trim(),
            occurred_at: rfc3339FromLocalDatetime(document.getElementById('f-date').value),
        };
        const flashEl = document.getElementById('form-flash');
        try {
            await apiCreate(payload);
            flash(flashEl, 'Запись добавлена', true);
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
        document.getElementById('q-from').value = '';
        document.getElementById('q-to').value = '';
        document.getElementById('q-type').value = '';
        document.getElementById('q-category').value = '';
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
            if (!confirm('Удалить запись #' + id + '?')) return;
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
        const payload = {
            type: document.getElementById('e-type').value,
            amount: parseFloat(document.getElementById('e-amount').value),
            category: document.getElementById('e-category').value.trim(),
            note: document.getElementById('e-note').value.trim(),
            occurred_at: rfc3339FromLocalDatetime(document.getElementById('e-date').value),
        };
        try {
            await apiUpdate(id, payload);
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
    document.getElementById('e-type').value = it.type;
    document.getElementById('e-amount').value = it.amount;
    document.getElementById('e-category').value = it.category;
    document.getElementById('e-note').value = it.note || '';
    document.getElementById('e-date').value = toLocalDatetimeInput(it.occurred_at);
    document.getElementById('edit-modal').hidden = false;
}

function toLocalDatetimeInput(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    const pad = n => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

window.addEventListener('DOMContentLoaded', () => {
    bind();
    refresh();
});
