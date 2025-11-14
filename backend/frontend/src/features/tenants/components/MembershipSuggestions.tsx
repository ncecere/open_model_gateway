import type { AdminUser } from "@/api/users";
import { Skeleton } from "@/components/ui/skeleton";

export type MembershipSuggestionsProps = {
  query: string;
  loading: boolean;
  suggestions: AdminUser[];
  onSelect: (email: string) => void;
};

export function MembershipSuggestions({
  query,
  loading,
  suggestions,
  onSelect,
}: MembershipSuggestionsProps) {
  if (loading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-8 w-full" />
      </div>
    );
  }

  const trimmed = query.trim();
  if (!trimmed) {
    return (
      <p className="text-xs text-muted-foreground">
        Start typing to search existing users. New accounts are created
        automatically when needed.
      </p>
    );
  }

  if (suggestions.length === 0) {
    return (
      <p className="text-xs text-muted-foreground">
        No existing user matches “{query}”. A new user will be created when you
        add them.
      </p>
    );
  }

  return (
    <div className="space-y-1 rounded-md border bg-muted/40 p-2 text-sm">
      <p className="text-xs font-medium uppercase text-muted-foreground">
        Suggestions
      </p>
      {suggestions.map((user) => (
        <button
          key={user.id}
          type="button"
          className="flex w-full flex-col rounded px-2 py-1 text-left hover:bg-background"
          onClick={() => onSelect(user.email)}
        >
          <span className="font-medium text-foreground">
            {user.name || user.email}
          </span>
          <span className="text-xs text-muted-foreground">{user.email}</span>
        </button>
      ))}
    </div>
  );
}
