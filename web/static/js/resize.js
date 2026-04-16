'use strict';

const Resize = {
  MIN_SIDEBAR_WIDTH: 150,
  MAX_SIDEBAR_WIDTH: 600,

  init() {
    const handle = document.getElementById('resize-handle');
    const sidebar = document.getElementById('sidebar');

    let startX;
    let startWidth;

    const onMouseMove = (e) => {
      const delta = e.clientX - startX;
      const newWidth = Math.min(
        this.MAX_SIDEBAR_WIDTH,
        Math.max(this.MIN_SIDEBAR_WIDTH, startWidth + delta)
      );
      sidebar.style.width = newWidth + 'px';
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
      startWidth = sidebar.getBoundingClientRect().width;
      handle.classList.add('dragging');
      document.body.classList.add('resizing');
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    });
  },
};
