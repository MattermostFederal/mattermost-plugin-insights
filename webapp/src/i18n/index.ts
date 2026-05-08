import en from './en.json';

const catalogs: Record<string, Record<string, string>> = {
    en,
};

export function getTranslationsForLocale(locale: string): Record<string, string> {
    return catalogs[locale] ?? {};
}
