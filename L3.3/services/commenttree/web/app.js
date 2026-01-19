(() => {
  const BASE = ""; // если фронт и API на одном домене
  const $ = (id) => document.getElementById(id);

  let limit = 10;
  let offset = 0;

  // режим: либо корни, либо поиск
  let mode = "roots"; // "roots" | "search"
  let searchText = "";

  // сортировка поиска (из dropdown)
  let searchSort = "rank";      // rank|created_at|id
  let searchOrder = "desc";     // asc|desc

  // ----- helpers -----

  function setStatus(msg) {
    const el = $("status");
    if (el) el.textContent = msg || "";
  }

  function setModeLabel() {
    const el = $("modeLabel");
    if (!el) return;
    if (mode === "roots") {
      el.textContent = "Режим: корневые";
    } else {
      el.textContent = `Режим: поиск (${searchText}), sort=${searchSort}, order=${searchOrder}`;
    }
  }

  function clearComments() {
    const box = $("comments");
    if (box) box.innerHTML = "";
  }

  function el(tag, text) {
    const e = document.createElement(tag);
    if (text != null) e.textContent = text;
    return e;
  }

  async function apiFetch(path, opts = {}) {
    const res = await fetch(BASE + path, {
      headers: { "Content-Type": "application/json", ...(opts.headers || {}) },
      ...opts,
    });

    const text = await res.text();
    let data = null;

    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      data = text;
    }

    if (!res.ok) {
      throw new Error(typeof data === "string" ? data : (data?.error || text || "request failed"));
    }
    return data;
  }

  // ----- API -----

  async function createComment(body, parentId = null) {
    const payload = parentId ? { parent_id: parentId, body } : { body };
    const data = await apiFetch("/comments", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return data.created_comment;
  }

  async function getChildren(id) {
    const data = await apiFetch(`/comments?parent=${encodeURIComponent(id)}&depth=1`);
    return data.tree?.children || [];
  }

  async function deleteComment(id) {
    const data = await apiFetch(`/comments/${encodeURIComponent(id)}`, { method: "DELETE" });
    return data.deleted_id;
  }

  async function getRoots(limit, offset) {
    // корни сортируем по времени (как было)
    return await apiFetch(`/roots?limit=${limit}&offset=${offset}&sort=created_at&order=desc`);
  }

  async function searchComments(q, limit, offset, sort, order) {
    return await apiFetch(
      `/search?q=${encodeURIComponent(q)}&limit=${limit}&offset=${offset}&sort=${encodeURIComponent(sort)}&order=${encodeURIComponent(order)}`
    );
  }

  // ----- UI render -----

  function renderComment(node, level = 0) {
    const wrap = document.createElement("div");
    wrap.style.marginLeft = (level * 20) + "px";
    wrap.style.padding = "8px 0";

    wrap.appendChild(el("div", `#${node.id}`));
    wrap.appendChild(el("div", node.body || ""));
    wrap.appendChild(el("div", node.created_at || ""));

    // rank отображаем только если сортировка rank и сервер отдал rank
    if (mode === "search" && searchSort === "rank" && node.rank != null) {
      wrap.appendChild(el("div", `rank: ${node.rank}`));
    }

    const actions = document.createElement("div");

    // Удалить
    const btnDel = el("button", "Удалить");
    btnDel.type = "button";
    btnDel.addEventListener("click", async (e) => {
      e.preventDefault();
      e.stopPropagation();

      try {
        await deleteComment(node.id);
        await loadList();
      } catch (err) {
        alert("Ошибка удаления: " + err.message);
      }
    });
    actions.appendChild(btnDel);

    // Ответить
    const btnReply = el("button", "Ответить");
    btnReply.type = "button";
    actions.appendChild(btnReply);

    // Показать/скрыть ответы
    const btnToggle = el("button", "Показать ответы");
    btnToggle.type = "button";
    actions.appendChild(btnToggle);

    wrap.appendChild(actions);

    // reply box
    const replyBox = document.createElement("div");
    replyBox.style.display = "none";
    replyBox.style.marginTop = "6px";

    const ta = document.createElement("textarea");
    ta.placeholder = `Ответ для #${node.id}`;
    replyBox.appendChild(ta);

    const btnSend = el("button", "Отправить");
    btnSend.type = "button";
    replyBox.appendChild(btnSend);

    const btnCancel = el("button", "Отмена");
    btnCancel.type = "button";
    btnCancel.onclick = () => (replyBox.style.display = "none");
    replyBox.appendChild(btnCancel);

    wrap.appendChild(replyBox);

    btnReply.onclick = () => {
      replyBox.style.display = replyBox.style.display === "none" ? "block" : "none";
    };

    // children container
    const childrenWrap = document.createElement("div");
    childrenWrap.style.marginTop = "6px";
    wrap.appendChild(childrenWrap);

    wrap.appendChild(document.createElement("hr"));

    let opened = false;

    async function loadReplies() {
      childrenWrap.textContent = "Загрузка...";
      const kids = await getChildren(node.id);

      childrenWrap.innerHTML = "";
      if (kids.length === 0) {
        childrenWrap.textContent = "Нет ответов";
        return;
      }

      for (const ch of kids) {
        childrenWrap.appendChild(renderComment(ch, level + 1));
      }
    }

    btnToggle.onclick = async () => {
      opened = !opened;

      if (!opened) {
        childrenWrap.innerHTML = "";
        btnToggle.textContent = "Показать ответы";
        return;
      }

      btnToggle.textContent = "Скрыть ответы";
      try {
        await loadReplies();
      } catch (err) {
        opened = false;
        childrenWrap.innerHTML = "";
        btnToggle.textContent = "Показать ответы";
        alert("Ошибка загрузки ответов: " + err.message);
      }
    };

    btnSend.onclick = async () => {
      const txt = (ta.value || "").trim();
      if (!txt) return alert("Пустой текст");

      try {
        await createComment(txt, node.id);
        ta.value = "";
        replyBox.style.display = "none";

        if (opened) {
          await loadReplies();
        }
      } catch (err) {
        alert("Ошибка создания: " + err.message);
      }
    };

    return wrap;
  }

  // ----- load list (roots or search) -----

  async function loadList() {
    setModeLabel();
    setStatus("Загрузка...");
    clearComments();

    try {
      let data;

      if (mode === "roots") {
        data = await getRoots(limit, offset);
      } else {
        data = await searchComments(searchText, limit, offset, searchSort, searchOrder);
      }

      const items = data.items || [];
      const box = $("comments");

      if (!box) {
        console.error("В HTML нет блока #comments");
        return;
      }

      if (items.length === 0) {
        setStatus(mode === "roots" ? "Комментариев пока нет" : "Ничего не найдено");
        return;
      }

      for (const item of items) {
        box.appendChild(renderComment(item, 0));
      }

      setStatus(`Показано ${items.length}. offset=${offset}`);
    } catch (err) {
      setStatus("Ошибка: " + err.message);
    }
  }

  // ----- UI wiring -----

  // создание корневого
  $("btnCreateRoot").onclick = async () => {
    const ta = $("newRootBody");
    const body = (ta?.value || "").trim();
    if (!body) return alert("Пустой комментарий");

    try {
      await createComment(body, null);
      if (ta) ta.value = "";

      mode = "roots";
      offset = 0;
      await loadList();
    } catch (err) {
      alert("Ошибка: " + err.message);
    }
  };

  // пагинация
  $("btnPrev").onclick = async () => {
    offset = Math.max(0, offset - limit);
    await loadList();
  };

  $("btnNext").onclick = async () => {
    offset += limit;
    await loadList();
  };

  // dropdown: sort/order
  $("searchSort").addEventListener("change", async () => {
    searchSort = $("searchSort").value;
    offset = 0;
    if (mode === "search" && searchText) await loadList();
  });

  $("searchOrder").addEventListener("change", async () => {
    searchOrder = $("searchOrder").value;
    offset = 0;
    if (mode === "search" && searchText) await loadList();
  });

  // поиск
  $("btnSearch").onclick = async () => {
    const q = ($("searchQ").value || "").trim();
    if (!q) return alert("Введите текст для поиска");

    // обновим sort/order из селектов (на всякий)
    searchSort = $("searchSort").value;
    searchOrder = $("searchOrder").value;

    mode = "search";
    searchText = q;
    offset = 0;
    await loadList();
  };

  $("btnClear").onclick = async () => {
    mode = "roots";
    searchText = "";
    offset = 0;
    $("searchQ").value = "";
    await loadList();
  };

  // enter в поле поиска
  $("searchQ").addEventListener("keydown", async (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      $("btnSearch").click();
    }
  });

  // init
  loadList();
})();
