// The main process.
//
// The shell is a small frameless window that sits on top of everything else
// on the desktop. It has no chrome, no menu bar and no dock icon, so the
// only thing on screen is the device itself.
//
// The window is bigger than the device so that the device can be dragged
// around by its edges, and the parts of the window that are not the device
// let the mouse through to whatever is underneath.

const { app, BrowserWindow, ipcMain, screen } = require("electron");
const fs = require("node:fs");
const path = require("node:path");

// DEVICE is the size of the Tamagotchi in CSS pixels.
//
// The original is 4.5 centimetres tall and about 3.4 wide, which gives a
// ratio of roughly three to four. These numbers keep that ratio and give a
// size that reads well on a laptop screen.
const DEVICE = { width: 276, height: 360 };

// PADDING is the room around the device inside the window. It is what the
// mouse grabs when the device is dragged around.
const PADDING = 40;

// POSITION_FILE remembers where the device was left on the desktop.
const POSITION_FILE = () => path.join(app.getPath("userData"), "position.json");

let win = null;

// readPosition returns the last saved position of the device, or nil if there
// is not one.
function readPosition() {
  try {
    const raw = fs.readFileSync(POSITION_FILE(), "utf8");
    const saved = JSON.parse(raw);
    if (Number.isFinite(saved.x) && Number.isFinite(saved.y)) {
      return saved;
    }
  } catch {
    // There is no saved position yet, which is the normal case.
  }
  return null;
}

// writePosition remembers where the device is on the desktop.
function writePosition() {
  if (!win) {
    return;
  }
  const [x, y] = win.getPosition();
  try {
    fs.mkdirSync(path.dirname(POSITION_FILE()), { recursive: true });
    fs.writeFileSync(
      POSITION_FILE(),
      JSON.stringify({ x, y }, null, 2) + "\n"
    );
  } catch (err) {
    console.error("tama: could not save the position", err);
  }
}

// defaultPosition puts the device in the bottom right corner of the screen,
// a little way in from the edge.
function defaultPosition() {
  const display = screen.getPrimaryDisplay();
  const area = display.workArea;
  return {
    x: area.x + area.width - DEVICE.width - PADDING - 24,
    y: area.y + area.height - DEVICE.height - PADDING - 24,
  };
}

// createWindow builds the floating window.
function createWindow() {
  const saved = readPosition() || defaultPosition();

  win = new BrowserWindow({
    x: saved.x,
    y: saved.y,
    width: DEVICE.width + PADDING * 2,
    height: DEVICE.height + PADDING * 2,
    frame: false,
    transparent: true,
    resizable: false,
    maximizable: false,
    minimizable: false,
    fullscreenable: false,
    hasShadow: false,
    skipTaskbar: true,
    acceptFirstMouse: true,
    backgroundColor: "#00000000",
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  });

  // Float above everything, on every space, including full screen ones.
  win.setAlwaysOnTop(true, "floating");
  win.setVisibleOnAllWorkspaces(true, { visibleOnFullScreen: true });

  // Let the mouse pass through the window until the pointer comes near the
  // device. The renderer turns this on and off as the pointer moves.
  win.setIgnoreMouseEvents(true, { forward: true });

  win.loadFile(path.join(__dirname, "index.html"));

  win.on("moved", writePosition);
  win.on("close", writePosition);
}

// The device is dragged by the renderer, which sends the movement here.
ipcMain.on("tama:move", (_event, dx, dy) => {
  if (!win) {
    return;
  }
  const [x, y] = win.getPosition();
  win.setPosition(Math.round(x + dx), Math.round(y + dy));
});

// The renderer tells the main process whether the pointer is over the device.
ipcMain.on("tama:pointer", (_event, over) => {
  if (!win) {
    return;
  }
  win.setIgnoreMouseEvents(!over, { forward: true });
});

// The renderer asks for the position to be written out.
ipcMain.on("tama:save", writePosition);

app.whenReady().then(() => {
  // The device is a pet, not an application. It does not belong in the dock.
  if (process.platform === "darwin") {
    app.dock?.hide();
  }

  createWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on("window-all-closed", () => {
  app.quit();
});
