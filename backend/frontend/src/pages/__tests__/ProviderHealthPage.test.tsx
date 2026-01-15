import { describe, expect, it, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { render, screen } from "@testing-library/react";

import { ProviderHealthPage } from "../ProviderHealthPage";

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(() => null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
};
Object.defineProperty(window, "localStorage", { value: localStorageMock });

const mockedSLIHook = vi.fn();
const mockedIncidentHook = vi.fn();
const mockedAlertHook = vi.fn();

vi.mock("@/api/hooks/useTelemetry", () => ({
  useProviderSLIs: (...args: unknown[]) => mockedSLIHook(...args),
  useProviderIncidents: (...args: unknown[]) => mockedIncidentHook(...args),
  useProviderAlerts: (...args: unknown[]) => mockedAlertHook(...args),
}));

vi.mock("@/providers/DirectoryProvider", () => ({
  useDirectoryData: () => ({
    tenants: [],
    users: [],
    models: [],
    tenantsLoading: false,
    usersLoading: false,
    modelsLoading: false,
    isLoading: false,
    refetchTenants: vi.fn(),
    refetchUsers: vi.fn(),
    refetchModels: vi.fn(),
  }),
}));

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProviderHealthPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ProviderHealthPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
  });

  it("renders empty states when no telemetry is available", () => {
    mockedSLIHook.mockReturnValue({ data: [], isLoading: false });
    mockedIncidentHook.mockReturnValue({ data: [], isLoading: false });
    mockedAlertHook.mockReturnValue({ data: [], isLoading: false });

    renderPage();

    expect(screen.getByText("Provider Health")).toBeInTheDocument();
    expect(screen.getByText("No telemetry data")).toBeInTheDocument();
    expect(screen.getByText("No active alerts")).toBeInTheDocument();
  });

  it("shows SLI and incident rows when data is returned", () => {
    mockedSLIHook.mockReturnValue({
      data: [
        {
          provider: "openai",
          alias: "gpt-5",
          route: "east",
          window_seconds: 300,
          window_start: new Date().toISOString(),
          window_end: new Date().toISOString(),
          request_count: 100,
          error_count: 5,
          timeout_count: 2,
          latency_p95_ms: 2500,
          error_rate: 0.05,
          timeout_rate: 0.02,
          thresholds: {
            latency_p95_ms: 3000,
            error_rate_threshold: 0.1,
            timeout_rate_threshold: 0.05,
            min_samples: 20,
          },
        },
      ],
      isLoading: false,
    });
    mockedIncidentHook.mockReturnValue({
      data: [
        {
          id: "inc-1",
          provider: "openai",
          alias: "gpt-5",
          type: "error_rate",
          status: "open",
          window: 300,
          opened_at: new Date().toISOString(),
          window_start: new Date().toISOString(),
          window_end: new Date().toISOString(),
          request_count: 100,
          error_count: 5,
          timeout_count: 0,
          latency_p95: 2500,
          error_rate: 0.05,
          timeout_rate: 0,
          thresholds: {
            latency_p95_ms: 3000,
            error_rate_threshold: 0.1,
            timeout_rate_threshold: 0.05,
            min_samples: 20,
          },
          metadata: {},
          sample_error: "boom",
        },
      ],
      isLoading: false,
    });
    mockedAlertHook.mockReturnValue({ data: [], isLoading: false });

    renderPage();

    expect(screen.getAllByText("openai").length).toBeGreaterThan(0);
    expect(screen.getAllByText("gpt-5").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/error rate/i).length).toBeGreaterThan(0);
    expect(screen.queryByText("No telemetry data")).not.toBeInTheDocument();
  });
});
