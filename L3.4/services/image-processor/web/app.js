/* app.js — Image Processor UI logic */

(function () {
  'use strict';

  // ─── DOM refs ────────────────────────────────────────────────────────────────
  const form          = document.getElementById('upload-form');
  const fileInput     = document.getElementById('file-input');
  const modeSelect    = document.getElementById('mode-select');
  const widthInput    = document.getElementById('width-input');
  const heightInput   = document.getElementById('height-input');
  const wmText        = document.getElementById('wm-text');
  const resizeGroup   = document.getElementById('resize-group');
  const watermarkGroup= document.getElementById('watermark-group');
  const submitBtn     = document.getElementById('submit-btn');
  const statusEl      = document.getElementById('status');
  const resultEl      = document.getElementById('result');

  // ─── Show/hide conditional fields based on selected mode ─────────────────────
  modeSelect.addEventListener('change', updateFieldVisibility);

  function updateFieldVisibility() {
    const mode = modeSelect.value;
    resizeGroup.style.display    = (mode === 'resize' || mode === 'thumb') ? '' : 'none';
    watermarkGroup.style.display = mode === 'watermark' ? '' : 'none';
  }

  // ─── Form submit ─────────────────────────────────────────────────────────────
  form.addEventListener('submit', async function (e) {
    e.preventDefault();

    const file = fileInput.files[0];
    if (!file) {
      setStatus('Выберите файл', 'error');
      return;
    }

    setStatus('<span class="spinner"></span>Загружаем…', '');
    resultEl.innerHTML = '';
    setDisabled(true);

    const fd = new FormData();
    fd.append('file', file);
    fd.append('mode', modeSelect.value);

    const w = parseInt(widthInput.value, 10);
    const h = parseInt(heightInput.value, 10);
    if (!isNaN(w) && w > 0) fd.append('width',  String(w));
    if (!isNaN(h) && h > 0) fd.append('height', String(h));

    const wm = wmText.value.trim();
    if (wm) fd.append('watermark_text', wm);

    try {
      // 1) Upload
      const uploadResp = await fetch('/upload', { method: 'POST', body: fd });
      if (!uploadResp.ok) {
        const txt = await uploadResp.text();
        throw new Error('Ошибка загрузки: ' + txt);
      }
      const uploadData = await uploadResp.json();
      const id = uploadData.id;
      if (!id) throw new Error('Сервер не вернул id');

      setStatus('<span class="spinner"></span>Обрабатываем… (id: ' + id + ')', '');

      // 2) Poll until ready / failed
      await poll(id);
    } catch (err) {
      setStatus('❌ ' + err.message, 'error');
    } finally {
      setDisabled(false);
    }
  });

  // ─── Polling ─────────────────────────────────────────────────────────────────
  const POLL_INTERVAL_MS = 1000;
  const POLL_TIMEOUT_MS  = 120_000; // 2 min max

  async function poll(id) {
    const deadline = Date.now() + POLL_TIMEOUT_MS;

    while (Date.now() < deadline) {
      const resp = await fetch('/image/' + id);

      if (resp.status === 202) {
        // still processing
        setStatus('<span class="spinner"></span>В очереди… (id: ' + id + ')', '');
        await sleep(POLL_INTERVAL_MS);
        continue;
      }

      if (resp.status === 500) {
        // server returned {status:"failed"}
        throw new Error('Обработка завершилась ошибкой на сервере');
      }

      if (resp.ok) {
        // 200 — ready, stream is the image
        const blob = await resp.blob();
        const url  = URL.createObjectURL(blob);
        resultEl.innerHTML = '<img src="' + url + '" alt="Результат" />';
        setStatus('✅ Готово!', 'success');
        return;
      }

      // some other status
      const txt = await resp.text();
      throw new Error('Неожиданный ответ (' + resp.status + '): ' + txt);
    }

    throw new Error('Таймаут ожидания результата (2 мин)');
  }

  // ─── Helpers ─────────────────────────────────────────────────────────────────
  function setStatus(html, cls) {
    statusEl.innerHTML = html;
    statusEl.className = cls;
  }

  function setDisabled(val) {
    submitBtn.disabled = val;
    fileInput.disabled = val;
    modeSelect.disabled = val;
  }

  function sleep(ms) {
    return new Promise(function (resolve) { setTimeout(resolve, ms); });
  }

  // Init
  updateFieldVisibility();
})();
