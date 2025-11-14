export function getBrowserTimezone(): string {
  if (typeof Intl !== "undefined" && typeof Intl.DateTimeFormat === "function") {
    try {
      const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
      if (tz && tz.length > 0) {
        return tz;
      }
    } catch (_err) {
      // ignore and fall back to UTC
    }
  }
  return "UTC";
}
