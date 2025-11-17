import { render, screen } from "@testing-library/react";

import { DataTable, type DataTableColumn } from "../DataTable";

describe("DataTable", () => {
  type Row = { id: string; name: string; status: string };
  const columns: DataTableColumn<Row>[] = [
    { header: "Name", cell: (row) => row.name },
    { header: "Status", cell: (row) => row.status },
  ];

  it("renders rows when data is provided", () => {
    const rows: Row[] = [
      { id: "1", name: "Tenant A", status: "active" },
      { id: "2", name: "Tenant B", status: "suspended" },
    ];

    render(
      <DataTable
        data={rows}
        columns={columns}
        getKey={(row) => row.id}
        emptyState="No rows"
      />,
    );

    expect(screen.getByText("Tenant A")).toBeInTheDocument();
    expect(screen.getByText("Tenant B")).toBeInTheDocument();
    expect(screen.queryByText("No rows")).not.toBeInTheDocument();
  });

  it("shows the empty state when no data is present", () => {
    render(
      <DataTable
        data={[]}
        columns={columns}
        getKey={(row) => row.id}
        emptyState="Nothing here"
      />,
    );

    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });
});
