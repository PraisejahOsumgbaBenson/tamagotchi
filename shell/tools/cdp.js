// A tiny Chrome DevTools Protocol client for poking at the running shell.
//
// Usage: node tools/cdp.js <command> [args]
//
//   shot <file>       capture a screenshot of the page
//   eval <expr>       evaluate an expression in the page
//   click <selector>  press and release the middle of an element
//   logs              print console messages collected since load

const fs = require("node:fs");

const ENDPOINT = process.env.CDP || "http://127.0.0.1:9222";

async function pageTarget() {
  const res = await fetch(`${ENDPOINT}/json/list`);
  const pages = await res.json();
  const page = pages.find((p) => p.type === "page");
  if (!page) {
    throw new Error("no page target found");
  }
  return page;
}

function connect(url) {
  return new Promise((resolve, reject) => {
    const socket = new WebSocket(url);
    socket.addEventListener("open", () => resolve(socket));
    socket.addEventListener("error", reject);
  });
}

let nextId = 1;

function send(socket, method, params = {}) {
  const id = nextId++;
  return new Promise((resolve, reject) => {
    const onMessage = (event) => {
      const message = JSON.parse(event.data);
      if (message.id !== id) {
        return;
      }
      socket.removeEventListener("message", onMessage);
      if (message.error) {
        reject(new Error(`${method}: ${message.error.message}`));
        return;
      }
      resolve(message.result);
    };
    socket.addEventListener("message", onMessage);
    socket.send(JSON.stringify({ id, method, params }));
  });
}

async function main() {
  const [command, ...args] = process.argv.slice(2);
  const target = await pageTarget();
  const socket = await connect(target.webSocketDebuggerUrl);

  const consoleMessages = [];
  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    if (message.method === "Runtime.consoleAPICalled") {
      consoleMessages.push(
        `${message.params.type}: ${message.params.args
          .map((a) => a.value ?? a.description)
          .join(" ")}`
      );
    }
    if (message.method === "Runtime.exceptionThrown") {
      consoleMessages.push(
        `exception: ${message.params.exceptionDetails.text} ${
          message.params.exceptionDetails.exception?.description ?? ""
        }`
      );
    }
  });

  await send(socket, "Runtime.enable");
  await send(socket, "Page.enable");

  switch (command) {
    case "shot": {
      const file = args[0] || "shot.png";
      const result = await send(socket, "Page.captureScreenshot", {
        format: "png",
        captureBeyondViewport: false,
        fromSurface: true,
      });
      fs.writeFileSync(file, Buffer.from(result.data, "base64"));
      console.log(`wrote ${file}`);
      break;
    }

    case "eval": {
      const result = await send(socket, "Runtime.evaluate", {
        expression: args.join(" "),
        returnByValue: true,
        awaitPromise: true,
      });
      if (result.exceptionDetails) {
        console.error(result.exceptionDetails.text, result.exceptionDetails.exception?.description);
        process.exitCode = 1;
      } else {
        console.log(JSON.stringify(result.result.value, null, 2));
      }
      break;
    }

    case "click": {
      const selector = args.join(" ");
      const found = await send(socket, "Runtime.evaluate", {
        expression: `(() => {
          const el = document.querySelector(${JSON.stringify(selector)});
          if (!el) return null;
          const r = el.getBoundingClientRect();
          return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
        })()`,
        returnByValue: true,
      });
      const point = found.result.value;
      if (!point) {
        throw new Error(`no element matches ${selector}`);
      }
      for (const type of ["mousePressed", "mouseReleased"]) {
        await send(socket, "Input.dispatchMouseEvent", {
          type,
          x: point.x,
          y: point.y,
          button: "left",
          buttons: type === "mousePressed" ? 1 : 0,
          clickCount: 1,
          pointerType: "mouse",
        });
      }
      console.log(`clicked ${selector} at ${point.x},${point.y}`);
      break;
    }

    case "drag": {
      // Drag from one point to another, in window CSS pixels.
      const [x1, y1, x2, y2] = args.map(Number);
      if ([x1, y1, x2, y2].some((n) => !Number.isFinite(n))) {
        throw new Error("usage: drag <x1> <y1> <x2> <y2>");
      }
      const steps = 8;
      await send(socket, "Input.dispatchMouseEvent", {
        type: "mousePressed",
        x: x1,
        y: y1,
        button: "left",
        buttons: 1,
        clickCount: 1,
      });
      for (let i = 1; i <= steps; i++) {
        await send(socket, "Input.dispatchMouseEvent", {
          type: "mouseMoved",
          x: x1 + ((x2 - x1) * i) / steps,
          y: y1 + ((y2 - y1) * i) / steps,
          button: "left",
          buttons: 1,
        });
      }
      await send(socket, "Input.dispatchMouseEvent", {
        type: "mouseReleased",
        x: x2,
        y: y2,
        button: "left",
        buttons: 0,
        clickCount: 1,
      });
      console.log(`dragged ${x1},${y1} to ${x2},${y2}`);
      break;
    }

    case "logs": {
      // Give the page a moment to produce anything it is going to.
      await new Promise((r) => setTimeout(r, 500));
      if (consoleMessages.length === 0) {
        console.log("(no console output)");
      } else {
        console.log(consoleMessages.join("\n"));
      }
      break;
    }

    default:
      throw new Error(`unknown command ${command}`);
  }

  socket.close();
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
