import fs from "fs";
import path from "path";

const providers = {} as any;

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
        providers[content.key] = content;
      }
    }
  });
};

export const searchMetadata = async (
  target: string,
  by: string,
  value: string
) => {
  await loadMetadataProviders();

  const provider = providers[target];
  const url = provider.endpoints[by];

  const response = await fetch(url.replace("{{query}}", value));
  const data = await response.json();

  if (data[provider.properties.results_entity]) {
    let selectedEntity = null;
    const results = data[provider.properties.results_entity];

    if (provider.properties.library_type === "manga") {
      selectedEntity = await iterateManga(results, value, provider.key);
    } else {
      //
    }

    // Parse result.

    console.log("Metadata: ", selectedEntity);
    return selectedEntity;
  } else {
    return null;
  }
};

export const iterateManga = async (
  results: any[],
  query: string,
  providerKey: string,
  by?: string
) => {
  let selectedEntity = null;

  for (const result of results) {
    if (by) {
      if (result[by].toLowerCase() === query.toLowerCase()) {
        console.log("Found manga by by: ", result);
        selectedEntity = result;
        break;
      }
    } else {
      if (providers[providerKey].properties.search_array) {
        if (result[providers[providerKey].properties.search_array.key]) {
          console.log(
            result[providers[providerKey].properties.search_array.key]
          );
          if (
            result[providers[providerKey].properties.search_array.key].some(
              (t: any) => t.title.toLowerCase() === query.toLowerCase()
            )
          ) {
            selectedEntity = result;
            break;
          }
        }
      } else {
        if (result[providers[providerKey].properties.search_fallback]) {
          if (
            result[
              providers[providerKey].properties.search_fallback
            ].toLowerCase() === query.toLowerCase()
          ) {
            selectedEntity = result;
            break;
          }
        }
      }
    }
  }

  if (!selectedEntity) {
    selectedEntity = results[0];
  }

  const metadata = parseMetadata(selectedEntity, providers[providerKey].parser);
  console.log("Metadata: ", metadata);

  return metadata;
};

export const parseMetadata = (data: any, parser: any) => {
  let metadata = {} as any;

  const parserKeys = Object.keys(parser);

  for (const key of parserKeys) {
    if (typeof parser[key] === "object") {
      if (parser[key].key === "static") {
        metadata[key] = parser[key].value;
      } else {
        metadata[key] = data[parser[key].key].map(
          (t: any) => t[parser[key].value]
        );
      }
    } else {
      metadata[key] = data[parser[key]];
    }
  }

  return metadata;
};
