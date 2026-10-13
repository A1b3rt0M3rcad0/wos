export function normalizeLabel(value) {
  if (typeof value !== "string") {
    throw new TypeError("normalizeLabel expects a string");
  }
  return value.trim();
}
