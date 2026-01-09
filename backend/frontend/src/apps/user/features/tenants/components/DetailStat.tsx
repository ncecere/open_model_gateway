export type DetailStatProps = {
  label: string;
  value: string;
};

export function DetailStat({ label, value }: DetailStatProps) {
  return (
    <div>
      <p className="text-xs uppercase text-muted-foreground">{label}</p>
      <p className="font-medium capitalize">{value}</p>
    </div>
  );
}
