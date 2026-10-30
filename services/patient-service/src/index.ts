import http from "http";

const PORT = Number(process.env.PORT ?? 3000);

const server = http.createServer((req, res) => {
  if (req.url === "/health") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ status: "UP", service: "patient-service" }));
    return;
  }
  res.writeHead(404, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ error: "Not Found" }));
});

server.listen(PORT, () => {
  console.log(`[Patient Service] aktif pada port ${PORT}`);
});

// Graceful shutdown (Twelve-Factor IX: Disposability)
process.on("SIGTERM", () => {
  console.log("[Patient Service] SIGTERM diterima, mematikan server...");
  server.close(() => process.exit(0));
});
