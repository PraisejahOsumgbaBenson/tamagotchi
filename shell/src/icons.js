// The status icons for the P1 display.
//
// On the original hardware these eight are separate segments of the glass
// rather than pixels in the matrix. Four sit across the top of the screen
// and four across the bottom, and they are always faintly visible, the way
// a printed segment on a liquid crystal display always is.
//
// The artwork is sixteen by sixteen and is stored as text so it can be read
// and edited. A '#' is a pixel of the segment.

const ICON_ART = {
  // Cutlery. The menu entry that feeds the pet.
  feed: [
    "................",
    "..#..#..#..###..",
    "..#..#..#..###..",
    "..#..#..#...#...",
    "..#..#..#...#...",
    "..######....#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "....#.......#...",
    "................",
  ],

  // A light bulb. The menu entry that switches the backlight.
  light: [
    "........#.......",
    "..#.....#.....#.",
    "...#....#....#..",
    "....#...#...#...",
    ".....########...",
    "....##########..",
    "...###########..",
    "...###########..",
    "....##########..",
    ".....########...",
    "......######....",
    "......######....",
    ".....########...",
    ".....##....##...",
    ".....##....##...",
    "................",
  ],

  // A bat and a ball. The menu entry that starts the game.
  play: [
    "................",
    "..........###...",
    ".........####...",
    "........####....",
    ".......####.....",
    "......####......",
    ".....####.......",
    "....####........",
    "...####.........",
    "..###...........",
    ".##.............",
    "................",
    "........####....",
    ".......######...",
    ".......######...",
    "........####....",
  ],

  // A syringe. The menu entry that gives the medicine.
  medicine: [
    "................",
    "..#####.........",
    "....#...........",
    "...#............",
    "..####..........",
    "..#..#..........",
    "..#..#..........",
    "..####..........",
    "....#...........",
    ".....#..........",
    "......#.........",
    ".......#........",
    "........#.......",
    ".........#......",
    "..........#.....",
    "................",
  ],

  // The duck. The menu entry that cleans up after the pet.
  duck: [
    "................",
    ".....#####......",
    "....#######.....",
    "....##..###.....",
    "###########.....",
    "...########.....",
    "..###########...",
    ".#############..",
    ".############...",
    ".############...",
    "..###########...",
    "..###########...",
    "...#########....",
    "....#######.....",
    "................",
    "................",
  ],

  // A dial. The menu entry that shows the health meter.
  meter: [
    "................",
    "................",
    ".....######.....",
    "...##########...",
    "..############..",
    "..##........##..",
    "..#..........#..",
    "..#....##....#..",
    "..#....##....#..",
    "..#....##....#..",
    "..#..........#..",
    "..##........##..",
    "..############..",
    "...##########...",
    ".....######.....",
    "................",
  ],

  // A face that has been told off. The menu entry that disciplines the pet.
  discipline: [
    "................",
    ".....######.....",
    "...##......##...",
    "..#..........#..",
    ".#..###..###..#.",
    ".#..##....##..#.",
    ".#..##....##..#.",
    ".#............#.",
    ".#....####....#.",
    ".#...##..##...#.",
    ".#..#......#..#.",
    ".#............#.",
    "..#..........#..",
    "...##......##...",
    ".....######.....",
    "................",
  ],

  // An exclamation mark. This one is not a menu entry. It lights up when the
  // pet wants something, and it sits in the bottom right corner.
  attention: [
    "................",
    "......####......",
    "......####......",
    "......####......",
    "......####......",
    "......####......",
    "......####......",
    "......####......",
    "......####......",
    "................",
    "................",
    "......####......",
    "......####......",
    "......####......",
    "................",
    "................",
  ],
};

// ICON_NAMES are the icons in the order the hardware wires them: the first
// four across the top of the glass, the last four across the bottom.
const ICON_NAMES = [
  "feed",
  "light",
  "play",
  "medicine",
  "duck",
  "meter",
  "discipline",
  "attention",
];

const ICON_SIZE = 16;

// iconMask turns one piece of artwork into an array of row bitmasks, with
// the leftmost pixel of a row in the least significant bit.
function iconMask(rows) {
  return rows.map((row) => {
    let bits = 0;
    for (let x = 0; x < row.length; x++) {
      if (row[x] === "#") {
        bits |= 1 << x;
      }
    }
    return bits;
  });
}

const ICONS = {};
for (const [name, rows] of Object.entries(ICON_ART)) {
  ICONS[name] = iconMask(rows);
}

window.TamaIcons = {
  names: ICON_NAMES,
  size: ICON_SIZE,
  bitmaps: ICONS,
};
