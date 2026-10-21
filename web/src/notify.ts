type Listener = (message: string) => void;

const listeners = new Set<Listener>();

export const notifications = {
  error: (message: string) => listeners.forEach((l) => l(message)),
  listen: (l: Listener) => {
    listeners.add(l);
    return () => {
      listeners.delete(l);
    };
  },
};
