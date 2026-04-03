'use strict';

const App = {
  diffContainer: null,

  async init() {
    this.diffContainer = document.getElementById('diff-container');

    const info = await API.getInfo();
    document.getElementById('branch-info').textContent =
      `${info.base_ref}..${info.head_ref}`;

    await Promise.all([
      this.loadFullDiff(),
      Sidebar.init(),
    ]);
  },

  async loadFullDiff() {
    const diff = await API.getDiff();
    this.renderDiff(diff);
  },

  async loadCommitDiff(sha) {
    const diff = await API.getDiff(sha);
    this.renderDiff(diff);
  },

  renderDiff(diffString) {
    if (!diffString.trim()) {
      this.diffContainer.innerHTML = '<p class="empty-diff">No changes found.</p>';
      return;
    }

    this.diffContainer.innerHTML = '';
    const targetElement = document.createElement('div');
    this.diffContainer.appendChild(targetElement);

    const diff2htmlUi = new Diff2HtmlUI(targetElement, diffString, {
      drawFileList: false,
      matching: 'lines',
      outputFormat: 'line-by-line',
      highlight: true,
      fileListToggle: false,
    });
    diff2htmlUi.draw();
    diff2htmlUi.highlightCode();

    DiffView.reattachComments();
  },
};

document.addEventListener('DOMContentLoaded', () => App.init());
