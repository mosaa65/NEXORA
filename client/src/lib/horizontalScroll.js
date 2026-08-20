// Let a conventional mouse wheel navigate dense horizontal media rails while
// preserving normal vertical scrolling when the rail has no overflow.
export function horizontalWheel(event) {
  const rail = event.currentTarget;
  if (rail.scrollWidth <= rail.clientWidth || Math.abs(event.deltaY) < Math.abs(event.deltaX)) return;
  event.preventDefault();
  rail.scrollLeft += event.deltaY;
}
