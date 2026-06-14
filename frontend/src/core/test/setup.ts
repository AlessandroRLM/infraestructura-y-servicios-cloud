import "@testing-library/jest-dom";

// Radix UI Select/Tooltip/Popover rely on Pointer Events APIs not implemented
// in happy-dom. These minimal stubs prevent "not a function" crashes in tests.
if (typeof window !== "undefined") {
  if (!window.HTMLElement.prototype.hasPointerCapture) {
    window.HTMLElement.prototype.hasPointerCapture = () => false;
  }
  if (!window.HTMLElement.prototype.releasePointerCapture) {
    window.HTMLElement.prototype.releasePointerCapture = () => undefined;
  }
  if (!window.HTMLElement.prototype.setPointerCapture) {
    window.HTMLElement.prototype.setPointerCapture = () => undefined;
  }
  // Radix Select calls scrollIntoView on highlighted items; happy-dom does not implement it.
  if (!Element.prototype.scrollIntoView) {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional no-op polyfill
    Element.prototype.scrollIntoView = () => {};
  }
}
