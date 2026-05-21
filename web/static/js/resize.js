'use strict';

const Resize = {
  MIN_SIDEBAR_WIDTH: 150,
  MAX_SIDEBAR_WIDTH: 600,
  MIN_COMMIT_HEADER_WIDTH: 200,
  MAX_COMMIT_HEADER_WIDTH: 800,
  MIN_REVIEW_SUMMARY_HEIGHT: 36,
  MAX_REVIEW_SUMMARY_HEIGHT: 600,

  init() {
    this.attachDrag(
      document.getElementById('resize-handle'),
      document.getElementById('sidebar'),
      this.MIN_SIDEBAR_WIDTH,
      this.MAX_SIDEBAR_WIDTH,
    );
    this.attachSubmitBarDrag(
      document.getElementById('submit-bar-resize-handle'),
      document.getElementById('review-summary'),
      document.getElementById('submit-bar'),
    );
  },

  attachCommitHeaderDrag(handle, header) {
    this.attachDrag(handle, header, this.MIN_COMMIT_HEADER_WIDTH, this.MAX_COMMIT_HEADER_WIDTH);
  },

  attachDrag(handle, target, minWidth, maxWidth) {
    let startX;
    let startWidth;

    const onMouseMove = (e) => {
      const delta = e.clientX - startX;
      const newWidth = Math.min(maxWidth, Math.max(minWidth, startWidth + delta));
      target.style.width = newWidth + 'px';
    };

    const onMouseUp = () => {
      handle.classList.remove('dragging');
      document.body.classList.remove('resizing');
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };

    handle.addEventListener('mousedown', (e) => {
      e.preventDefault();
      startX = e.clientX;
      startWidth = target.getBoundingClientRect().width;
      handle.classList.add('dragging');
      document.body.classList.add('resizing');
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    });
  },

  attachSubmitBarDrag(handle, textarea, submitBar) {
    let startY;
    let startHeight;

    const onMouseMove = (e) => {
      const delta = startY - e.clientY;
      const newHeight = Math.min(
        this.MAX_REVIEW_SUMMARY_HEIGHT,
        Math.max(this.MIN_REVIEW_SUMMARY_HEIGHT, startHeight + delta),
      );
      textarea.style.height = newHeight + 'px';
      document.documentElement.style.setProperty(
        '--submit-bar-height',
        submitBar.offsetHeight + 'px',
      );
    };

    const onMouseUp = () => {
      handle.classList.remove('dragging');
      document.body.classList.remove('resizing-vertical');
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };

    handle.addEventListener('mousedown', (e) => {
      e.preventDefault();
      startY = e.clientY;
      startHeight = textarea.getBoundingClientRect().height;
      handle.classList.add('dragging');
      document.body.classList.add('resizing-vertical');
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    });
  },
};
