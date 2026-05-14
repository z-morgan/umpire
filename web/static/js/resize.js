'use strict';

const Resize = {
  MIN_SIDEBAR_WIDTH: 150,
  MAX_SIDEBAR_WIDTH: 600,
  MIN_COMMIT_HEADER_WIDTH: 200,
  MAX_COMMIT_HEADER_WIDTH: 800,

  init() {
    this.attachDrag(
      document.getElementById('resize-handle'),
      document.getElementById('sidebar'),
      this.MIN_SIDEBAR_WIDTH,
      this.MAX_SIDEBAR_WIDTH,
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
};
