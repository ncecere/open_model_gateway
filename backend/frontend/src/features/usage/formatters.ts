type CurrencyOptions = {
  minimumFractionDigits?: number;
  maximumFractionDigits?: number;
};

export { formatTokensShort } from "@/lib/numbers";

const currencyFormatters = new Map<string, Intl.NumberFormat>();

function getCurrencyFormatter(options?: CurrencyOptions) {
  const minimumFractionDigits = options?.minimumFractionDigits ?? 2;
  const maximumFractionDigits = options?.maximumFractionDigits ?? minimumFractionDigits;
  const key = `${minimumFractionDigits}:${maximumFractionDigits}`;
  const cached = currencyFormatters.get(key);
  if (cached) {
    return cached;
  }
  const formatter = new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
    minimumFractionDigits,
    maximumFractionDigits,
  });
  currencyFormatters.set(key, formatter);
  return formatter;
}

export function formatUsageUSD(usd?: number, cents?: number, options?: CurrencyOptions) {
  const formatter = getCurrencyFormatter(options);
  const value =
    typeof usd === "number"
      ? usd
      : typeof cents === "number"
        ? cents / 100
        : 0;
  return formatter.format(value);
}

export function parseSpendValue(usd?: number, cents?: number) {
  if (typeof usd === "number" && !Number.isNaN(usd) && usd !== 0) {
    return usd;
  }
  if (typeof cents === "number") {
    return cents / 100;
  }
  return 0;
}
