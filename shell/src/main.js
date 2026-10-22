// The main process.
//
// The shell is a small frameless window that sits on top of everything else
// on the desktop. It has no chrome, no menu bar and no dock icon, so the
// only thing on screen is the device itself.
//
// The window is bigger than the device so that the device can be dragged
// around by its edges, and the parts of the window that are not the device
// let the mouse through to whatever is underneath.

const { app, BrowserWindow, Menu, ipcMain, screen } = require("electron");
const { spawn } = require("node:child_process");
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

// engine is the pet, running as a child process.
let engine = null;

// ENGINE is where the engine binary is expected to be, relative to this file.
const ENGINE = path.join(__dirname, "..", "..", "dist", "tama-engine");

// startEngine runs the engine and keeps hold of it.
//
// The two halves are separate processes on purpose, but launching them is not
// something the person using this should have to think about, and forgetting
// the engine looks exactly like a broken pet: a blank screen. So the shell
// starts it, and quits it.
function startEngine() {
  if (!fs.existsSync(ENGINE)) {
    console.error(
      `tama: no engine at ${ENGINE}. Run "make build" first.`
    );
    return null;
  }

  engine = spawn(ENGINE, [], { stdio: "inherit" });

  engine.on("error", (err) => {
    console.error("tama: the engine would not start", err);
    engine = null;
  });

  if (engine) {
    engine.on("exit", (code, signal) => {
      if (signal !== "SIGTERM") {
        console.error(`tama: the engine stopped (code ${code}, signal ${signal})`);
      }
      engine = null;
    });
  }

  return engine;
}

// stopEngine asks the engine to save and quit.
function stopEngine() {
  if (engine) {
    engine.kill("SIGTERM");
    engine = null;
  }
}

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

// inView brings a saved position back onto a display.
//
// A position saved next to a monitor that is no longer plugged in would leave
// the device outside every screen, with no dock icon and no menu bar to find
// it with, so the position is pulled back into the work area first.
function inView(position) {
  const display = screen.getDisplayNearestPoint(position);
  const area = display.workArea;
  const width = DEVICE.width + PADDING * 2;
  const height = DEVICE.height + PADDING * 2;

  return {
    x: Math.min(Math.max(position.x, area.x), area.x + area.width - width),
    y: Math.min(Math.max(position.y, area.y), area.y + area.height - height),
  };
}

// createWindow builds the floating window.
function createWindow() {
  const saved = readPosition();
  const start = inView(saved || defaultPosition());

  win = new BrowserWindow({
    x: start.x,
    y: start.y,
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
  //
  // A window that is moved to every space can be left out of the window
  // order, and this app has no dock icon to click to bring it back. Showing
  // it again once it is ready puts it in front of the other floating windows
  // without taking the keyboard from whatever is being typed into.
  win.setAlwaysOnTop(true, "floating");
  win.setVisibleOnAllWorkspaces(true, { visibleOnFullScreen: true });

  // Let the mouse pass through the window until the pointer comes near the
  // device. The renderer turns this on and off as the pointer moves.
  win.setIgnoreMouseEvents(true, { forward: true });

  win.loadFile(path.join(__dirname, "index.html"));

  win.once("ready-to-show", () => {
    // Inactive, so that the pet does not steal the cursor from whatever is
    // being typed into.
    win.showInactive();
    win.moveTop();
  });

  win.on("moved", writePosition);
  win.on("close", writePosition);
}

// The device has no dock icon and no menu bar, so it needs a way to be
// closed that does not involve a terminal.
ipcMain.on("tama:menu", () => {
  if (!win) {
    return;
  }
  Menu.buildFromTemplate([
    {
      label: "Quit Tamagotchi",
      accelerator: "Command+Q",
      click: () => app.quit(),
    },
  ]).popup({ window: win });
});

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
    // A menu is still needed for the keyboard shortcuts that come with it,
    // so that Command Q closes the pet while it has the keyboard.
    Menu.setApplicationMenu(Menu.buildFromTemplate([{ role: "appMenu" }]));
  }

  createWindow();
  startEngine();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

// The engine is a child of the shell, so it goes when the shell goes. It
// saves its state on the way out.
app.on("before-quit", stopEngine);
app.on("window-all-closed", () => {
  stopEngine();
  app.quit();
});
