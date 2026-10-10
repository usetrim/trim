"use client";

import { useCallback, useEffect, useRef, useState } from "react";

const DIALOG_CLEAR_MS = 200;

export function useDeferredDialogSelection(clearMs = DIALOG_CLEAR_MS) {
  const [open, setOpen] = useState(false);
  const [selectedId, setSelectedId] = useState("");
  const clearTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancelClear = useCallback(() => {
    if (clearTimer.current) {
      clearTimeout(clearTimer.current);
      clearTimer.current = null;
    }
  }, []);

  const openWith = useCallback(
    (id: string) => {
      const next = id.trim();
      if (!next) return;
      cancelClear();
      setSelectedId(next);
      setOpen(true);
    },
    [cancelClear],
  );

  const close = useCallback(() => {
    setOpen(false);
    cancelClear();
    clearTimer.current = setTimeout(() => {
      setSelectedId("");
      clearTimer.current = null;
    }, clearMs);
  }, [cancelClear, clearMs]);

  const onOpenChange = useCallback(
    (next: boolean) => {
      if (next) {
        cancelClear();
        setOpen(true);
        return;
      }
      close();
    },
    [cancelClear, close],
  );

  useEffect(() => () => cancelClear(), [cancelClear]);

  return { open, selectedId, openWith, close, onOpenChange, setSelectedId };
}

/**
 * Controlled dialog open state keyed by an arbitrary value (object/row).
 * Keeps the value until the close animation finishes (same glitch class as id selection).
 */
export function useDeferredDialogValue<T>(clearMs = DIALOG_CLEAR_MS) {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState<T | null>(null);
  const clearTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancelClear = useCallback(() => {
    if (clearTimer.current) {
      clearTimeout(clearTimer.current);
      clearTimer.current = null;
    }
  }, []);

  const openWith = useCallback(
    (next: T) => {
      cancelClear();
      setValue(next);
      setOpen(true);
    },
    [cancelClear],
  );

  const close = useCallback(() => {
    setOpen(false);
    cancelClear();
    clearTimer.current = setTimeout(() => {
      setValue(null);
      clearTimer.current = null;
    }, clearMs);
  }, [cancelClear, clearMs]);

  const onOpenChange = useCallback(
    (next: boolean) => {
      if (next) {
        cancelClear();
        setOpen(true);
        return;
      }
      close();
    },
    [cancelClear, close],
  );

  useEffect(() => () => cancelClear(), [cancelClear]);

  return { open, value, openWith, close, onOpenChange, setValue };
}

export function releaseBodyPointerEvents() {
  if (typeof document === "undefined") return;
  document.body.style.pointerEvents = "";
}
