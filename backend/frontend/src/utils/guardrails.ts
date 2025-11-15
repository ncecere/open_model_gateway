export function parseKeywordInput(value: string): string[] {
  return value
    .split(/[,\n]/)
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
}

export function formatKeywordInput(values?: string[]): string {
  if (!values || values.length === 0) {
    return "";
  }
  return values.join("\n");
}
