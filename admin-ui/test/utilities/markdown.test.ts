import { describe, expect, it } from "vitest";
import { renderMarkdown } from "@/utilities/markdown";

describe("renderMarkdown", () => {
  it("renders common markdown", () => {
    const html = renderMarkdown("## Heading\n\nSome **bold** text.");

    expect(html).toContain("<h2");
    expect(html).toContain("<strong>bold</strong>");
  });

  it("renders lists", () => {
    const html = renderMarkdown("- one\n- two");

    expect(html).toContain("<ul>");
    expect(html).toContain("<li>one</li>");
  });

  it("renders links", () => {
    const html = renderMarkdown("[itch](https://example.itch.io)");

    expect(html).toContain('href="https://example.itch.io"');
  });

  // The admin writes this markdown, so the sanitiser is not guarding against a
  // hostile author -- it guards against a paste. Markdown lets raw HTML
  // through, so a snippet copied off a web page could otherwise carry a script
  // into the preview and run it inside the panel.
  it("strips script tags pasted into the source", () => {
    const html = renderMarkdown("Hello <script>alert('xss')</script> world");

    expect(html).not.toContain("<script");
    expect(html).not.toContain("alert(");
    expect(html).toContain("Hello");
  });

  it("strips inline event handlers", () => {
    const html = renderMarkdown('<img src="x" onerror="alert(1)">');

    expect(html).not.toContain("onerror");
  });

  it("strips javascript: URLs", () => {
    const html = renderMarkdown("[click](javascript:alert(1))");

    expect(html).not.toContain("javascript:");
  });

  it("returns an empty string for empty input", () => {
    expect(renderMarkdown("")).toBe("");
  });
});
