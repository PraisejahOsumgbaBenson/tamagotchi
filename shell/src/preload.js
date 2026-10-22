// The preload script.
//
// It gives the renderer a small, fixed set of things it is allowed to do.
// The renderer has no Node and no direct access to the main process.

const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("tama", {
  // move shifts the window by a number of pixels.
  move: (dx, dy) => ipcRenderer.send("tama:move", dx, dy),

  // pointer tells the main process whether the mouse is over the device, so
  // that clicks outside it fall through to whatever is underneath.
  pointer: (over) => ipcRenderer.send("tama:pointer", over),

  // save writes out where the device is on the desktop.
  save: () => ipcRenderer.send("tama:save"),

  // menu opens the little menu that holds the way out of the app.
  menu: () => ipcRenderer.send("tama:menu"),
});
