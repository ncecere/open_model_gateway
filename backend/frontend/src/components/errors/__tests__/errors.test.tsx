import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ErrorFallback } from "../ErrorFallback";
import { ErrorBoundary } from "../ErrorBoundary";
import { FeatureErrorBoundary } from "../FeatureErrorBoundary";

// Suppress console.error for error boundary tests
beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
});

// Component that throws an error for testing
function ThrowError({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error("Test error message");
  }
  return <div>Content rendered successfully</div>;
}

describe("ErrorFallback", () => {
  const mockReset = vi.fn();
  const defaultError = new Error("Test error");

  it("renders card variant by default", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
      />
    );

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("Test error")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
  });

  it("renders page variant", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
        variant="page"
      />
    );

    expect(screen.getByRole("heading", { name: "Something went wrong" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
  });

  it("renders inline variant", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
        variant="inline"
      />
    );

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Test error")).toBeInTheDocument();
  });

  it("uses custom title", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
        title="Custom error title"
      />
    );

    expect(screen.getByText("Custom error title")).toBeInTheDocument();
  });

  it("hides error message when showError is false", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
        showError={false}
      />
    );

    expect(screen.queryByText("Test error")).not.toBeInTheDocument();
  });

  it("hides reset button when showReset is false", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
        showReset={false}
      />
    );

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("uses custom reset text", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
        resetText="Retry operation"
      />
    );

    expect(screen.getByRole("button", { name: /retry operation/i })).toBeInTheDocument();
  });

  it("calls resetErrorBoundary when reset button is clicked", () => {
    render(
      <ErrorFallback
        error={defaultError}
        resetErrorBoundary={mockReset}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: /try again/i }));
    expect(mockReset).toHaveBeenCalledTimes(1);
  });
});

describe("ErrorBoundary", () => {
  it("renders children when no error occurs", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={false} />
      </ErrorBoundary>
    );

    expect(screen.getByText("Content rendered successfully")).toBeInTheDocument();
  });

  it("renders fallback when error occurs", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Test error message")).toBeInTheDocument();
  });

  it("uses custom title", () => {
    render(
      <ErrorBoundary title="Custom boundary error">
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText("Custom boundary error")).toBeInTheDocument();
  });

  it("uses page variant", () => {
    render(
      <ErrorBoundary variant="page">
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByRole("heading")).toBeInTheDocument();
  });

  it("calls onError callback when error occurs", () => {
    const onError = vi.fn();
    render(
      <ErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(onError).toHaveBeenCalled();
    expect(onError.mock.calls[0][0]).toBeInstanceOf(Error);
    expect(onError.mock.calls[0][0].message).toBe("Test error message");
  });
});

describe("FeatureErrorBoundary", () => {
  it("renders children when no error occurs", () => {
    render(
      <FeatureErrorBoundary featureName="TestFeature">
        <ThrowError shouldThrow={false} />
      </FeatureErrorBoundary>
    );

    expect(screen.getByText("Content rendered successfully")).toBeInTheDocument();
  });

  it("renders fallback with feature name in title", () => {
    render(
      <FeatureErrorBoundary featureName="TestFeature">
        <ThrowError shouldThrow={true} />
      </FeatureErrorBoundary>
    );

    expect(screen.getByText("TestFeature encountered an error")).toBeInTheDocument();
  });

  it("uses custom title if provided", () => {
    render(
      <FeatureErrorBoundary featureName="TestFeature" title="Custom feature error">
        <ThrowError shouldThrow={true} />
      </FeatureErrorBoundary>
    );

    expect(screen.getByText("Custom feature error")).toBeInTheDocument();
  });

  it("supports inline variant", () => {
    render(
      <FeatureErrorBoundary featureName="TestFeature" variant="inline">
        <ThrowError shouldThrow={true} />
      </FeatureErrorBoundary>
    );

    expect(screen.getByRole("alert")).toHaveClass("flex", "items-center");
  });

  it("allows retry", () => {
    const { rerender } = render(
      <FeatureErrorBoundary featureName="TestFeature">
        <ThrowError shouldThrow={true} />
      </FeatureErrorBoundary>
    );

    expect(screen.getByText("TestFeature encountered an error")).toBeInTheDocument();

    // Click retry
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));

    // After reset, the component will try to render again
    // Since it will throw again, we should still see the error
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});

// Note: PageErrorBoundary and AppErrorBoundary tests would need more setup
// since they depend on react-router-dom and window.location
