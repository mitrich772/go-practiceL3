/* ------------------ CREATE ------------------ */
document.getElementById("notifyForm").addEventListener("submit", async (e) => {
    e.preventDefault();

    const message = document.getElementById("message").value.trim();
    const date = document.getElementById("date").value;
    const time = document.getElementById("time").value;

    const local = new Date(`${date}T${time}`);
    const sendAt = local.toISOString();

    const data = { message, send_at: sendAt };

    try {
        const res = await fetch("/notify", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(data)
        });

        const text = await res.text();

        if (res.ok) {
            document.getElementById("result").innerText =
                "Notification created! ID: " + JSON.parse(text).id;
        } else {
            document.getElementById("result").innerText = "Error: " + text;
        }
    } catch (err) {
        document.getElementById("result").innerText = "Network error.";
    }
});

/* ------------------ CHECK STATUS ------------------ */
document.getElementById("checkForm").addEventListener("submit", async (e) => {
    e.preventDefault();

    const id = document.getElementById("checkId").value.trim();
    const box = document.getElementById("checkResult");

    try {
        const res = await fetch(`/notify/${id}`);

        if (!res.ok) {
            box.innerText = "Not found";
            return;
        }

        const data = await res.json();
        box.innerText = JSON.stringify(data, null, 2);
    } catch (err) {
        box.innerText = "Network error.";
    }
});

/* ------------------ CANCEL ------------------ */
document.getElementById("cancelForm").addEventListener("submit", async (e) => {
    e.preventDefault();

    const id = document.getElementById("cancelId").value.trim();
    const box = document.getElementById("cancelResult");

    try {
        const res = await fetch(`/notify/${id}`, { method: "DELETE" });

        if (res.ok) {
            box.innerText = "Notification canceled!";
        } else {
            box.innerText = "Not found";
        }
    } catch (err) {
        box.innerText = "Network error.";
    }
});
