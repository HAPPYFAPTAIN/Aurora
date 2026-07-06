const $ = s => document.querySelector(s);
const $$ = s => document.querySelectorAll(s);

const KNOWN_FIELDS = ["来源论文","来源","素材类型","主题分类","核心内容","关键引文","关键细节","可应用场景","与现有设定的关系","对照设定","提炼原则","编号规则","来源范围","简介","摘要","标签","关键词","重要度","基本信息","性格特征","背景故事","核心目标","长期关系","能力边界","环境描述","功能与规则","关联角色","发生事件","核心利益","内部结构","立场与关系","资源与能力","规则内容","适用范围","代价与后果","例外情况","概念定义","核心内涵","分析角度","核心发现","原文依据","分析结论"];

const App = {
    currentQuery: '',
    debounceTimer: null,

    async init() {
        this.bindTabs();
        this.bindSearch();
        this.bindImport();
        this.bindGenerate();
        this.bindRebuild();
        await this.loadStats();
        await this.loadTemplates();
        await this.loadImports();
        $('#searchInput').focus();
    },

    bindTabs() {
        $$('.tab').forEach(tab => {
            tab.onclick = () => {
                $$('.tab').forEach(t => t.classList.remove('active'));
                $$('.panel').forEach(p => p.classList.remove('active'));
                tab.classList.add('active');
                $('#panel-' + tab.dataset.tab)?.classList.add('active');
                if (tab.dataset.tab === 'import') this.loadImports();
            };
        });
    },

    bindSearch() {
        const input = $('#searchInput');
        input.oninput = () => {
            clearTimeout(this.debounceTimer);
            this.debounceTimer = setTimeout(() => this.doSearch(), 250);
        };
        input.onkeydown = e => { if (e.key === 'Enter') this.doSearch(); };
        $('#searchBtn').onclick = () => this.doSearch();
        $('#typeFilter').onchange = () => this.doSearch();
        $('#resetBtn').onclick = () => {
            input.value = '';
            $('#typeFilter').value = '';
            $('#results').innerHTML = '';
            $('#workspaceResults').innerHTML = '';
            $('#resultsInfo').textContent = '';
            $('#emptyHint').style.display = 'block';
        };
    },

    async loadStats() {
        try {
            const stats = await (await fetch('/api/stats')).json();
            $('#statsBadge').textContent = stats.total + ' 张卡片';
            const sel = $('#typeFilter');
            sel.innerHTML = '<option value="">全部类型</option>';
            Object.keys(stats.byType || {}).sort().forEach(t => {
                sel.innerHTML += `<option value="${this.esc(t)}">${this.esc(t)} (${stats.byType[t]})</option>`;
            });
        } catch(e) { $('#statsBadge').textContent = '连接失败'; }
    },

    async doSearch() {
        const query = $('#searchInput').value.trim();
        const type = $('#typeFilter').value;
        const searchWS = $('#searchWorkspace').checked;
        this.currentQuery = query;

        if (!query && !type) {
            $('#results').innerHTML = '';
            $('#workspaceResults').innerHTML = '';
            $('#resultsInfo').textContent = '';
            $('#emptyHint').style.display = 'block';
            return;
        }
        $('#emptyHint').style.display = 'none';
        $('#resultsInfo').textContent = '搜索中...';
        $('#results').innerHTML = '';
        $('#workspaceResults').innerHTML = '';

        try {
            const params = new URLSearchParams();
            if (query) params.set('q', query);
            if (type) params.set('type', type);
            params.set('limit', '50');
            const data = await (await fetch('/api/search?' + params)).json();
            this.renderResults(data);

            // 同时搜索工作区文件
            if (query && searchWS) {
                this.searchWorkspace(query);
            }
        } catch(e) {
            $('#resultsInfo').textContent = '搜索失败';
        }
    },

    renderResults(data) {
        if (!data.results || data.results.length === 0) {
            $('#resultsInfo').textContent = '资料库中无匹配结果';
            return;
        }
        $('#resultsInfo').textContent = `资料库找到 ${data.total} 条结果`;
        const terms = this.currentQuery.split(/\s+/).filter(Boolean);
        $('#results').innerHTML = data.results.map(r => this.renderCard(r, terms)).join('');
        this.bindCardEvents();
    },

    renderCard(result, terms) {
        const c = result.card;
        const contentHtml = this.renderMarkdown(c.content || '', terms);
        const briefHtml = c.brief ? `<div class="card-brief">${this.applyHighlight(this.esc(c.brief), terms)}</div>` : '';
        const keywordsHtml = (c.keywords && c.keywords.length) ?
            `<div class="card-keywords">${c.keywords.map(k => `<span class="card-keyword">${this.esc(k)}</span>`).join('')}</div>` : '';
        const score = result.score ? `<span class="card-score">评分: ${result.score.toFixed(1)}</span>` : '';
        const matches = (result.matches && result.matches.length) ? `<span class="card-score">命中: ${result.matches.map(m => this.esc(m)).join(', ')}</span>` : '';

        return `
            <div class="card" data-id="${this.esc(c.id)}">
                <div class="card-header" onclick="App.toggleCard(this)">
                    ${c.typeLabel ? `<span class="card-type">${this.esc(c.typeLabel)}</span>` : `<span class="card-type">${this.esc(c.type)}</span>`}
                    ${c.importance ? `<span class="card-importance">${this.esc(c.importance)}</span>` : ''}
                    <span class="card-title">${this.applyHighlight(this.esc(c.name), terms)}</span>
                    ${score} ${matches}
                    <span class="card-expand-icon">▶</span>
                </div>
                <div class="card-body">
                    ${briefHtml}
                    <div class="card-content">${contentHtml}</div>
                    ${keywordsHtml}
                    <div class="card-actions">
                        <button onclick="App.deleteCard('${this.esc(c.id)}')">删除</button>
                    </div>
                </div>
            </div>
        `;
    },

    toggleCard(header) {
        const card = header.parentElement;
        card.classList.toggle('expanded');
    },

    async deleteCard(id) {
        if (!confirm('确认从资料库删除此卡片？')) return;
        try {
            await fetch('/api/lore/delete', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({id})
            });
            await this.loadStats();
            this.doSearch();
        } catch(e) { alert('删除失败'); }
    },

    async searchWorkspace(query) {
        try {
            const res = await fetch(`/api/workspace-search?q=${encodeURIComponent(query)}`);
            const data = await res.json();
            if (data.results && data.results.length > 0) {
                const terms = query.split(/\s+/).filter(Boolean);
                let html = `<div class="section-title">章节/设定文件 (${data.results.length})</div>`;
                html += data.results.slice(0, 20).map(r => `
                    <div class="ws-result">
                        <div class="ws-file">${this.esc(r.path || '')}:${r.line || ''}</div>
                        <div class="ws-preview">${this.applyHighlight(this.esc(r.preview || ''), terms)}</div>
                    </div>
                `).join('');
                $('#workspaceResults').innerHTML = html;
            }
        } catch(e) { /* ignore */ }
    },

    renderMarkdown(text, terms) {
        const escaped = this.esc(text);
        const lines = escaped.split('\n');
        const html = [];
        let inList = false;

        for (let line of lines) {
            const trimmed = line.trim();
            if (trimmed === '') {
                if (inList) { html.push('</ul>'); inList = false; }
                continue;
            }
            if (trimmed.startsWith('&gt; ')) {
                if (inList) { html.push('</ul>'); inList = false; }
                html.push(`<blockquote>${this.parseInline(trimmed.slice(5), terms)}</blockquote>`);
                continue;
            }
            if (trimmed.startsWith('### ') || trimmed.startsWith('## ')) {
                if (inList) { html.push('</ul>'); inList = false; }
                const level = trimmed.startsWith('### ') ? 4 : 3;
                html.push(`<h${level}>${this.applyHighlight(trimmed.replace(/^#{2,4}\s+/, ''), terms)}</h${level}>`);
                continue;
            }
            if (/^---+$/.test(trimmed)) {
                if (inList) { html.push('</ul>'); inList = false; }
                html.push('<hr>');
                continue;
            }
            const listMatch = line.match(/^(\s*)-\s+(.*)$/);
            if (listMatch) {
                if (!inList) { html.push('<ul>'); inList = true; }
                html.push(`<li>${this.parseListItem(listMatch[2], terms)}</li>`);
                continue;
            }
            if (inList) { html.push('</ul>'); inList = false; }
            html.push(`<p>${this.parseInline(trimmed, terms)}</p>`);
        }
        if (inList) html.push('</ul>');
        return html.join('');
    },

    parseListItem(content, terms) {
        const m = content.match(/^\*\*(.+?)\*\*\s*[:：]/);
        if (m) {
            const field = m[1];
            const rest = content.slice(m[0].length);
            const isKnown = KNOWN_FIELDS.some(f => field.includes(f));
            return `<span class="${isKnown ? 'field-name' : ''}"><strong>${this.applyHighlight(field, terms)}</strong></span>：${this.parseInline(rest, terms)}`;
        }
        return this.parseInline(content, terms);
    },

    parseInline(text, terms) {
        let result = text.replace(/\*\*(.+?)\*\*/g, (m, p1) => `<strong>${this.applyHighlight(p1, terms)}</strong>`);
        result = result.replace(/`(.+?)`/g, '<code>$1</code>');
        return this.applyHighlight(result, terms);
    },

    applyHighlight(html, terms) {
        if (!terms || !terms.length) return html;
        for (const term of terms) {
            if (!term) continue;
            const parts = html.split(/(<[^>]+>)/g);
            for (let i = 0; i < parts.length; i++) {
                if (parts[i].startsWith('<')) continue;
                parts[i] = parts[i].replace(new RegExp(this.regexEscape(this.esc(term)), 'gi'), m => `<mark>${m}</mark>`);
            }
            html = parts.join('');
        }
        return html;
    },

    esc(s) { const d = document.createElement('div'); d.textContent = s || ''; return d.innerHTML; },
    regexEscape(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); },

    bindCardEvents() {},

    // 导入
    bindImport() {
        const zone = $('#uploadZone');
        const input = $('#fileInput');
        zone.onclick = () => input.click();
        input.onchange = () => { if (input.files.length) this.uploadFile(input.files[0]); input.value = ''; };
        zone.ondragover = e => { e.preventDefault(); zone.classList.add('dragover'); };
        zone.ondragleave = () => zone.classList.remove('dragover');
        zone.ondrop = e => {
            e.preventDefault();
            zone.classList.remove('dragover');
            if (e.dataTransfer.files.length) this.uploadFile(e.dataTransfer.files[0]);
        };
    },

    async uploadFile(file) {
        const fd = new FormData();
        fd.append('file', file);
        try {
            const res = await fetch('/api/import', {method: 'POST', body: fd});
            const data = await res.json();
            if (data.error) { alert('上传失败: ' + data.error); return; }
            alert(`上传成功: ${data.filename}`);
            await this.loadImports();
        } catch(e) { alert('上传失败'); }
    },

    async loadImports() {
        try {
            const data = await (await fetch('/api/imports')).json();
            const files = data.files || [];
            const sel = $('#importSelect');
            sel.innerHTML = '<option value="">-- 手动粘贴 --</option>';
            files.forEach(f => { sel.innerHTML += `<option value="${this.esc(f.name)}">${this.esc(f.name)}</option>`; });

            const list = $('#importsList');
            if (!files.length) { list.innerHTML = '<div class="empty-hint">暂无导入文件</div>'; return; }
            list.innerHTML = files.map(f => `
                <div class="file-item">
                    <span class="file-name">${this.esc(f.name)}</span>
                    <span class="file-size">${this.formatSize(f.size)}</span>
                    <span class="file-time">${this.esc(f.time)}</span>
                    <button class="btn-icon" data-file="${this.esc(f.name)}">生成</button>
                    <button class="btn-icon danger" data-del="${this.esc(f.name)}">删除</button>
                </div>
            `).join('');
            list.querySelectorAll('[data-file]').forEach(b => b.onclick = () => this.loadImportContent(b.dataset.file));
            list.querySelectorAll('[data-del]').forEach(b => b.onclick = async () => {
                if (!confirm('删除?')) return;
                await fetch('/api/delete-import', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({path:b.dataset.del})});
                this.loadImports();
            });
        } catch(e) { $('#importsList').innerHTML = '<div class="empty-hint">加载失败</div>'; }
    },

    async loadImportContent(name) {
        try {
            // 通过搜索 API 获取导入文件内容
            const res = await fetch('/api/search?type=导入资料&limit=500');
            const data = await res.json();
            const found = (data.results || []).find(r => r.card.name === name);
            if (found) {
                $('#genTextInput').value = found.card.content || '';
                $$('.tab').forEach(t => t.classList.remove('active'));
                $$('.panel').forEach(p => p.classList.remove('active'));
                document.querySelector('[data-tab="generate"]').classList.add('active');
                $('#panel-generate').classList.add('active');
            }
        } catch(e) {}
    },

    // 生成
    bindGenerate() {
        $('#importSelect').onchange = e => { if (e.target.value) this.loadImportContent(e.target.value); };
        $('#generateBtn').onclick = () => this.generate();
    },

    async loadTemplates() {
        try {
            const data = await (await fetch('/api/templates')).json();
            const sel = $('#templateSelect');
            sel.innerHTML = '';
            (data.templates || []).forEach(t => {
                sel.innerHTML += `<option value="${t.id}">${t.name} - ${t.desc}</option>`;
            });
        } catch(e) {}
    },

    async generate() {
        const text = $('#genTextInput').value.trim();
        if (!text) { alert('请输入文本'); return; }
        const btn = $('#generateBtn');
        btn.disabled = true;
        btn.textContent = '生成中...';
        $('#genResult').innerHTML = '<div class="gen-loading"><div class="spinner"></div><p>AI 正在分析并写入资料库...</p></div>';
        try {
            const res = await fetch('/api/generate', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    text,
                    templateId: $('#templateSelect').value,
                    customPrompt: $('#customPromptInput').value.trim()
                })
            });
            const data = await res.json();
            this.renderGenResult(data);
            if (data.success) {
                // 刷新索引和统计
                await fetch('/api/rebuild', {method: 'POST'});
                await this.loadStats();
            }
        } catch(e) {
            $('#genResult').innerHTML = `<div class="gen-error">请求失败: ${this.esc(e.message)}</div>`;
        } finally {
            btn.disabled = false;
            btn.textContent = '生成卡片并写入资料库';
        }
    },

    renderGenResult(data) {
        const el = $('#genResult');
        if (!data.success) {
            el.innerHTML = `<div class="gen-error">${this.esc(data.error || '生成失败')}</div>` +
                (data.rawResponse ? `<details><summary>原始响应</summary><pre>${this.esc(data.rawResponse)}</pre></details>` : '');
            return;
        }
        let html = `<div class="gen-success-hint">成功生成 ${data.cards.length} 张卡片并写入资料库</div>`;
        data.cards.forEach(c => {
            html += `
                <div class="gen-card-item">
                    <div class="gc-header">
                        <span class="gc-type">${this.esc(c.type)}</span>
                        <span class="gc-title">${this.esc(c.name)}</span>
                    </div>
                    ${c.brief ? `<div class="gc-brief">${this.esc(c.brief)}</div>` : ''}
                    <div class="gc-content">${this.renderMarkdown(c.content || '', [])}</div>
                    ${c.keywords && c.keywords.length ? `<div class="card-keywords">${c.keywords.map(k => `<span class="card-keyword">${this.esc(k)}</span>`).join('')}</div>` : ''}
                </div>
            `;
        });
        el.innerHTML = html;
    },

    // 重建索引
    bindRebuild() {
        $('#rebuildBtn').onclick = async () => {
            $('#rebuildBtn').textContent = '刷新中...';
            try {
                const data = await (await fetch('/api/rebuild', {method: 'POST'})).json();
                if (data.error) { alert('失败: ' + data.error); }
                else { await this.loadStats(); }
            } catch(e) { alert('刷新失败'); }
            $('#rebuildBtn').textContent = '刷新索引';
        };
    },

    formatSize(b) {
        if (b < 1024) return b + ' B';
        if (b < 1048576) return (b / 1024).toFixed(1) + ' KB';
        return (b / 1048576).toFixed(1) + ' MB';
    }
};

document.addEventListener('DOMContentLoaded', () => App.init());
