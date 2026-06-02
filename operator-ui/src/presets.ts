export type PresetEntry = {
  presetId: string;
  title: string;
  description: string;
};

export const presetCatalog: PresetEntry[] = [
  {
    presetId: "echo-reference",
    title: "Echo Reference",
    description: "Queues the reference echo-count preset used for service validation.",
  },
];
