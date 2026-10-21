import { useCallback, useEffect, useState } from "react";
import { Center, Loader } from "@mantine/core";
import { api, SignedOut, type Device } from "./api";
import { Login } from "./Login";
import { Shell } from "./Shell";

export function App() {
  const [device, setDevice] = useState<Device | null>();
  const load = useCallback(() => {
    api.device().then(setDevice, (e) => {
      if (e instanceof SignedOut) setDevice(null);
    });
  }, []);
  useEffect(load, [load]);

  useEffect(() => {
    if (device) document.title = device.name;
  }, [device]);

  if (device === undefined) {
    return (
      <Center h="100vh">
        <Loader />
      </Center>
    );
  }
  if (device === null) return <Login onIn={load} />;
  return <Shell device={device} onOut={() => setDevice(null)} />;
}
