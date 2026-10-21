import { useEffect, useState } from "react";
import { Button, Center, Paper, PasswordInput, Stack, Text } from "@mantine/core";
import { api } from "./api";
import { Logo } from "./Logo";

export function Login({ onIn }: { onIn: () => void }) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [name, setName] = useState("");
  useEffect(() => {
    api.name().then((n) => {
      setName(n);
      document.title = n;
    }, () => {});
  }, []);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.login(password);
      onIn();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Center h="100vh" p="md">
      <Paper w={340} p="xl" withBorder>
        <form onSubmit={submit}>
          <Stack gap="md">
            <Stack gap={6} align="center">
              <Logo h={72} />
              {name && (
                <Text fw={600} size="lg">
                  {name}
                </Text>
              )}
            </Stack>
            <PasswordInput
              label="Admin password"
              value={password}
              onChange={(e) => setPassword(e.currentTarget.value)}
              autoFocus
            />
            {error && (
              <Text c="red" size="sm">
                {error}
              </Text>
            )}
            <Button type="submit" loading={busy} fullWidth>
              Sign in
            </Button>
          </Stack>
        </form>
      </Paper>
    </Center>
  );
}
