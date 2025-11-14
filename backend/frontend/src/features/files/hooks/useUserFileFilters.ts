import { useMemo, useState } from "react";
import type { UserFileRecord } from "@/api/user/files";

export function useUserFileFilters(files: UserFileRecord[]) {
  const [searchTerm, setSearchTerm] = useState("");
  const [purposeFilter, setPurposeFilter] = useState("all");

  const purposeOptions = useMemo(() => {
    const set = new Set<string>();
    files.forEach((file) => file.purpose && set.add(file.purpose));
    return Array.from(set).sort();
  }, [files]);

  const filteredFiles = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return files.filter((file) => {
      const matchesTerm =
        !term ||
        file.filename.toLowerCase().includes(term) ||
        file.purpose?.toLowerCase().includes(term);
      const matchesPurpose = purposeFilter === "all" || file.purpose === purposeFilter;
      return matchesTerm && matchesPurpose;
    });
  }, [files, searchTerm, purposeFilter]);

  return {
    searchTerm,
    setSearchTerm,
    purposeFilter,
    setPurposeFilter,
    purposeOptions,
    filteredFiles,
  };
}
