export const clamp = (value: number, min: number, max: number): number =>
  Math.min(max, Math.max(min, value));

export const roundStep = (value: number, step: number): number =>
  Math.round(value / step) * step;
