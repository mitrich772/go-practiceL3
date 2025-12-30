const API_BASE = "http://localhost:8080";

async function shorten() {
    const originalUrl = document.getElementById("originalUrl").value.trim();
    const customAlias = document.getElementById("customAlias").value.trim();

    if (!originalUrl) {
        alert("Введите URL");
        return;
    }

    const payload = {
        url: originalUrl,
        alias: customAlias || ""
    };

    try {
        const res = await fetch(`${API_BASE}/shorten`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(payload)
        });

        if (res.status === 409) {
            alert("Такая короткая ссылка уже существует");
            return;
        }

        if (!res.ok) {
            alert("Ошибка создания ссылки");
            return;
        }

        const data = await res.json();

        document.getElementById("shortenResult").innerHTML = `
            <strong>Короткая ссылка:</strong><br />
            <a href="${data.short_url}" target="_blank">${data.short_url}</a>
        `;
    } catch (err) {
        alert("Сервис недоступен");
    }
}


async function loadAnalytics() {
    const alias = document.getElementById("analyticsAlias").value;

    if (!alias) {
        alert("Введите short_url");
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/analytics/${alias}`);

        if (!res.ok) {
            throw new Error("Ошибка получения аналитики");
        }

        const data = await res.json();

        document.getElementById("analyticsResult").textContent =
            JSON.stringify(data, null, 2);
    } catch (err) {
        alert(err.message);
    }
}
