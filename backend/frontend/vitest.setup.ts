import '@testing-library/jest-dom/vitest';

// Recharts expects ResizeObserver; provide a minimal stub for jsdom.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

if (typeof globalThis.ResizeObserver === "undefined") {
  // @ts-ignore
  globalThis.ResizeObserver = ResizeObserverStub;
}
