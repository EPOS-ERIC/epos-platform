const GROUP_ICON_PATHS: Record<string, string> = {
  '1375500a-fc04-5831-a7f5-39ea5fcb0445': '/group-icons/1375500a-fc04-5831-a7f5-39ea5fcb0445.svg',
  '94a5445f-a8c4-5805-a79d-b8993738ea30': '/group-icons/94a5445f-a8c4-5805-a79d-b8993738ea30.png',
  '23b9b58a-9f1c-5d18-85f9-73842733835c': '/group-icons/23b9b58a-9f1c-5d18-85f9-73842733835c.png',
  'f3acbf1a-908b-55d4-953e-fadf1221a49e': '/group-icons/f3acbf1a-908b-55d4-953e-fadf1221a49e.png',
  'c1d952e7-f523-5fb4-a8ac-ef6c93e25316': '/group-icons/c1d952e7-f523-5fb4-a8ac-ef6c93e25316.png',
  '332c6840-2b07-58e9-9316-3d7f2123f543': '/group-icons/332c6840-2b07-58e9-9316-3d7f2123f543.png',
  'ea3f23ac-eda7-5624-b51b-6c6e4c3661db': '/group-icons/ea3f23ac-eda7-5624-b51b-6c6e4c3661db.png',
  'cc5f4f5f-340f-573b-a118-8b7fc0d7e365': '/group-icons/cc5f4f5f-340f-573b-a118-8b7fc0d7e365.png',
  '9a5cf0d0-21c2-5e21-b27c-f4359f5ee056': '/group-icons/9a5cf0d0-21c2-5e21-b27c-f4359f5ee056.png',
  '75f0157b-b1a4-5955-9079-f2993c6cc8c4': '/group-icons/75f0157b-b1a4-5955-9079-f2993c6cc8c4.png',
  'dab56e08-0b75-5d89-aa51-e3433314cbbc': '/group-icons/dab56e08-0b75-5d89-aa51-e3433314cbbc.png'
};

// Default accent (for elements like the orbit in the "manage" pages)
const DEFAULT_GROUP_ACCENT_COLOR = '#60a5fa';

export const GROUP_ACCENT_COLORS: Record<string, string> = {
  '23b9b58a-9f1c-5d18-85f9-73842733835c': '#476f57', // Seismology
  'f3acbf1a-908b-55d4-953e-fadf1221a49e': '#5e3160', // Near Fault Obs.
  '75f0157b-b1a4-5955-9079-f2993c6cc8c4': '#941d7f', // GNSS Data and Products
  '9a5cf0d0-21c2-5e21-b27c-f4359f5ee056': '#e70802', // Volcano Observations
  'cc5f4f5f-340f-573b-a118-8b7fc0d7e365': '#02adb2', // Satellite Data
  '332c6840-2b07-58e9-9316-3d7f2123f543': '#2e3486', // Geomagnetic Observations
  'c1d952e7-f523-5fb4-a8ac-ef6c93e25316': '#06a8e2', // Anthropogenic Hazards
  'dab56e08-0b75-5d89-aa51-e3433314cbbc': '#fe4b3a', // Geological Information and Modeling
  'ea3f23ac-eda7-5624-b51b-6c6e4c3661db': '#00aa65', // Multi-scale Laboratories
  '94a5445f-a8c4-5805-a79d-b8993738ea30': '#6f9ea8', // Tsunami
  '1375500a-fc04-5831-a7f5-39ea5fcb0445': '#f37726', // Jupyter Notebooks (no accent color for now, uses default)
};

export function groupIconPath(groupId: string): string | undefined {
  return GROUP_ICON_PATHS[groupId];
}

export function groupAccentColor(groupId: string): string {
  return GROUP_ACCENT_COLORS[groupId] ?? DEFAULT_GROUP_ACCENT_COLOR;
}
