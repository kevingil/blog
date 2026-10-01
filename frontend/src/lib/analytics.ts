const MEASUREMENT_ID_PATTERN = /^G-[A-Z0-9]+$/;

let lastTrackedPath: string | undefined;

export function gaMeasurementId(
  value: unknown = import.meta.env.VITE_GA_MEASUREMENT_ID,
): string | undefined {
  if (typeof value !== "string" || !MEASUREMENT_ID_PATTERN.test(value)) {
    return undefined;
  }
  return value;
}

export function installGoogleTag(measurementId = gaMeasurementId()): void {
  if (!measurementId || typeof window.gtag === "function") return;

  const script = document.createElement("script");
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(measurementId)}`;
  document.head.appendChild(script);

  window.dataLayer = window.dataLayer || [];
  window.gtag = function gtag() {
    // gtag.js reads the arguments object from this queue.
    window.dataLayer?.push(arguments);
  };
  window.gtag("js", new Date());
  window.gtag("config", measurementId, { send_page_view: false });
}

export function trackPageView(pagePath: string, measurementId = gaMeasurementId()): void {
  if (!measurementId || typeof window.gtag !== "function") return;
  if (pagePath === lastTrackedPath) return;
  lastTrackedPath = pagePath;
  window.gtag("event", "page_view", {
    send_to: measurementId,
    page_path: pagePath,
    page_location: `${window.location.origin}${pagePath}`,
    page_title: document.title,
  });
}
