const ENUM_LABELS: Readonly<Record<string, string>> = {
  MIN_LON: 'Minimum longitude',
  MAX_LON: 'Maximum longitude',
  MIN_LAT: 'Minimum latitude',
  MAX_LAT: 'Maximum latitude'
};

export function enumLabel(value: string | null | undefined): string {
  if (!value) return 'None';

  const configuredLabel = ENUM_LABELS[value];
  if (configuredLabel) return configuredLabel;

  const label = value.replace(/_/g, ' ').toLocaleLowerCase();
  return label.charAt(0).toLocaleUpperCase() + label.slice(1);
}
