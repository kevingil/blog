import { describe, expect, test } from "bun:test";
import { gaMeasurementId, installGoogleTag, trackPageView } from "./analytics";

describe("gaMeasurementId", () => {
  test("accepts a GA4 measurement id", () => {
    expect(gaMeasurementId("G-TEST1234")).toBe("G-TEST1234");
  });

  test("rejects missing and unsafe values", () => {
    expect(gaMeasurementId(undefined)).toBeUndefined();
    expect(gaMeasurementId("")).toBeUndefined();
    expect(gaMeasurementId("UA-123456")).toBeUndefined();
    expect(gaMeasurementId("G-lower")).toBeUndefined();
    expect(gaMeasurementId('G-TEST";alert(1)')).toBeUndefined();
    expect(gaMeasurementId("%VITE_GA_MEASUREMENT_ID%")).toBeUndefined();
  });
});

describe("google tag", () => {
  test("skips injection when no measurement id is configured", () => {
    const created: HTMLScriptElement[] = [];
    const head = {
      appendChild(node: HTMLScriptElement) {
        created.push(node);
      },
    };
    const previousDocument = globalThis.document;
    const previousWindow = globalThis.window;
    globalThis.document = {
      createElement: () => ({ async: false, src: "" }) as HTMLScriptElement,
      head,
    } as unknown as Document;
    globalThis.window = { location: { origin: "http://localhost" } } as Window & typeof globalThis;

    installGoogleTag(undefined);
    trackPageView("/posts", undefined);

    expect(created).toHaveLength(0);
    expect(globalThis.window.gtag).toBeUndefined();

    globalThis.document = previousDocument;
    globalThis.window = previousWindow;
  });

  test("loads gtag once and records each path once", () => {
    const created: HTMLScriptElement[] = [];
    const head = {
      appendChild(node: HTMLScriptElement) {
        created.push(node);
      },
    };
    const previousDocument = globalThis.document;
    const previousWindow = globalThis.window;
    globalThis.document = {
      title: "Kevin Gil",
      createElement: () => ({ async: false, src: "" }) as HTMLScriptElement,
      head,
    } as unknown as Document;
    globalThis.window = {
      location: { origin: "https://kevingil.com" },
    } as Window & typeof globalThis;

    installGoogleTag("G-TEST1234");
    installGoogleTag("G-TEST1234");
    trackPageView("/notes", "G-TEST1234");
    trackPageView("/notes", "G-TEST1234");
    trackPageView("/notes?page=2", "G-TEST1234");

    expect(created).toHaveLength(1);
    expect(created[0]?.src).toBe("https://www.googletagmanager.com/gtag/js?id=G-TEST1234");
    expect(created[0]?.async).toBe(true);

    const queued = globalThis.window.dataLayer ?? [];
    expect(queued.map((entry) => Array.from(entry as ArrayLike<unknown>)[0])).toEqual([
      "js",
      "config",
      "event",
      "event",
    ]);
    const pageViews = queued.filter((entry) => Array.from(entry as ArrayLike<unknown>)[0] === "event");
    expect(Array.from(pageViews[0] as ArrayLike<unknown>)[2]).toMatchObject({
      page_path: "/notes",
      page_location: "https://kevingil.com/notes",
      send_to: "G-TEST1234",
    });
    expect(Array.from(pageViews[1] as ArrayLike<unknown>)[2]).toMatchObject({
      page_path: "/notes?page=2",
    });

    globalThis.document = previousDocument;
    globalThis.window = previousWindow;
  });
});
