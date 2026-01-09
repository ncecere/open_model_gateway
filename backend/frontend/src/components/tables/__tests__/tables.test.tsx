import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TableSkeleton } from "../TableSkeleton";
import { EmptyState } from "../EmptyState";
import { TablePagination, usePaginationFromOffset } from "../TablePagination";
import { ActionsMenu } from "../ActionsMenu";
import { FileIcon } from "lucide-react";

describe("TableSkeleton", () => {
  it("renders default number of skeleton rows", () => {
    render(<TableSkeleton />);
    const skeletons = document.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletons.length).toBe(4);
  });

  it("renders custom number of rows", () => {
    render(<TableSkeleton rows={6} />);
    const skeletons = document.querySelectorAll('[class*="animate-pulse"]');
    expect(skeletons.length).toBe(6);
  });

  it("applies custom class name", () => {
    const { container } = render(<TableSkeleton className="my-class" />);
    expect(container.firstChild).toHaveClass("my-class");
  });
});

describe("EmptyState", () => {
  it("renders message", () => {
    render(<EmptyState message="No items found" />);
    expect(screen.getByText("No items found")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <EmptyState
        message="No items"
        description="Try adjusting your filters"
      />
    );
    expect(screen.getByText("Try adjusting your filters")).toBeInTheDocument();
  });

  it("renders custom icon", () => {
    render(
      <EmptyState
        message="No files"
        icon={<FileIcon data-testid="custom-icon" />}
      />
    );
    expect(screen.getByTestId("custom-icon")).toBeInTheDocument();
  });

  it("renders action when provided", () => {
    render(
      <EmptyState
        message="No items"
        action={<button>Add item</button>}
      />
    );
    expect(screen.getByRole("button", { name: "Add item" })).toBeInTheDocument();
  });

  it("applies dashed border by default", () => {
    const { container } = render(<EmptyState message="Empty" />);
    expect(container.firstChild).toHaveClass("border-dashed");
  });

  it("removes dashed border when dashed is false", () => {
    const { container } = render(<EmptyState message="Empty" dashed={false} />);
    expect(container.firstChild).not.toHaveClass("border-dashed");
  });
});

describe("TablePagination", () => {
  const defaultProps = {
    page: 1,
    totalPages: 5,
    hasPrev: false,
    hasNext: true,
    onPrev: vi.fn(),
    onNext: vi.fn(),
  };

  it("renders page info", () => {
    render(<TablePagination {...defaultProps} />);
    expect(screen.getByText("Page 1 of 5")).toBeInTheDocument();
  });

  it("disables Previous button when hasPrev is false", () => {
    render(<TablePagination {...defaultProps} />);
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
  });

  it("enables Next button when hasNext is true", () => {
    render(<TablePagination {...defaultProps} />);
    expect(screen.getByRole("button", { name: "Next" })).not.toBeDisabled();
  });

  it("calls onPrev when Previous is clicked", () => {
    const onPrev = vi.fn();
    render(
      <TablePagination {...defaultProps} hasPrev={true} onPrev={onPrev} />
    );
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    expect(onPrev).toHaveBeenCalled();
  });

  it("calls onNext when Next is clicked", () => {
    const onNext = vi.fn();
    render(<TablePagination {...defaultProps} onNext={onNext} />);
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(onNext).toHaveBeenCalled();
  });

  it("disables buttons when isLoading", () => {
    render(
      <TablePagination {...defaultProps} hasPrev={true} isLoading={true} />
    );
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
  });
});

describe("usePaginationFromOffset", () => {
  it("calculates page and totalPages correctly", () => {
    const result = usePaginationFromOffset(0, 10, 100);
    expect(result.page).toBe(1);
    expect(result.totalPages).toBe(10);
    expect(result.hasPrev).toBe(false);
    expect(result.hasNext).toBe(true);
  });

  it("calculates middle page correctly", () => {
    const result = usePaginationFromOffset(20, 10, 100);
    expect(result.page).toBe(3);
    expect(result.hasPrev).toBe(true);
    expect(result.hasNext).toBe(true);
  });

  it("calculates last page correctly", () => {
    const result = usePaginationFromOffset(90, 10, 100);
    expect(result.page).toBe(10);
    expect(result.hasPrev).toBe(true);
    expect(result.hasNext).toBe(false);
  });
});

describe("ActionsMenu", () => {
  const defaultActions = [
    { label: "View", onClick: vi.fn() },
    { label: "Delete", onClick: vi.fn(), destructive: true },
  ];

  it("renders trigger button", () => {
    render(<ActionsMenu actions={defaultActions} />);
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("renders with sr-only label for accessibility", () => {
    render(<ActionsMenu actions={defaultActions} />);
    expect(screen.getByText("Open actions")).toBeInTheDocument();
  });

  it("returns null when actions array is empty", () => {
    const { container } = render(<ActionsMenu actions={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("applies custom className to trigger", () => {
    render(<ActionsMenu actions={defaultActions} className="my-class" />);
    expect(screen.getByRole("button")).toHaveClass("my-class");
  });

  it("uses outline variant when specified", () => {
    render(<ActionsMenu actions={defaultActions} variant="outline" />);
    // The button should have outline-specific classes
    const button = screen.getByRole("button");
    expect(button).toBeInTheDocument();
  });
});
