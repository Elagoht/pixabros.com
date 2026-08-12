// DOM traversal from Testing Library is typed as nullable: parentElement,
// closest and querySelector can all return null. A bare `!` silences the type
// error and then fails somewhere else entirely -- "cannot read properties of
// null" a few lines down, with nothing to say which lookup came back empty.
//
// required() fails at the lookup instead, naming what was missing, so a test
// that breaks because the markup changed says so.
export function required<T>(value: T | null | undefined, what: string): T {
  if (value == null) {
    throw new Error(`expected ${what} to be present in the DOM`);
  }
  return value;
}

// parentOf is the common case: the element wrapping a node the test found by
// its text.
export function parentOf(element: Element, what: string): HTMLElement {
  return required(element.parentElement, `the ${what}`);
}
