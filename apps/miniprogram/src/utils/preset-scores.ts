export const PRESET_SCORE_OPTIONS = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
export const DEFAULT_PRESET_SCORES = [20, 30, 40, 60];
export const MAX_PRESET_SCORES = 4;

export interface PresetScoreOption {
  value: number;
  selected: boolean;
}

export function buildPresetOptions(selected: number[]): PresetScoreOption[] {
  return PRESET_SCORE_OPTIONS.map((value) => ({ value, selected: selected.includes(value) }));
}

export function selectedPresetScores(options: PresetScoreOption[]): number[] {
  return options.filter((option) => option.selected).map((option) => option.value);
}
