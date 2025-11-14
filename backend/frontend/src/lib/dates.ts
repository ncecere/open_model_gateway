const DEFAULT_TIMEZONE = "UTC";

export function formatUsageDate(
	value: string,
	timezone: string = DEFAULT_TIMEZONE,
	options?: Intl.DateTimeFormatOptions,
) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value;
	}
	const formatter = new Intl.DateTimeFormat(undefined, {
		month: "short",
		day: "numeric",
		timeZone: timezone || DEFAULT_TIMEZONE,
		...options,
	});
	return formatter.format(date);
}
