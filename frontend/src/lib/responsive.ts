/**
 * Responsive Design Breakpoints
 *
 * Tailwind CSS breakpoints (default):
 * - sm: 640px (small devices)
 * - md: 768px (tablets)
 * - lg: 1024px (desktops)
 * - xl: 1280px (large desktops)
 * - 2xl: 1536px (extra large desktops)
 *
 * Mobile breakpoint: < 768px
 * Tablet breakpoint: 768px - 1023px
 * Desktop breakpoint: >= 1024px
 *
 * Touch-friendly minimum sizes:
 * - Buttons: minimum 44x44px (h-11)
 * - Touch targets: minimum 44x44px
 * - Form inputs: minimum 44px height
 */

export const BREAKPOINTS = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  '2xl': 1536,
} as const;

export const TOUCH_TARGET_MIN_SIZE = 44; // pixels
