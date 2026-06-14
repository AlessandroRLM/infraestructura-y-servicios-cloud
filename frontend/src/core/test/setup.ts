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
}
