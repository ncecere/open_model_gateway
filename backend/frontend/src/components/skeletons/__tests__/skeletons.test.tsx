import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { PageSkeleton } from "../PageSkeleton";
import { FeatureSkeleton } from "../FeatureSkeleton";

describe("PageSkeleton", () => {
  it("renders with default props", () => {
    const { container } = render(<PageSkeleton />);

    // Should have header, stat cards, and content sections
    expect(container.querySelector(".space-y-6")).toBeInTheDocument();
  });

  it("hides header when showHeader is false", () => {
    const { container } = render(<PageSkeleton showHeader={false} />);

    // Should not have header section with title skeleton (h-8 w-48)
    const headerSkeleton = container.querySelector(".h-8.w-48");
    expect(headerSkeleton).not.toBeInTheDocument();
  });

  it("renders correct number of stat cards", () => {
    const { container } = render(<PageSkeleton statCards={5} />);

    // Should have 5 stat card containers
    const statCards = container.querySelectorAll(".grid > .rounded-lg");
    expect(statCards).toHaveLength(5);
  });

  it("hides stat cards when statCards is 0", () => {
    const { container } = render(<PageSkeleton statCards={0} />);

    // Should not have stat card grid
    const grid = container.querySelector(".grid.gap-4");
    expect(grid).not.toBeInTheDocument();
  });

  it("hides content when showContent is false", () => {
    const { container } = render(<PageSkeleton showContent={false} />);

    // Content area has specific structure - check for absence
    const contentBorder = container.querySelectorAll(".rounded-lg.border.bg-card");
    // Only stat cards should remain (3 by default)
    expect(contentBorder).toHaveLength(3);
  });

  it("applies custom className", () => {
    const { container } = render(<PageSkeleton className="custom-class" />);

    expect(container.querySelector(".custom-class")).toBeInTheDocument();
  });

  it("uses custom contentHeight", () => {
    const { container } = render(<PageSkeleton contentHeight="h-64" />);

    expect(container.querySelector(".h-64")).toBeInTheDocument();
  });
});

describe("FeatureSkeleton", () => {
  it("renders card variant by default", () => {
    const { container } = render(<FeatureSkeleton />);

    // Card variant has specific structure
    expect(container.querySelector(".rounded-lg.border.bg-card.p-6")).toBeInTheDocument();
  });

  it("renders list variant", () => {
    const { container } = render(<FeatureSkeleton variant="list" items={3} />);

    // List variant renders items with rounded-full avatars
    const avatars = container.querySelectorAll(".rounded-full");
    expect(avatars.length).toBeGreaterThanOrEqual(3);
  });

  it("renders form variant", () => {
    const { container } = render(<FeatureSkeleton variant="form" />);

    // Form variant has input field skeletons (h-10 w-full)
    const inputs = container.querySelectorAll(".h-10.w-full");
    expect(inputs.length).toBeGreaterThanOrEqual(2);
  });

  it("renders stats variant", () => {
    const { container } = render(<FeatureSkeleton variant="stats" items={4} />);

    // Stats variant uses grid layout
    const grid = container.querySelector(".grid.gap-4");
    expect(grid).toBeInTheDocument();

    // Should have 4 stat cards
    const cards = container.querySelectorAll(".rounded-lg.border.bg-card.p-4");
    expect(cards).toHaveLength(4);
  });

  it("uses default 3 items for list variant", () => {
    const { container } = render(<FeatureSkeleton variant="list" />);

    // Default is 3 items
    const items = container.querySelectorAll(".flex.items-center.gap-4.p-3");
    expect(items).toHaveLength(3);
  });

  it("applies custom className", () => {
    const { container } = render(<FeatureSkeleton className="custom-class" />);

    expect(container.querySelector(".custom-class")).toBeInTheDocument();
  });
});
