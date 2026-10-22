// The renderer.
//
// It sits between the engine and the device. It keeps the WebSocket to the
// engine open, paints the display, makes the noises, and turns the pointer
// into button presses.
//
// The engine is the half that knows how to keep a pet alive. This half
// knows nothing about pets at all.

(function () {
  const canvas = document.getElementById("lcd");
  const device = document.getElementById("device");
  const buttons = Array.from(document.querySelectorAll(".tama-btn"));

  // Where the engine listens. It is a local process on this machine.
  const ENGINE_URL = "ws://127.0.0.1:8733/ws";

  // How often to tell the engine what time it is. The pet keeps its own
  // clock, and the engine corrects it from this.
  const CLOCK_INTERVAL = 30000;

  const panel = {
    rows: new Array(16).fill(0),
    icons: 0,
    sel: 0,
  };

  const held = { a: false, b: false, c: false };

  const notice = document.getElementById("notice");

  // A second of silence is normal while the engine starts, so the notice
  // waits. Once the engine has answered, a later gap keeps the last frame and
  // says nothing, because a pet that briefly loses its engine should not look
  // like it has been replaced by an error message.
  let everConnected = false;
  const noticeTimer = setTimeout(() => {
    if (!everConnected) {
      notice.textContent = "No engine. Run make build.";
      notice.hidden = false;
    }
  }, 3000);

  function hideNotice() {
    everConnected = true;
    clearTimeout(noticeTimer);
    notice.hidden = true;
  }

  let socket = null;
  let reconnectTimer = null;
  let reconnectDelay = 500;
  let pointerOver = false;

  // ---- the engine ----

  // connect opens the WebSocket to the engine.
  function connect() {
    try {
      socket = new WebSocket(ENGINE_URL);
    } catch (err) {
      scheduleReconnect();
      return;
    }

    socket.addEventListener("open", () => {
      reconnectDelay = 500;
      hideNotice();
      sendClock();
    });

    socket.addEventListener("message", (event) => {
      let message;
      try {
        message = JSON.parse(event.data);
      } catch (err) {
        return;
      }
      handle(message);
    });

    socket.addEventListener("close", scheduleReconnect);
    socket.addEventListener("error", () => {
      // The close event follows, and that is where the retry happens.
    });
  }

  // scheduleReconnect tries again after a while, backing off so that a
  // missing engine does not spin.
  function scheduleReconnect() {
    if (reconnectTimer) {
      return;
    }
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      reconnectDelay = Math.min(reconnectDelay * 2, 5000);
      connect();
    }, reconnectDelay);
  }

  // send writes a message to the engine.
  function send(message) {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message));
    }
  }

  // handle acts on a message from the engine.
  function handle(message) {
    switch (message.t) {
      case "hello":
        sendClock();
        break;

      case "frame":
        if (Array.isArray(message.rows) && message.rows.length === 16) {
          panel.rows = message.rows;
        }
        panel.icons = message.icons || 0;
        panel.sel = message.sel || 0;
        draw();
        break;

      case "audio":
        window.TamaAudio.beep(message.freq, message.dur);
        break;

      default:
        break;
    }
  }

  // draw paints the display.
  function draw() {
    window.TamaLCD.draw(canvas, panel.rows, panel.icons, panel.sel);
  }

  // sendClock tells the engine what time it is.
  function sendClock() {
    send({ t: "clock", epoch: Math.floor(Date.now() / 1000) });
  }

  // ---- the buttons ----

  // sendButtons reports the state of all three buttons. The engine works out
  // what a press means, including the combinations, so it needs the whole
  // picture rather than a list of events.
  function sendButtons() {
    send({ t: "btn", a: held.a, b: held.b, c: held.c });
  }

  buttons.forEach((button) => {
    const name = button.dataset.btn;

    button.addEventListener("pointerdown", (event) => {
      event.preventDefault();
      event.stopPropagation();

      window.TamaAudio.unlock();
      window.TamaAudio.click();

      held[name] = true;
      button.classList.add("pressed");
      button.setPointerCapture(event.pointerId);
      sendButtons();
    });

    const release = (event) => {
      if (!held[name]) {
        return;
      }
      held[name] = false;
      button.classList.remove("pressed");
      sendButtons();
      if (event && event.pointerId !== undefined) {
        try {
          button.releasePointerCapture(event.pointerId);
        } catch (err) {
          // The capture may already have gone.
        }
      }
    };

    button.addEventListener("pointerup", release);
    button.addEventListener("pointercancel", release);
    button.addEventListener("lostpointercapture", release);
  });

  // ---- moving the device around the desktop ----

  let dragging = false;
  let dragPointer = null;

  device.addEventListener("pointerdown", (event) => {
    if (event.target.closest(".tama-btn")) {
      return;
    }
    dragging = true;
    dragPointer = event.pointerId;
    device.setPointerCapture(event.pointerId);
    event.preventDefault();
  });

  document.addEventListener("pointermove", (event) => {
    if (!dragging || event.pointerId !== dragPointer) {
      return;
    }
    // The movement is relative, so it keeps working as the window moves
    // out from under the pointer.
    window.tama.move(event.movementX, event.movementY);
  });

  const endDrag = (event) => {
    if (event.pointerId !== dragPointer) {
      return;
    }
    dragging = false;
    dragPointer = null;
    window.tama.save();
  };

  document.addEventListener("pointerup", endDrag);
  document.addEventListener("pointercancel", endDrag);

  // ---- quitting ----

  // There is no dock icon and no title bar, so the device carries the only
  // way out of the app that does not involve a terminal. The same menu holds
  // the shell designs, which the main process remembers and sends from here.
  device.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    window.tama.menu();
  });

  window.tama.onTheme((name) => {
    device.dataset.theme = name;
  });

  // ---- letting the mouse through ----

  // The window covers more than the device so that the device can be moved
  // around. Everything outside the device has to let the mouse through to
  // whatever is underneath it.
  //
  // The test asks the document what is under the pointer rather than
  // comparing against a rectangle, because the device is an egg. A rectangle
  // would swallow clicks in the transparent corners.
  document.addEventListener("mousemove", (event) => {
    const hit = document.elementFromPoint(event.clientX, event.clientY);
    const over = hit !== null && device.contains(hit);

    if (over !== pointerOver) {
      pointerOver = over;
      window.tama.pointer(over);
    }
  });

  // ---- start up ----

  draw();
  connect();
  sendClock();
  setInterval(sendClock, CLOCK_INTERVAL);
})();
