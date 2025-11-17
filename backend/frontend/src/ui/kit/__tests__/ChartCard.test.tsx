import { render, screen } from "@testing-library/react";

import { ChartCard } from "../ChartCard";

describe("ChartCard", () => {
  it("renders title, description, and children", () => {
    render(
      <ChartCard title="Top tenants" description="Usage" isLoading={false}>
        <div>Chart body</div>
      </ChartCard>,
    );

    expect(screen.getByText("Top tenants")).toBeInTheDocument();
    expect(screen.getByText("Usage")).toBeInTheDocument();
    expect(screen.getByText("Chart body")).toBeInTheDocument();
  });

  it("shows a skeleton when loading", () => {
    render(
      <ChartCard title="Loading" isLoading>
        <div>Hidden</div>
      </ChartCard>,
    );

    expect(screen.queryByText("Hidden")).not.toBeInTheDocument();
    expect(screen.getByTestId("chartcard-skeleton")).toBeInTheDocument();
  });
});
