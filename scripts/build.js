const { execSync } = require("child_process");

// Parse command line arguments
const parseArgs = () => {
  const args = process.argv.slice(2);
  let target = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === "-t" && i + 1 < args.length) {
      target = args[i + 1];
      break;
    }
  }

  return { target };
};

// Main build process

/*
    Valid Targets:
    "node20-win-x64",
    "node20-linux-x64",
    "node20-macos-x64"
*/

const build = () => {
  const { target } = parseArgs();

  console.log("Building TypeScript...");
  execSync("npm run build", { stdio: "inherit" });

  console.log("Generating Prisma client...");
  execSync("npm run pkg:generate", { stdio: "inherit" });

  console.log("Building executable...");
  const pkgCommand = target ? `pkg . -t ${target}` : "pkg .";
  execSync(pkgCommand, { stdio: "inherit" });

  console.log("Build complete!");
};

build();
