import DOMPurify from "dompurify";
import { marked } from "marked";

// The admin writes this markdown themselves, so the sanitiser is not guarding
// against a hostile author. It guards against a paste: markdown allows raw
// HTML through, so a snippet copied from a web page could carry a script tag
// into the preview and run it inside the panel.
export const renderMarkdown = (source: string): string => {
  const html = marked.parse(source, { async: false, gfm: true, breaks: true });
  return DOMPurify.sanitize(html);
};
