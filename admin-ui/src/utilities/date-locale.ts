export const DAYS_EN = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"];
export const DAYS_TR = ["Pzr", "Pzt", "Sal", "Çar", "Per", "Cum", "Cmt"];

export const MONTHS_EN = [
	"January",
	"February",
	"March",
	"April",
	"May",
	"June",
	"July",
	"August",
	"September",
	"October",
	"November",
	"December",
];

export const MONTHS_TR = [
	"Ocak",
	"Şubat",
	"Mart",
	"Nisan",
	"Mayıs",
	"Haziran",
	"Temmuz",
	"Ağustos",
	"Eylül",
	"Ekim",
	"Kasım",
	"Aralık",
];

export const localeData: Record<string, { days: string[]; months: string[] }> =
	{
		en: { days: DAYS_EN, months: MONTHS_EN },
		tr: { days: DAYS_TR, months: MONTHS_TR },
	};

export const getDaysInMonth = (y: number, m: number): number =>
	new Date(y, m + 1, 0).getDate();

export const getFirstDay = (y: number, m: number): number =>
	new Date(y, m, 1).getDay();

export const formatDateDisplay = (dateStr: string, locale: Locale): string => {
	const d = new Date(`${dateStr}T00:00:00`);
	return d.toLocaleDateString(locale === "tr" ? "tr-TR" : "en-US", {
		year: "numeric",
		month: "long",
		day: "numeric",
	});
};
