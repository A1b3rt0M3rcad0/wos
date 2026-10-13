export function normalizeLabel(value) {
  if (typeof value !== "string") {
    throw new TypeError("normalizeLabel expects a string");
  }
  // Round 1 controlled omission: internal whitespace collapse is intentionally absent.
  return value.trim();
}
