import fs from "fs";
import path from "path";
import util from "util";

const MAX_LOG_SIZE_BYTES = 5 * 1024 * 1024;

const logPath = process.pkg
  ? path.join(process.cwd(), "devourer.log")
  : path.resolve(__dirname, "..", "..", "devourer.log");

let stream: fs.WriteStream | null = null;
let currentSize = 0;

try {
  if (fs.existsSync(logPath)) {
    currentSize = fs.statSync(logPath).size;
  }
} catch {
  //
}

try {
  stream = fs.createWriteStream(logPath, { flags: "a" });
  stream.on("error", () => {
    stream = null;
  });
} catch {
  //
}

const orig = {
  error: console.error.bind(console),
  warn: console.warn.bind(console),
  debug: console.debug.bind(console),
  log: console.log.bind(console),
};

function formatArgs(args: any[]): string {
  return args
    .map((a) => {
      if (a instanceof Error) return a.stack || a.message;
      if (typeof a === "string") return a;
      return util.inspect(a, { depth: 6, colors: false, breakLength: 120 });
    })
    .join(" ");
}

function truncateLogIfNeeded(nextBytes: number) {
  if (currentSize + nextBytes <= MAX_LOG_SIZE_BYTES) return;

  try {
    if (stream && typeof (stream as any).fd === "number") {
      fs.ftruncateSync((stream as any).fd, 0);
    } else {
      fs.truncateSync(logPath, 0);
    }
    currentSize = 0;
  } catch {
    stream = null;
  }
}

function write(level: "ERROR" | "WARN" | "DEBUG" | "LOG", args: any[]) {
  const ts = new Date().toISOString();
  const line = `[${ts}] [${level}] ${formatArgs(args)}\n`;
  const bytes = Buffer.byteLength(line, "utf8");

  try {
    truncateLogIfNeeded(bytes);
    if (stream) {
      stream.write(line);
      currentSize += bytes;
    }
  } catch {
    //
  }
}

console.error = (...args: any[]) => {
  write("ERROR", args);
  orig.error(...args);
};

console.warn = (...args: any[]) => {
  write("WARN", args);
  orig.warn(...args);
};

console.debug = (...args: any[]) => {
  write("DEBUG", args);
  orig.debug(...args);
};

console.log = (...args: any[]) => {
  write("LOG", args);
  orig.log(...args);
};

process.on("uncaughtException", (err) => {
  write("ERROR", [err]);
  orig.error(err);
});

process.on("unhandledRejection", (reason) => {
  write("ERROR", [reason]);
  orig.error("Unhandled Rejection:", reason as any);
});

export {};
