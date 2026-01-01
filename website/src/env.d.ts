/// <reference path="../.astro/types.d.ts" />

// Extend Window interface for analytics
interface Window {
  trackEvent?: (eventName: string) => Promise<void>;
}