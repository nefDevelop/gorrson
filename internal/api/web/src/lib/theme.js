/**
 * @param {string} hex
 * @returns {[number, number, number]}
 */
function hexToHsl(hex) {
  let r = parseInt(hex.slice(1, 3), 16) / 255;
  let g = parseInt(hex.slice(3, 5), 16) / 255;
  let b = parseInt(hex.slice(5, 7), 16) / 255;
  let max = Math.max(r, g, b),
    min = Math.min(r, g, b);
  let h = 0,
    s = 0,
    l = (max + min) / 2;
  if (max === min) h = s = 0;
  else {
    let d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    switch (max) {
      case r:
        h = (g - b) / d + (g < b ? 6 : 0);
        break;
      case g:
        h = (b - r) / d + 2;
        break;
      case b:
        h = (r - g) / d + 4;
        break;
    }
    h /= 6;
  }
  return [h * 360, s * 100, l * 100];
}

/**
 * @param {number} h
 * @param {number} s
 * @param {number} l
 * @returns {string}
 */
function hslToHex(h, s, l) {
  l /= 100;
  const a = (s * Math.min(l, 1 - l)) / 100;
  /** @param {number} n */
  const f = (n) => {
    const k = (n + h / 30) % 12;
    const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
    return Math.round(255 * color)
      .toString(16)
      .padStart(2, "0");
  };
  return `#${f(0)}${f(8)}${f(4)}`;
}

/**
 * @param {string} hex
 * @param {string} mode
 */
export function generateTonal(hex, mode) {
  const [h, s, l] = hexToHsl(hex);
  const isDark = mode === "dark";

  const primary = isDark ? hslToHex(h, Math.max(s, 50), 80) : hex;
  const secondary = hslToHex((h + 25) % 360, Math.max(s - 15, 30), isDark ? 70 : 40);
  const secondaryContainer = isDark
    ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 25)
    : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90);
  const onSecondaryContainer = isDark
    ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90)
    : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 10);
  const tertiary = hslToHex((h + 80) % 360, Math.max(s, 40), isDark ? 75 : 45);

  const adwaita = {
    dark: { bg: hslToHex(h, 5, 12), surface: hslToHex(h, 6, 16), card: hslToHex(h, 7, 20), border: hslToHex(h, 10, 24) },
    light: { bg: "#FFFFFF", surface: hslToHex(h, 10, 96), card: "#FFFFFF", border: hslToHex(h, 15, 88) },
  };
  const currentA = isDark ? adwaita.dark : adwaita.light;

  return {
    primary,
    onPrimary: isDark ? hslToHex(h, s, 15) : "#FFFFFF",
    container: hslToHex(h, Math.max(s, isDark ? 60 : 70), isDark ? 25 : 92),
    onContainer: hslToHex(h, Math.max(s, isDark ? 40 : s), isDark ? 80 : 10),
    secondary,
    secondaryContainer,
    onSecondaryContainer,
    tertiary,
    surface: currentA.surface,
    onSurface: hslToHex(h, isDark ? 10 : 20, isDark ? 85 : 10),
    onSurfaceVariant: hslToHex(h, 15, isDark ? 65 : 40),
    background: currentA.bg,
    variant: currentA.card,
    outline: currentA.border,
  };
}
