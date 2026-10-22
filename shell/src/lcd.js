// The liquid crystal display.
//
// The P1 has a 32 by 16 matrix of pixels in the middle of the glass and
// eight status icons around it. The icons are separate segments of the
// glass rather than pixels in the matrix, which is why they are always
// faintly visible even when they are switched off.
//
// This file paints all of it onto one canvas.

(function () {
  // The size of the glass in CSS pixels.
  const GLASS_WIDTH = 148;
  const GLASS_HEIGHT = 150;

  // The matrix. It sits inside the glass with a margin.
  const MATRIX_X = 10;
  const MATRIX_Y = 43;
  const CELL = 4;
  const COLUMNS = 32;
  const ROWS = 16;

  // The status icons. Four across the top and four across the bottom.
  const ICON_SIZE = 32;
  const ICON_X = [4, 40, 76, 112];
  const ICON_Y_TOP = 9;
  const ICON_Y_BOTTOM = 115;

  // The marker that points at the icon the pet has highlighted.
  const MARK_TOP = 2;
  const MARK_BOTTOM = 108;
  const MARK_WIDTH = 9;
  const MARK_HEIGHT = 6;

  // The colour of a pixel that is on, and of a pixel that is off. A liquid
  // crystal panel does not go black, it goes a very slightly different
  // shade of the background.
  const INK = "#1d2233";
  const INK_OFF = "rgba(29, 34, 51, 0.08)";
  const INK_GHOST = "rgba(29, 34, 51, 0.2)";

  // iconBitmaps turns the artwork into one array of row masks per icon.
  const iconBitmaps = window.TamaIcons.bitmaps;
  const iconNames = window.TamaIcons.names;

  // draw paints the screen.
  //
  // rows holds one bitmask per row of the matrix, icons is a bitmask over
  // the eight status icons and sel marks the one that is highlighted.
  function draw(canvas, rows, icons, sel) {
    const ratio = window.devicePixelRatio || 1;

    const width = Math.round(GLASS_WIDTH * ratio);
    const height = Math.round(GLASS_HEIGHT * ratio);
    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width;
      canvas.height = height;
    }

    const ctx = canvas.getContext("2d");
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, width, height);

    drawMatrix(ctx, rows, ratio);
    drawIcons(ctx, icons, sel, ratio);
  }

  // drawMatrix paints the 32 by 16 grid of pixels.
  function drawMatrix(ctx, rows, ratio) {
    const cell = CELL * ratio;
    const gap = Math.max(1, Math.round(0.9 * ratio));
    const size = cell - gap;

    const left = Math.round(MATRIX_X * ratio);
    const top = Math.round(MATRIX_Y * ratio);

    for (let row = 0; row < ROWS; row++) {
      const bits = rows[row] >>> 0;
      for (let column = 0; column < COLUMNS; column++) {
        const on = (bits >>> column) & 1;
        ctx.fillStyle = on ? INK : INK_OFF;
        ctx.fillRect(left + column * cell, top + row * cell, size, size);
      }
    }
  }

  // drawIcons paints the eight status icons and the marker.
  function drawIcons(ctx, icons, sel, ratio) {
    const step = Math.round((ICON_SIZE * ratio) / 16);

    for (let index = 0; index < iconNames.length; index++) {
      const name = iconNames[index];
      const bitmap = iconBitmaps[name];
      if (!bitmap) {
        continue;
      }

      const topRow = index < 4;
      const x = Math.round(ICON_X[index % 4] * ratio);
      const y = Math.round((topRow ? ICON_Y_TOP : ICON_Y_BOTTOM) * ratio);

      const on = (icons >>> index) & 1;
      ctx.fillStyle = on ? INK : INK_GHOST;

      for (let row = 0; row < 16; row++) {
        const bits = bitmap[row] >>> 0;
        for (let column = 0; column < 16; column++) {
          if ((bits >>> column) & 1) {
            ctx.fillRect(x + column * step, y + row * step, step, step);
          }
        }
      }

      // The marker points at the icon the pet has highlighted.
      if (sel && (sel >>> index) & 1) {
        const centre = x + (ICON_SIZE * ratio) / 2;
        const markerY = Math.round((topRow ? MARK_TOP : MARK_BOTTOM) * ratio);
        const width = MARK_WIDTH * ratio;
        const height = MARK_HEIGHT * ratio;

        ctx.fillStyle = INK;
        ctx.beginPath();
        ctx.moveTo(centre - width / 2, markerY);
        ctx.lineTo(centre + width / 2, markerY);
        ctx.lineTo(centre, markerY + height);
        ctx.closePath();
        ctx.fill();
      }
    }
  }

  window.TamaLCD = { draw };
})();
