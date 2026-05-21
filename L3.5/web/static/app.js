const API_URL = '/events';

// ===== Пользовательская часть =====

// Загрузка списка мероприятий для пользователя
async function loadEvents() {
    try {
        const res = await fetch(API_URL);
        const events = await res.json();
        const container = document.getElementById('events-list');

        if (!events || events.length === 0) {
            container.innerHTML = '<p class="empty">Нет доступных мероприятий</p>';
            return;
        }

        container.innerHTML = events.map(e => `
            <div class="card">
                <h3>${e.name}</h3>
                <p class="event-date">📅 ${new Date(e.event_date).toLocaleString('ru-RU')}</p>
                <p class="event-capacity">🎫 Мест: ${e.capacity}</p>
                <button onclick="viewEvent(${e.id})">Смотреть места</button>
            </div>
        `).join('');
    } catch (err) {
        console.error('Ошибка загрузки мероприятий:', err);
    }
}

// Деталка мероприятия и места
async function viewEvent(id) {
    try {
        const res = await fetch(`${API_URL}/${id}`);
        const data = await res.json();

        document.getElementById('modal-title').innerText = data.event.name;
        document.getElementById('modal-date').innerText =
            '📅 ' + new Date(data.event.event_date).toLocaleString('ru-RU');
        document.getElementById('modal-free-count').innerText = data.free_seats_count;
        document.getElementById('event-detail').style.display = 'block';
        document.getElementById('event-detail').dataset.eventId = id;

        const seatsContainer = document.getElementById('seats-grid');
        seatsContainer.innerHTML = data.seats.map(s => {
            let statusClass = s.status; // 'free', 'reserved', 'confirmed'
            let btnText = '';
            let action = '';

            if (s.status === 'free') {
                btnText = 'Забронировать';
                action = `bookSeat(${id}, ${s.id})`;
            } else if (s.status === 'reserved') {
                btnText = 'Оплатить';
                action = `confirmSeat(${id}, ${s.id})`;
            }

            let statusLabel = {
                'free': 'Свободно',
                'reserved': 'Забронировано',
                'confirmed': 'Оплачено',
                'paid': 'Оплачено'
            }[s.status] || s.status;

            return `
                <div class="seat ${statusClass}">
                    <span class="seat-number">№${s.seat_number}</span>
                    <span class="seat-status">${statusLabel}</span>
                    ${action ? `<button onclick="${action}">${btnText}</button>` : ''}
                </div>
            `;
        }).join('');
    } catch (err) {
        console.error('Ошибка загрузки мероприятия:', err);
    }
}

// Бронирование
async function bookSeat(eventId, seatId) {
    try {
        const res = await fetch(`${API_URL}/${eventId}/book`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ seat_id: seatId })
        });
        if (res.ok) {
            alert('Забронировано! У вас есть 2 минуты на оплату.');
            viewEvent(eventId);
        } else {
            const text = await res.text();
            alert('Ошибка бронирования: ' + text);
        }
    } catch (err) {
        console.error('Ошибка бронирования:', err);
    }
}

// Оплата
async function confirmSeat(eventId, seatId) {
    try {
        const res = await fetch(`${API_URL}/${eventId}/confirm`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ seat_id: seatId })
        });
        if (res.ok) {
            alert('Оплата подтверждена!');
            viewEvent(eventId);
        } else {
            const text = await res.text();
            alert('Ошибка подтверждения: ' + text);
        }
    } catch (err) {
        console.error('Ошибка подтверждения:', err);
    }
}

// Закрыть модалку
function closeModal() {
    document.getElementById('event-detail').style.display = 'none';
}

// ===== Админская часть =====

// Загрузка списка мероприятий для админа (с подробностями)
async function loadEventsAdmin() {
    try {
        const res = await fetch(API_URL);
        const events = await res.json();
        const container = document.getElementById('events-list-admin');

        if (!events || events.length === 0) {
            container.innerHTML = '<p class="empty">Нет мероприятий</p>';
            return;
        }

        let html = '';
        for (const e of events) {
            // Загружаем детали для каждого мероприятия
            const detailRes = await fetch(`${API_URL}/${e.id}`);
            const detail = await detailRes.json();

            const totalSeats = detail.seats ? detail.seats.length : 0;
            const freeCount = detail.free_seats_count || 0;
            const reservedCount = detail.seats
                ? detail.seats.filter(s => s.status === 'reserved').length : 0;
            const confirmedCount = detail.seats
                ? detail.seats.filter(s => s.status === 'confirmed').length : 0;

            html += `
                <div class="admin-card">
                    <h3>${e.name}</h3>
                    <p class="event-date">📅 ${new Date(e.event_date).toLocaleString('ru-RU')}</p>
                    ${e.description ? `<p class="event-desc">${e.description}</p>` : ''}
                    <div class="stats">
                        <span class="stat free">Свободно: ${freeCount}</span>
                        <span class="stat reserved">Забронировано: ${reservedCount}</span>
                        <span class="stat confirmed">Оплачено: ${confirmedCount}</span>
                        <span class="stat total">Всего: ${totalSeats}</span>
                    </div>
                </div>
            `;
        }
        container.innerHTML = html;
    } catch (err) {
        console.error('Ошибка загрузки мероприятий (админ):', err);
    }
}

// Создание мероприятия (для админа)
async function createEvent(e) {
    e.preventDefault();

    const nameVal = document.getElementById('title').value.trim();
    const descVal = document.getElementById('description').value.trim();
    const dateVal = document.getElementById('date').value;
    const capVal = document.getElementById('capacity').value;

    if (!nameVal) {
        alert('Введите название мероприятия');
        return;
    }
    if (!dateVal) {
        alert('Выберите дату мероприятия');
        return;
    }
    if (!capVal || parseInt(capVal) <= 0) {
        alert('Укажите количество мест (больше 0)');
        return;
    }

    const parsedDate = new Date(dateVal);
    if (isNaN(parsedDate.getTime())) {
        alert('Некорректная дата');
        return;
    }

    const eventData = {
        name: nameVal,
        description: descVal,
        event_date: parsedDate.toISOString(),
        capacity: parseInt(capVal)
    };

    try {
        const res = await fetch(API_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(eventData)
        });

        if (res.ok) {
            alert('Мероприятие создано!');
            document.getElementById('create-event-form').reset();
            loadEventsAdmin();
        } else {
            const text = await res.text();
            alert('Ошибка создания: ' + text);
        }
    } catch (err) {
        console.error('Ошибка создания мероприятия:', err);
        alert('Ошибка сети: ' + err.message);
    }
}

// Автообновление детального вида каждые 10 секунд (для отслеживания отмены броней)
setInterval(() => {
    const modal = document.getElementById('event-detail');
    if (modal && modal.style.display === 'block' && modal.dataset.eventId) {
        viewEvent(parseInt(modal.dataset.eventId));
    }
}, 10000);