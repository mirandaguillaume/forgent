export interface DebouncedFn {
  (): void;
  cancel(): void;
}

export function debounce(fn: () => void, delayMs: number): DebouncedFn {
  let timer: ReturnType<typeof setTimeout> | null = null;

  const debounced = (() => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      timer = null;
      fn();
    }, delayMs);
  }) as DebouncedFn;

  debounced.cancel = () => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
  };

  return debounced;
}
