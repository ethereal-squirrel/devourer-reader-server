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
  value: string,
  apiKey?: string
) => {
  await loadMetadataProviders();

  const provider = providers[target];
  let url = provider.endpoints[by].replace("{{query}}", value);

  if (apiKey && provider.endpoints[by].includes("{{apiKey}}")) {
    url = url.replace("{{apiKey}}", apiKey);
  }

  const response = await fetch(url);
  const data = await response.json();

  const resultsData = getNestedProperty(
    data,
    provider.properties.results_entity
  );
  if (resultsData) {
    let selectedEntity = null;
    selectedEntity = await iterateResults(resultsData, value, provider.key);

    return selectedEntity;
  } else {
    return null;
  }
};

export const iterateResults = async (
  results: any[],
  query: string,
  providerKey: string,
  by?: string
) => {
  let selectedEntity = null;

  if (!results || results.length === 0) {
    return null;
  }

  for (const result of results) {
    if (by) {
      if (result[by].toLowerCase() === query.toLowerCase()) {
        selectedEntity = result;
        break;
      }
    } else {
      if (providers[providerKey].properties.search_array) {
        if (result[providers[providerKey].properties.search_array.key]) {
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

  const metadata = parseMetadata(
    selectedEntity,
    providers[providerKey].parser,
    providers[providerKey].postProcessing
  );

  return metadata;
};

const getNestedProperty = (obj: any, path: string): any => {
  if (!path) return null;

  const keys = path.split(".");
  let current = obj;

  for (const key of keys) {
    if (
      current === null ||
      current === undefined ||
      typeof current !== "object"
    ) {
      return null;
    }
    current = current[key];
  }

  return current === undefined ? null : current;
};

export const parseMetadata = (data: any, parser: any, postProcessing?: any) => {
  let metadata = {} as any;

  const parserKeys = Object.keys(parser);

  for (const key of parserKeys) {
    if (typeof parser[key] === "object" && parser[key] !== null) {
      if (parser[key].key === "static") {
        metadata[key] = parser[key].value;
      } else {
        const nestedData = getNestedProperty(data, parser[key].key);
        if (nestedData && Array.isArray(nestedData)) {
          metadata[key] = nestedData.map((t: any) => {
            const nestedValue = getNestedProperty(t, parser[key].value);
            return nestedValue !== null ? nestedValue : t[parser[key].value];
          });
        } else {
          metadata[key] = nestedData;
        }
      }
    } else if (parser[key] === null) {
      metadata[key] = null;
    } else {
      const nestedValue = getNestedProperty(data, parser[key]);
      metadata[key] = nestedValue !== null ? nestedValue : data[parser[key]];
    }
  }

  if (postProcessing && typeof postProcessing === "object") {
    const postProcessingKeys = Object.keys(postProcessing);

    for (const key of postProcessingKeys) {
      const config = postProcessing[key];

      if (config && config.action && metadata.hasOwnProperty(key)) {
        switch (config.action) {
          case "convertToArray":
            if (
              metadata[key] !== null &&
              metadata[key] !== undefined &&
              !Array.isArray(metadata[key])
            ) {
              metadata[key] = [metadata[key]];
            }
            break;
          case "convertToIndustryIdentifier":
            if (metadata[key]) {
              if (!metadata["identifiers"]) {
                metadata["identifiers"] = [];
              }

              metadata["identifiers"].push({
                type: key,
                value: metadata[key],
              });
            }
            break;
        }
      }
    }
  }

  return metadata;
};
