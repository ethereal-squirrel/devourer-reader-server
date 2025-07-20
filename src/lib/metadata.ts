import fs from "fs";
import path from "path";

const providers = {
  book: [] as any,
  manga: [] as any[],
} as any;

export const loadMetadataProviders = async () => {
  const providersPath = path.join(__dirname, "../../plugins/providers");
  loadJsonFiles(providersPath);

  return providers;
};

const loadJsonFiles = (dir: string) => {
  const files = fs.readdirSync(dir);

  files.forEach((file: string) => {
    const fullPath = path.join(dir, file);
    const stat = fs.statSync(fullPath);

    if (stat.isDirectory()) {
      loadJsonFiles(fullPath);
    } else if (file.endsWith(".json")) {
      const content = JSON.parse(fs.readFileSync(fullPath, "utf8"));
      if (content.type === "metadata" && content.properties?.library_type) {
        providers[content.properties.library_type].push(content);
      }
    }
  });
};
