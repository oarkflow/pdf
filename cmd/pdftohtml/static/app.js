/* =========================================
   PDF→HTML Converter — Application Logic
   ========================================= */

(() => {
  'use strict';

  // ---- State ----
  const state = {
    files: [],        // [{id, filename, pages, metadata, status, element}]
    activeId: null,   // currently selected file id
    mode: 'reflowed',
    htmlSource: '',
  };

  // ---- DOM Refs ----
  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => document.querySelectorAll(sel);

  const dom = {
    uploadZone:      $('#uploadZone'),
    fileInput:       $('#fileInput'),
    uploadIdle:      $('#uploadIdle'),
    uploadHover:     $('#uploadHover'),
    passwordRow:     $('#passwordRow'),
    passwordInput:   $('#passwordInput'),
    fileListSection: $('#fileListSection'),
    fileList:        $('#fileList'),
    clearAllBtn:     $('#clearAllBtn'),
    optionsSection:  $('#optionsSection'),
    convertSection:  $('#convertSection'),
    convertBtn:      $('#convertBtn'),
    progressContainer: $('#progressContainer'),
    progressFill:    $('#progressFill'),
    progressText:    $('#progressText'),
    modeReflowed:    $('#modeReflowed'),
    modePositioned:  $('#modePositioned'),
    pageRange:       $('#pageRange'),
    extractImages:   $('#extractImages'),
    detectTables:    $('#detectTables'),
    emptyState:      $('#emptyState'),
    resultView:      $('#resultView'),
    tabPreview:      $('#tabPreview'),
    tabSource:       $('#tabSource'),
    previewPanel:    $('#previewPanel'),
    sourcePanel:     $('#sourcePanel'),
    previewFrame:    $('#previewFrame'),
    sourceCode:      $('#sourceCode'),
    copyBtn:         $('#copyBtn'),
    downloadBtn:     $('#downloadBtn'),
    metadataBar:     $('#metadataBar'),
    metaItems:       $('#metaItems'),
    toastContainer:  $('#toastContainer'),
  };

  // ---- Init ----
  function init() {
    bindUpload();
    bindOptions();
    bindConvert();
    bindToolbar();
    bindClearAll();
  }

  // ---- Upload ----
  function bindUpload() {
    const zone = dom.uploadZone;

    zone.addEventListener('click', () => dom.fileInput.click());
    dom.fileInput.addEventListener('change', (e) => {
      if (e.target.files.length) handleFiles(e.target.files);
      e.target.value = '';
    });

    zone.addEventListener('dragover', (e) => {
      e.preventDefault();
      zone.classList.add('dragging');
    });

    zone.addEventListener('dragleave', (e) => {
      e.preventDefault();
      zone.classList.remove('dragging');
    });

    zone.addEventListener('drop', (e) => {
      e.preventDefault();
      zone.classList.remove('dragging');
      if (e.dataTransfer.files.length) handleFiles(e.dataTransfer.files);
    });
  }

  async function handleFiles(fileList) {
    for (const file of fileList) {
      if (file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf')) {
        toast('Only PDF files are accepted', 'error');
        continue;
      }
      await uploadFile(file);
    }
  }

  async function uploadFile(file) {
    const formData = new FormData();
    formData.append('file', file);

    const password = dom.passwordInput.value;
    if (password) formData.append('password', password);

    try {
      const res = await fetch('/api/upload', { method: 'POST', body: formData });
      const data = await res.json();

      if (data.error) {
        toast(data.error, 'error');
        return;
      }

      const fileEntry = {
        id: data.id,
        filename: data.filename,
        pages: data.pages,
        metadata: data.metadata || {},
        status: 'uploaded',
      };

      state.files.push(fileEntry);
      renderFileList();
      selectFile(fileEntry.id);
      showSections();

      toast(`${data.filename} uploaded — ${data.pages} page${data.pages > 1 ? 's' : ''}`, 'success');
    } catch (err) {
      toast('Upload failed: ' + err.message, 'error');
    }
  }

  // ---- File List ----
  function renderFileList() {
    dom.fileList.innerHTML = '';
    state.files.forEach((f, i) => {
      const li = document.createElement('li');
      li.className = `file-item${f.id === state.activeId ? ' active' : ''}`;
      li.style.animationDelay = `${i * 0.05}s`;
      li.innerHTML = `
        <div class="file-icon">PDF</div>
        <div class="file-info">
          <div class="file-name">${escapeHtml(f.filename)}</div>
          <div class="file-meta">${f.pages} page${f.pages > 1 ? 's' : ''}${f.status === 'done' ? ' · Converted' : ''}</div>
        </div>
        <div class="file-status ${f.status}"></div>
        <button class="file-remove btn-ghost" data-id="${f.id}" title="Remove">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      `;

      li.addEventListener('click', (e) => {
        if (e.target.closest('.file-remove')) return;
        selectFile(f.id);
      });

      li.querySelector('.file-remove').addEventListener('click', () => removeFile(f.id));

      dom.fileList.appendChild(li);
    });
  }

  function selectFile(id) {
    state.activeId = id;
    renderFileList();

    const file = state.files.find(f => f.id === id);
    if (!file) return;

    renderMetadata(file);

    if (file.status === 'done') {
      showResult(id);
    }
  }

  function removeFile(id) {
    fetch(`/api/job/${id}`, { method: 'DELETE' }).catch(() => {});
    state.files = state.files.filter(f => f.id !== id);

    if (state.activeId === id) {
      state.activeId = state.files.length ? state.files[0].id : null;
    }

    renderFileList();

    if (!state.files.length) {
      hideSections();
    } else if (state.activeId) {
      selectFile(state.activeId);
    }
  }

  function bindClearAll() {
    dom.clearAllBtn.addEventListener('click', () => {
      state.files.forEach(f => {
        fetch(`/api/job/${f.id}`, { method: 'DELETE' }).catch(() => {});
      });
      state.files = [];
      state.activeId = null;
      renderFileList();
      hideSections();
    });
  }

  // ---- Sections Visibility ----
  function showSections() {
    dom.fileListSection.style.display = '';
    dom.optionsSection.style.display = '';
    dom.convertSection.style.display = '';
  }

  function hideSections() {
    dom.fileListSection.style.display = 'none';
    dom.optionsSection.style.display = 'none';
    dom.convertSection.style.display = 'none';
    dom.emptyState.style.display = '';
    dom.resultView.style.display = 'none';
  }

  // ---- Options ----
  function bindOptions() {
    dom.modeReflowed.addEventListener('click', () => setMode('reflowed'));
    dom.modePositioned.addEventListener('click', () => setMode('positioned'));
  }

  function setMode(mode) {
    state.mode = mode;
    dom.modeReflowed.classList.toggle('active', mode === 'reflowed');
    dom.modePositioned.classList.toggle('active', mode === 'positioned');
  }

  function parsePageRange(input, maxPages) {
    if (!input.trim()) return null; // all pages
    const pages = new Set();
    const parts = input.split(',');
    for (const part of parts) {
      const trimmed = part.trim();
      if (trimmed.includes('-')) {
        const [startStr, endStr] = trimmed.split('-');
        const start = parseInt(startStr, 10);
        const end = parseInt(endStr, 10);
        if (!isNaN(start) && !isNaN(end)) {
          for (let i = Math.max(0, start - 1); i < Math.min(maxPages, end); i++) {
            pages.add(i);
          }
        }
      } else {
        const n = parseInt(trimmed, 10);
        if (!isNaN(n) && n >= 1 && n <= maxPages) {
          pages.add(n - 1);
        }
      }
    }
    return pages.size > 0 ? [...pages].sort((a, b) => a - b) : null;
  }

  // ---- Convert ----
  function bindConvert() {
    dom.convertBtn.addEventListener('click', startConversion);
  }

  async function startConversion() {
    const id = state.activeId;
    if (!id) return;

    const file = state.files.find(f => f.id === id);
    if (!file) return;

    const pages = parsePageRange(dom.pageRange.value, file.pages);

    dom.convertBtn.disabled = true;
    dom.progressContainer.style.display = '';
    dom.progressFill.style.width = '0%';
    dom.progressText.textContent = 'Starting conversion…';

    file.status = 'converting';
    renderFileList();

    try {
      const res = await fetch('/api/convert', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id,
          pages,
          mode: state.mode,
          extractImages: dom.extractImages.checked,
          detectTables: dom.detectTables.checked,
        }),
      });

      const data = await res.json();
      if (data.error) throw new Error(data.error);

      // Poll status.
      await pollStatus(id);

    } catch (err) {
      toast('Conversion failed: ' + err.message, 'error');
      file.status = 'error';
      renderFileList();
    } finally {
      dom.convertBtn.disabled = false;
    }
  }

  async function pollStatus(id) {
    const file = state.files.find(f => f.id === id);
    if (!file) return;

    while (true) {
      await sleep(400);
      try {
        const res = await fetch(`/api/status/${id}`);
        const data = await res.json();

        if (data.status === 'done') {
          file.status = 'done';
          renderFileList();
          dom.progressFill.style.width = '100%';
          dom.progressText.textContent = 'Done!';

          await sleep(500);
          dom.progressContainer.style.display = 'none';

          showResult(id);
          toast('Conversion complete!', 'success');
          return;
        }

        if (data.status === 'error') {
          file.status = 'error';
          renderFileList();
          dom.progressContainer.style.display = 'none';
          toast(data.error || 'Conversion failed', 'error');
          return;
        }

        // Update progress.
        if (data.total > 0) {
          const pct = Math.round((data.progress / data.total) * 100);
          dom.progressFill.style.width = pct + '%';
          dom.progressText.textContent = `Converting page ${data.progress} of ${data.total}…`;
        }
      } catch {
        break;
      }
    }
  }

  // ---- Result Display ----
  async function showResult(id) {
    dom.emptyState.style.display = 'none';
    dom.resultView.style.display = '';

    try {
      const res = await fetch(`/api/preview/${id}`);
      const html = await res.text();
      state.htmlSource = html;

      // Write to iframe.
      const frame = dom.previewFrame;
      const doc = frame.contentDocument || frame.contentWindow.document;
      doc.open();
      doc.write(html);
      doc.close();

      // Source tab.
      dom.sourceCode.textContent = html;

      // Show preview tab by default.
      switchTab('preview');
    } catch (err) {
      toast('Failed to load preview: ' + err.message, 'error');
    }
  }

  // ---- Toolbar ----
  function bindToolbar() {
    dom.tabPreview.addEventListener('click', () => switchTab('preview'));
    dom.tabSource.addEventListener('click', () => switchTab('source'));
    dom.copyBtn.addEventListener('click', copyHTML);
    dom.downloadBtn.addEventListener('click', downloadHTML);
  }

  function switchTab(tab) {
    dom.tabPreview.classList.toggle('active', tab === 'preview');
    dom.tabSource.classList.toggle('active', tab === 'source');
    dom.previewPanel.style.display = tab === 'preview' ? '' : 'none';
    dom.sourcePanel.style.display = tab === 'source' ? '' : 'none';
  }

  async function copyHTML() {
    if (!state.htmlSource) return;
    try {
      await navigator.clipboard.writeText(state.htmlSource);
      toast('HTML copied to clipboard', 'success');
    } catch {
      toast('Failed to copy', 'error');
    }
  }

  function downloadHTML() {
    if (!state.activeId) return;
    window.location.href = `/api/download/${state.activeId}`;
  }

  // ---- Metadata ----
  function renderMetadata(file) {
    const items = [];
    items.push({ key: 'File', value: file.filename });
    items.push({ key: 'Pages', value: file.pages });

    if (file.metadata) {
      if (file.metadata.Title)    items.push({ key: 'Title', value: file.metadata.Title });
      if (file.metadata.Author)   items.push({ key: 'Author', value: file.metadata.Author });
      if (file.metadata.Subject)  items.push({ key: 'Subject', value: file.metadata.Subject });
      if (file.metadata.Creator)  items.push({ key: 'Creator', value: file.metadata.Creator });
      if (file.metadata.Producer) items.push({ key: 'Producer', value: file.metadata.Producer });
    }

    dom.metaItems.innerHTML = items.map(({ key, value }) =>
      `<div class="meta-item">
        <span class="meta-key">${escapeHtml(key)}</span>
        <span class="meta-value">${escapeHtml(String(value))}</span>
      </div>`
    ).join('');
  }

  // ---- Toast ----
  function toast(message, type = 'info') {
    const el = document.createElement('div');
    el.className = `toast ${type}`;
    el.textContent = message;
    dom.toastContainer.appendChild(el);

    setTimeout(() => {
      el.classList.add('toast-exit');
      setTimeout(() => el.remove(), 300);
    }, 3500);
  }

  // ---- Helpers ----
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // ---- Boot ----
  init();
})();
