const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

// Main build process
const build = () => {
  console.log("Building TypeScript...");
  execSync("npm run build", { stdio: "inherit" });

  console.log("Generating Prisma client...");
  execSync("npm run pkg:generate", { stdio: "inherit" });

  console.log("Building executable...");
  execSync("pkg .", { stdio: "inherit" });

  console.log("Build complete!");
};

build();
