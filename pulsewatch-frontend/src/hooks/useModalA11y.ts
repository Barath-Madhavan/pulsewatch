import { useEffect, useRef } from "react";

// Escape closes the modal. Focus restoration on close is deliberately NOT
// handled here via document.activeElement. By the time an effect in this
// hook could read it, the modal's own autoFocus input has usually already
// stolen it (native autofocus fires during DOM commit, before any
// useEffect/useLayoutEffect can intercept). Callers instead restore focus
// themselves via a ref on the actual trigger element, passed through their
// own onClose.
export function useModalA11y(onClose: () => void) {
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onCloseRef.current();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);
}
