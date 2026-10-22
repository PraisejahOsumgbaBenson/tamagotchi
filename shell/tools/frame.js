// Print the frame the engine is showing right now.
//
// The engine only publishes frames when the screen changes, so a fresh
// connection is handed a snapshot instead. That is what this asks for.

const ws = new WebSocket("ws://127.0.0.1:8733/ws");

// --json prints the raw frame so a script can compare it with the pixels.
const asJSON = process.argv.includes("--json");

ws.addEventListener("message", (event) => {
  const message = JSON.parse(event.data);
  if (message.t !== "frame") {
    return;
  }
  if (asJSON) {
    console.log(
      JSON.stringify({
        rows: message.rows,
        icons: (message.icons || 0) >>> 0,
        sel: (message.sel || 0) >>> 0,
      })
    );
    ws.close();
    process.exit(0);
    return;
  }
  const icons = (message.icons || 0) >>> 0;
  const sel = (message.sel || 0) >>> 0;
  console.log(`icons ${icons.toString(2).padStart(8, "0")}  sel ${sel.toString(2).padStart(8, "0")}`);
  message.rows.forEach((row, index) => {
    const bits = (row >>> 0).toString(2).padStart(32, "0");
    console.log(String(index).padStart(2, " ") + " " + bits);
  });
  ws.close();
  process.exit(0);
});

setTimeout(() => {
  console.error("the engine did not answer");
  process.exit(1);
}, 5000);
