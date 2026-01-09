import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DetailItem } from "../DetailItem";
import { BaseDialog } from "../BaseDialog";
import { DetailsDialog } from "../DetailsDialog";
import { FormDialog } from "../FormDialog";
import { ConfirmDialog } from "../ConfirmDialog";

describe("DetailItem", () => {
  it("renders label and children", () => {
    render(<DetailItem label="Name">John Doe</DetailItem>);
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("applies className to container", () => {
    const { container } = render(
      <DetailItem label="Status" className="capitalize">
        active
      </DetailItem>
    );
    const wrapper = container.firstChild;
    expect(wrapper).toHaveClass("capitalize");
  });
});

describe("BaseDialog", () => {
  it("renders dialog with title and content", () => {
    render(
      <BaseDialog
        open={true}
        onOpenChange={() => {}}
        title="Test Dialog"
      >
        <p>Dialog content</p>
      </BaseDialog>
    );
    expect(screen.getByText("Test Dialog")).toBeInTheDocument();
    expect(screen.getByText("Dialog content")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <BaseDialog
        open={true}
        onOpenChange={() => {}}
        title="Test Dialog"
        description="This is a description"
      >
        <p>Content</p>
      </BaseDialog>
    );
    expect(screen.getByText("This is a description")).toBeInTheDocument();
  });

  it("renders footer when provided", () => {
    render(
      <BaseDialog
        open={true}
        onOpenChange={() => {}}
        title="Test"
        footer={<button>Submit</button>}
      >
        <p>Content</p>
      </BaseDialog>
    );
    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
  });

  it("does not render when closed", () => {
    render(
      <BaseDialog
        open={false}
        onOpenChange={() => {}}
        title="Hidden Dialog"
      >
        <p>Hidden content</p>
      </BaseDialog>
    );
    expect(screen.queryByText("Hidden Dialog")).not.toBeInTheDocument();
  });
});

describe("DetailsDialog", () => {
  it("renders with Close button", () => {
    render(
      <DetailsDialog
        open={true}
        onOpenChange={() => {}}
        title="Details"
      >
        <p>Details content</p>
      </DetailsDialog>
    );
    expect(screen.getByText("Details")).toBeInTheDocument();
    // There are two Close buttons (custom one and X icon), so we use getAllByRole
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    expect(closeButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("uses custom close text", () => {
    render(
      <DetailsDialog
        open={true}
        onOpenChange={() => {}}
        title="Details"
        closeText="Dismiss"
      >
        <p>Content</p>
      </DetailsDialog>
    );
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  });

  it("calls onOpenChange when Close is clicked", () => {
    const onOpenChange = vi.fn();
    render(
      <DetailsDialog
        open={true}
        onOpenChange={onOpenChange}
        title="Details"
      >
        <p>Content</p>
      </DetailsDialog>
    );
    // Get all Close buttons and click the one that's not the X icon
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    const footerCloseButton = closeButtons.find(
      (btn) => btn.textContent === "Close"
    );
    if (footerCloseButton) {
      fireEvent.click(footerCloseButton);
    }
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

describe("FormDialog", () => {
  it("renders form with Cancel and Submit buttons", () => {
    render(
      <FormDialog
        open={true}
        onOpenChange={() => {}}
        title="Create Item"
        onSubmit={() => {}}
      >
        <input placeholder="Name" />
      </FormDialog>
    );
    expect(screen.getByText("Create Item")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
  });

  it("uses custom button text", () => {
    render(
      <FormDialog
        open={true}
        onOpenChange={() => {}}
        title="Create"
        onSubmit={() => {}}
        submitText="Create"
        cancelText="Discard"
      >
        <p>Form</p>
      </FormDialog>
    );
    expect(screen.getByRole("button", { name: "Discard" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeInTheDocument();
  });

  it("shows loading state when submitting", () => {
    render(
      <FormDialog
        open={true}
        onOpenChange={() => {}}
        title="Create"
        onSubmit={() => {}}
        isSubmitting={true}
        submittingText="Creating..."
      >
        <p>Form</p>
      </FormDialog>
    );
    expect(screen.getByRole("button", { name: "Creating..." })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
  });

  it("calls onSubmit when form is submitted", () => {
    const onSubmit = vi.fn();
    render(
      <FormDialog
        open={true}
        onOpenChange={() => {}}
        title="Create"
        onSubmit={onSubmit}
      >
        <input placeholder="Name" />
      </FormDialog>
    );
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(onSubmit).toHaveBeenCalled();
  });

  it("disables submit when submitDisabled is true", () => {
    render(
      <FormDialog
        open={true}
        onOpenChange={() => {}}
        title="Create"
        onSubmit={() => {}}
        submitDisabled={true}
      >
        <p>Form</p>
      </FormDialog>
    );
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});

describe("ConfirmDialog", () => {
  it("renders confirm dialog with title and description", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={() => {}}
        title="Delete Item"
        description="Are you sure you want to delete this item?"
        onConfirm={() => {}}
      />
    );
    expect(screen.getByText("Delete Item")).toBeInTheDocument();
    expect(
      screen.getByText("Are you sure you want to delete this item?")
    ).toBeInTheDocument();
  });

  it("uses custom button text", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={() => {}}
        title="Delete"
        description="Confirm?"
        onConfirm={() => {}}
        confirmText="Delete"
        cancelText="Keep"
      />
    );
    expect(screen.getByRole("button", { name: "Keep" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("calls onConfirm when confirmed", () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={() => {}}
        title="Delete"
        description="Confirm?"
        onConfirm={onConfirm}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    expect(onConfirm).toHaveBeenCalled();
  });

  it("shows loading indicator when isLoading", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={() => {}}
        title="Delete"
        description="Confirm?"
        onConfirm={() => {}}
        isLoading={true}
      />
    );
    expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
  });

  it("applies destructive variant styles", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={() => {}}
        title="Delete"
        description="Confirm?"
        onConfirm={() => {}}
        variant="destructive"
        confirmText="Delete"
      />
    );
    const deleteButton = screen.getByRole("button", { name: "Delete" });
    expect(deleteButton).toHaveClass("bg-destructive");
  });
});
