export function applyBranding(iconHref?: string) {
  if (typeof document === "undefined" || !iconHref) {
    return;
  }

  const ensureLink = (rel: string) => {
    const existing = document.head.querySelector<HTMLLinkElement>(
      `link[rel='${rel}']`,
    );
    if (existing) {
      return existing;
    }
    const link = document.createElement("link");
    link.setAttribute("rel", rel);
    document.head.appendChild(link);
    return link;
  };

  ["icon", "shortcut icon"].forEach((rel) => {
    const link = ensureLink(rel);
    link.type = "image/svg+xml";
    link.href = iconHref;
  });
}
