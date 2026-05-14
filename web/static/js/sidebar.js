'use strict';

const Sidebar = {
  mode: 'commits',
  commits: [],
  files: [],
  activeCommitSHA: null,

  async init() {
    const sidebar = document.getElementById('sidebar');
    sidebar.innerHTML = `
      <div class="sidebar-tabs">
        <button class="tab" data-tab="files">Files changed</button>
        <button class="tab active" data-tab="commits">Commits</button>
      </div>
      <div id="sidebar-content"></div>
    `;

    sidebar.querySelectorAll('.tab').forEach(tab => {
      tab.addEventListener('click', () => this.switchTab(tab.dataset.tab));
    });

    this.commits = await API.getCommits() || [];
    this.files = await API.getFiles() || [];
    this.render();
  },

  switchTab(mode) {
    this.mode = mode;
    document.querySelectorAll('.sidebar-tabs .tab').forEach(tab => {
      tab.classList.toggle('active', tab.dataset.tab === mode);
    });
    this.render();
  },

  render() {
    const container = document.getElementById('sidebar-content');
    if (this.mode === 'files') {
      this.renderFiles(container);
    } else {
      this.renderCommits(container);
    }
  },

  renderFiles(container) {
    if (this.files.length === 0) {
      container.innerHTML = '<p class="sidebar-empty">No files changed</p>';
      return;
    }

    const statusLabels = { A: 'added', M: 'modified', D: 'deleted', R: 'renamed' };

    const html = this.files.map(file => {
      const statusClass = (file.status || 'M').toLowerCase();
      const statusLabel = statusLabels[file.status] || file.status;
      return `
        <div class="sidebar-item file-item" data-path="${file.path}">
          <span class="file-status file-status-${statusClass}" title="${statusLabel}">${file.status}</span>
          <span class="file-name" title="${file.path}">${file.path}</span>
        </div>
      `;
    }).join('');

    container.innerHTML = html;

    container.querySelectorAll('.file-item').forEach(item => {
      item.addEventListener('click', () => {
        this.scrollToFile(item.dataset.path);
        this.setActiveItem(item);
      });
    });
  },

  renderCommits(container) {
    if (this.commits.length === 0) {
      container.innerHTML = '<p class="sidebar-empty">No commits found</p>';
      return;
    }

    const html = this.commits.map(commit => {
      const shortSHA = commit.sha.substring(0, 7);
      const isActive = commit.sha === this.activeCommitSHA;
      return `
        <div class="sidebar-item commit-item${isActive ? ' active' : ''}" data-sha="${commit.sha}">
          <span class="commit-sha">${shortSHA}</span>
          <span class="commit-subject">${this.escapeHTML(commit.subject)}</span>
          <span class="commit-meta">${commit.author} &middot; ${commit.date}</span>
        </div>
      `;
    }).join('');

    // Add "All changes" option at top
    const allActive = this.activeCommitSHA === null;
    container.innerHTML = `
      <div class="sidebar-item commit-item${allActive ? ' active' : ''}" data-sha="">
        <span class="commit-subject">All changes</span>
        <span class="commit-meta">Full diff between base and head</span>
      </div>
      ${html}
    `;

    container.querySelectorAll('.commit-item').forEach(item => {
      item.addEventListener('click', () => {
        const sha = item.dataset.sha || null;
        this.activeCommitSHA = sha;
        this.render();
        if (sha) {
          App.loadCommitDiff(sha);
        } else {
          App.loadFullDiff();
        }
      });
    });
  },

  scrollToFile(path) {
    const fileHeaders = document.querySelectorAll('.d2h-file-header');
    for (const header of fileHeaders) {
      const nameEl = header.querySelector('.d2h-file-name');
      if (nameEl && nameEl.textContent.trim() === path) {
        header.scrollIntoView({ behavior: 'smooth', block: 'start' });
        return;
      }
    }
  },

  setActiveItem(item) {
    document.querySelectorAll('.sidebar-item').forEach(el => el.classList.remove('active'));
    item.classList.add('active');
  },

  escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  },
};
